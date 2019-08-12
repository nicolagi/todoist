package todoist

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

// ErrStatusCode is returned in case the response from the API contains a status code that the client can't handle.
var ErrStatusCode = errors.New("unhandled status code")

type pushResponse struct {
	// Map keys are UUIDs corresponding to commands.  The map value is an empty interface because the value
	// type is either a string "ok", in case of a successful command, or another complex type to represent
	// the command error.  Use the Err() method to extract the errors from this field.
	SyncStatus map[string]interface{} `json:"sync_status"`

	// Maps each temporary UUID to its permanent id.
	TempIDMapping map[string]int64 `json:"temp_id_mapping"`

	// Lazily populated from the sync status.
	err error
}

func (r *pushResponse) Err() error {
	// We set the sync status to nil to signal that we've already computed the error.
	if r.SyncStatus == nil {
		return r.err
	}
	for command, status := range r.SyncStatus {
		// Assume that if it's a string, it means the command went well, per API documentation.
		if _, ok := status.(string); ok {
			delete(r.SyncStatus, command)
		}
	}
	if len(r.SyncStatus) != 0 {
		b, err := json.Marshal(r.SyncStatus)
		if err != nil {
			r.err = fmt.Errorf("could not re-marshal errors: %w", err)
		} else {
			r.err = errors.New(string(b))
		}
	}
	r.SyncStatus = nil
	return r.err
}

// Push flushes all queued commands. It will return an error if any of them is not successful. If more than one
// command returns an error, those errors will all be reported as one. Other than the temporary to permanent id
// mapping (see PermanentID), internal state is not updated after the push, so one should call Pull for that.
//
// Note for possible future changes. We could avoid pulling back our changes in principle, by already doing the
// changes in the client's in-memory data using the temporary IDs, and only updating such ids after the push,
// but that would require more implementation work in the client. Since one still has to call Pull periodically to
// incorporate changes done in other clients (e.g., mobile phone) all the same, I'm sticking with pull-after-push
// for now.
func (c *Client) Push() error {
	data := make(url.Values)
	data.Set("token", c.token)
	b, err := json.Marshal(c.commands)
	if err != nil {
		return err
	}
	_, _ = c.wlog.Write([]byte(`{"type": "commands", "commands": `))
	_, _ = c.wlog.Write(b)
	_, _ = c.wlog.Write([]byte("}\n"))
	data.Set("commands", string(b))
	r, err := http.PostForm(c.endpoint, data)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.WithFields(log.Fields{
				"op":    "push",
				"cause": err,
			}).Warning("Could not close request body")
		}
	}()
	switch r.StatusCode {
	case http.StatusOK:
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil
		}
		_, _ = c.wlog.Write([]byte(`{"type": "response", "response": `))
		_, _ = c.wlog.Write(b)
		_, _ = c.wlog.Write([]byte("}\n"))
		var pr pushResponse
		err = json.Unmarshal(b, &pr)
		if err != nil {
			return fmt.Errorf("push, unmarshal: %w", err)
		}
		if err := pr.Err(); err != nil {
			return err
		}
		for tid, pid := range pr.TempIDMapping {
			c.t2p[tid] = pid
		}
		c.commands = nil
		c.lastPulled = time.Time{}
		return nil
	default:
		var responseText string
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			responseText = fmt.Sprintf("unknown, because of error reading body: %v", err)
		} else {
			responseText = string(b)
		}
		// This log line should be superfluous, because the caller should handle the error.
		// Possibly logging; only the outermost layer should log.
		log.WithFields(log.Fields{
			"op":   "push",
			"code": r.StatusCode,
			"text": responseText,
		}).Error("Unhandled response status code")
		return fmt.Errorf("%d: %w", r.StatusCode, ErrStatusCode)
	}
}
