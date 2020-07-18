package todoist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// Note holds a subset of attributes of a Todoist note. Treat as read-only, use NotePatch for add/update commands.
type Note struct {
	ID        int64  `json:"id"`
	ItemID    int64  `json:"item_id"`
	ProjectID int64  `json:"project_id"`
	Content   string `json:"content"`
	IsDeleted int    `json:"is_deleted"`
	Posted    string `json:"posted"`

	time time.Time
}

func (note *Note) Time() time.Time {
	if !note.time.IsZero() {
		return note.time
	}
	note.time, _ = time.Parse(time.RFC3339, note.Posted)
	return note.time
}

type NotePatch struct {
	id    int64
	attrs map[string]string
	err   error
}

func NewNotePatch(id int64) *NotePatch {
	var note NotePatch
	note.id = id
	note.attrs = make(map[string]string)
	return &note
}

func (note *NotePatch) WithItemID(value ID) *NotePatch {
	if note.err != nil {
		return note
	}
	b, err := json.Marshal(value)
	if err != nil {
		note.err = fmt.Errorf("setting item id: %w", err)
	} else {
		note.attrs["item_id"] = string(b)
	}
	return note
}

func (note *NotePatch) WithContent(value string) *NotePatch {
	if note.err != nil {
		return note
	}
	note.attrs["content"] = fmt.Sprintf("%q", value)
	return note
}

func (note *NotePatch) Empty() bool {
	return len(note.attrs["content"]) == 0
}

// MarshalJSON implements json.Marshaler.
func (note *NotePatch) MarshalJSON() ([]byte, error) {
	if note.err != nil {
		return nil, note.err
	}
	buf := bytes.NewBuffer(nil)
	_, _ = fmt.Fprintf(buf, `{"id":%d`, note.id)
	for k, v := range note.attrs {
		_, _ = fmt.Fprintf(buf, `,%q:%s`, k, v)
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}
