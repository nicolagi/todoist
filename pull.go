package todoist

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
)

// pullResponse partially represents the JSON response from the Sync API.  I've only added what I actually need,
// the actual response is far richer.
type pullResponse struct {
	// This token is used for synchronization. It correlates one sync API call with the next. We will only receive
	// entities that have changed since the previous time (according to this token) we have called the sync API.
	SyncToken string `json:"sync_token"`

	Labels   []*Label   `json:"labels"`
	Projects []*Project `json:"projects"`
	Items    []*Item    `json:"items"`
	Notes    []*Note    `json:"notes"`
}

// Pull makes a sync API call to get everything that changed since the last time it was called, and updates the
// client's in-memory data. This is used to sync back changes initiated by the client (first enqueueing commands,
// e.g., with QueueItemAdd, and then pushing them with Push) or to sync back changes initiated by other apps (e.g.,
// items added from a mobile phone). To reduce API calls, if this client hasn't pushed any commands since the last
// pull, and the client already pulled once in the last minute, this method won't do anything.
func (c *Client) Pull() error {
	// Avoid pulling too often. The timestamp is set by this method on successful update, but can be set by
	// the push method too in order to signal that we need to pull the changes down.
	if time.Since(c.lastPulled) <= time.Minute {
		return nil
	}
	data := make(url.Values)
	data.Set("token", c.token)
	data.Set("sync_token", c.data.SyncToken)
	data.Set("resource_types", `["items","labels","notes","projects"]`)
	r, err := http.PostForm(c.endpoint, data)
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			log.WithFields(log.Fields{
				"op":    "pull",
				"cause": err,
			}).Warning("Could not close request body")
		}
	}()
	switch r.StatusCode {
	case http.StatusOK:
		var pr *pullResponse
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("pull, read body: %w", err)
		}
		_, _ = c.wlog.Write([]byte(`{"type": "response", "response": `))
		_, _ = c.wlog.Write(b)
		_, _ = c.wlog.Write([]byte("}\n"))
		err = json.Unmarshal(b, &pr)
		if err != nil {
			return fmt.Errorf("pull, unmarshal: %w", err)
		}
		c.data.SyncToken = pr.SyncToken

		for _, item := range pr.Items {
			c.updateItem(item)
		}
		for _, project := range pr.Projects {
			c.updateProject(project)
		}
		for _, label := range pr.Labels {
			c.updateLabel(label)
		}
		for _, note := range pr.Notes {
			c.updateNote(note)
		}
		c.lastPulled = time.Now()
		return nil
	default:
		var responseText string
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			responseText = fmt.Sprintf("unknown, because of error: %v", err)
		} else {
			responseText = string(b)
		}
		log.WithFields(log.Fields{
			"op":   "pull",
			"code": r.StatusCode,
			"text": responseText,
		}).Error("Unhandled response")
		return fmt.Errorf("%d: %w", r.StatusCode, ErrStatusCode)
	}
}
