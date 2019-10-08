package todoist

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

// Error is one of the two possible responses for a Todoist command.
// The other is an "ok" string.
type Error struct {
	Code    int    `json:"error_code"`
	Message string `json:"error"`
}

// Error implements error.
func (e *Error) Error() string {
	return fmt.Sprintf("%s (%d)", e.Message, e.Code)
}

type commandStatus struct {
	err *Error
}

func (status commandStatus) Err() error {
	// If this looks odd:
	// https://golang.org/doc/faq#nil_error
	if status.err != nil {
		return status.err
	}
	return nil
}

// The byte slice is either a string "ok", in case of a successful command, or another complex type to represent
// the command error.
func (status *commandStatus) UnmarshalJSON(b []byte) error {
	if !bytes.Equal(b, []byte(`"ok"`)) {
		if e := json.Unmarshal(b, &status.err); e != nil {
			return e
		}
	}
	return nil
}

// ErrStatusCode is returned in case the response from the API contains a status code that the client can't handle.
var ErrStatusCode = errors.New("unhandled status code")

type pushResponse struct {
	// Map keys are UUIDs corresponding to commands.
	SyncStatus map[string]*commandStatus `json:"sync_status"`

	// Maps each temporary UUID to its permanent id.
	TempIDMapping map[string]int64 `json:"temp_id_mapping"`
}

func (r *pushResponse) Err() error {
	var b bytes.Buffer
	for command, status := range r.SyncStatus {
		if err := status.Err(); err != nil {
			fmt.Fprintf(&b, "%v: %v\n", command, err)
		}
	}
	if b.Len() > 0 {
		return errors.New(b.String())
	}
	return nil
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
