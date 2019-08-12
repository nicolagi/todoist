package todoist

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"time"
)

// ErrCorrupted can be returned by Load.
var ErrCorrupted = errors.New("local data is corrupted")

type clientOption func(*Client) error

// WithEndpoint is a client option to set the endpoint when building a client with NewClient. This is meant to be
// used in tests only.
func WithEndpoint(endpoint string) clientOption {
	return func(c *Client) error {
		c.endpoint = endpoint
		return nil
	}
}

// WithWireLog is a client option to be passed to NewClient in order to log all requests and responses to the
// specified log file. Useful for debugging the client itself, shouldn't be needed in normal operation.
func WithWireLog(pathname string) clientOption {
	return func(c *Client) error {
		f, err := os.OpenFile(pathname, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err == nil {
			c.wlog = f
		}
		return err
	}
}

// clientData is quite similar to pull response, only it maintains maps instead of slices.
// This is what the client will persist (Dump, Load).
type clientData struct {
	// This token is used for synchronization. It correlates one sync API call with the next. We will only receive
	// entities that have changed since the previous time (according to this token) we have called the sync API.
	SyncToken string `json:"sync_token"`

	Labels   map[int64]*Label   `json:"labels"`
	Projects map[int64]*Project `json:"projects"`
	Items    map[int64]*Item    `json:"items"`
	Notes    map[int64]*Note    `json:"notes"`
}

// Client is a Todoist Sync API client, for the v8 API version. For more documentation on the API see
// https://developer.todoist.com/sync/v8/.
type Client struct {
	endpoint string

	// The secret token to authenticate and authorize API calls.
	token string

	// If non-nil, log all requests and responses to this file, one per line, in JSON format.
	wlog io.Writer

	// Represents our cached contents.
	data *clientData

	// Temporary id (client-generated UUID) to permanent id (server-generated int64).
	t2p map[string]int64

	// Commands, such as changing an item's content, are queued here and flushed when the Push() method is called.
	commands []*command

	lastPulled time.Time
}

// NewClient creates a new client authenticated and authorized by the given token.
func NewClient(token string, opts ...clientOption) (*Client, error) {
	var data clientData
	data.SyncToken = "*"
	data.Labels = make(map[int64]*Label)
	data.Projects = make(map[int64]*Project)
	data.Items = make(map[int64]*Item)
	data.Notes = make(map[int64]*Note)
	c := &Client{
		endpoint: "https://api.todoist.com/sync/v8/sync",
		token:    token,
		data:     &data,
		t2p:      make(map[string]int64),
		wlog:     ioutil.Discard,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Load loads the client state from the state files in lib/todoist/ in the user's home directory.
func (c *Client) Load() error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	data, err := ioutil.ReadFile(path.Join(u.HomeDir, "lib/todoist/state.data"))
	if err != nil {
		return err
	}
	savedSum, err := ioutil.ReadFile(path.Join(u.HomeDir, "lib/todoist/state.sum"))
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	if len(savedSum) != len(sum) {
		return fmt.Errorf("length mismatch: %w", ErrCorrupted)
	}
	for i := 0; i < len(sum); i++ {
		if savedSum[i] != sum[i] {
			return fmt.Errorf("checksum difference at byte %d: %w", i, ErrCorrupted)
		}
	}
	var loaded clientData
	err = json.Unmarshal(data, &loaded)
	if err == nil {
		c.data = &loaded
	}
	return err
}

// Dump saves the client's in-memory state to a pair of files in the lib/todoist/ directory within the user's
// home directory.  The counterpart method to load the state is Load. This dump and load mechanism is present to
// avoid  full syncs and do incremental syncs only, see https://developer.todoist.com/sync/v8/#sync for details. All
// clients use the same state files, so state can be overridden if using more than one instance of the client.
func (c *Client) Dump() error {
	data, err := json.Marshal(c.data)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	u, err := user.Current()
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(u.HomeDir, "lib/todoist/state.data"), data, 0600); err != nil {
		return err
	}
	if err := ioutil.WriteFile(path.Join(u.HomeDir, "lib/todoist/state.sum"), sum[:], 0600); err != nil {
		return err
	}
	return nil
}

// ItemByID looks up the item by id in the client's data (no remote call is made). The item should be treated as
// read-only. To update an item's property, the workflow is to enqueue commands to update the item, e.g., using
// ItemPatch and QueueItemUpdate, then Push the commands to the servers, and finally Pull() the updated state.
func (c *Client) ItemByID(id int64) (*Item, bool) {
	i, ok := c.data.Items[id]
	return i, ok
}

// ProjectByID is analogous to ItemByID.
func (c *Client) ProjectByID(id int64) (*Project, bool) {
	p, ok := c.data.Projects[id]
	return p, ok
}

// LabelByID is analogous to ItemByID.
func (c *Client) LabelByID(id int64) (*Label, bool) {
	l, ok := c.data.Labels[id]
	return l, ok
}

// LabelByName is analogous to ItemByID.
func (c *Client) LabelByName(name string) *Label {
	for _, l := range c.data.Labels {
		if l.Name == name {
			return l
		}
	}
	return nil
}

// NoteByID is analogous to ItemByID.
func (c *Client) NoteByID(id int64) (*Note, bool) {
	n, ok := c.data.Notes[id]
	return n, ok
}

// PermanentID looks up the permanent id corresponding to a given temporary id. Temporary ids are UUIDs assigned
// by the client when creating resources such as items and projects via commands. When those commands are pushed to
// the servers, to each temporary id is assigned a unique id by the server, which we're calling here permanent id,
// and this mapping is returned in the response to the push API call. The client maintains such mapping and uses
// it to implement this method. See https://developer.todoist.com/sync/v8/#sync for details.
func (c *Client) PermanentID(temporaryID string) (permanentID int64, found bool) {
	permanentID, found = c.t2p[temporaryID]
	return
}

func (c *Client) updateItem(current *Item) {
	stale, ok := c.ItemByID(current.ID)
	if ok {
		*stale = *current
	} else {
		c.data.Items[current.ID] = current
	}
}

func (c *Client) updateProject(current *Project) {
	stale, ok := c.ProjectByID(current.ID)
	if ok {
		*stale = *current
	} else {
		c.data.Projects[current.ID] = current
	}
}

func (c *Client) updateLabel(current *Label) {
	stale, ok := c.LabelByID(current.ID)
	if ok {
		*stale = *current
	} else {
		c.data.Labels[current.ID] = current
	}
}

func (c *Client) updateNote(current *Note) {
	stale, ok := c.NoteByID(current.ID)
	if ok {
		*stale = *current
	} else {
		c.data.Notes[current.ID] = current
	}
}
