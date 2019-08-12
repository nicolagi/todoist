package todoist

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
)

// Item partially describes an item in Todoist. It only includes a subset of the fields.  It is used to deserialize
// new or updated items in the response to Pull and should be treated as read-only. Mutating client methods use
// different types, e.g., ItemPatch.
type Item struct {
	ID         int64   `json:"id"`
	ProjectID  int64   `json:"project_id"`
	Labels     []int64 `json:"labels"`
	Content    string  `json:"content"`
	ChildOrder int     `json:"child_order"`
	Checked    int     `json:"checked"`
	IsDeleted  int     `json:"is_deleted"`
	Due        *Due    `json:"due"`
}

// ItemPatch describes an update to an item object. (The setter methods With* might incur an error, which will
// surface when marshalling to JSON. Since serializing to JSON and using as a command is the only intended usage
// of this type, the approach seems fine.)
type ItemPatch struct {
	id    int64
	attrs map[string]string
	err   error // If an error occurred in any of the .With* methods.
}

func NewItemPatch(id int64) *ItemPatch {
	var item ItemPatch
	item.id = id
	item.attrs = make(map[string]string)
	return &item
}

func (item *ItemPatch) WithProjectID(value int64) *ItemPatch {
	if item.err != nil {
		return item
	}
	item.attrs["project_id"] = strconv.FormatInt(value, 10)
	return item
}

func (item *ItemPatch) WithContent(value string) *ItemPatch {
	if item.err != nil {
		return item
	}
	item.attrs["content"] = fmt.Sprintf("%q", value)
	return item
}

// WithLabels marks the item's labels property to be updated to the given value. Note that it takes arguments of
// type ID.  That means temporary ids can be used, e.g., one can create a label only locally with QueueLabelAdd,
// and reference it here using its temporary ID, and then push both commands at once with Push.
func (item *ItemPatch) WithLabels(value ...ID) *ItemPatch {
	if item.err != nil {
		return item
	}
	if len(value) == 0 {
		item.attrs["labels"] = "[]"
		return item
	}
	b, err := json.Marshal(value)
	if err != nil {
		item.err = fmt.Errorf("setting labels: %w", err)
	} else {
		item.attrs["labels"] = string(b)
	}
	return item
}

// WithDue sets a due date or date and time, using the RFC3339 format, i.e., in the form 2019-08-07 or
// 2019-08-07T21:20:34Z.
func (item *ItemPatch) WithDue(date string) *ItemPatch {
	item.attrs["due"] = fmt.Sprintf(`{"date":%q}`, date)
	return item
}

// MarshalJSON implements json.Marshaler.
func (item *ItemPatch) MarshalJSON() ([]byte, error) {
	if item.err != nil {
		return nil, item.err
	}
	buf := bytes.NewBuffer(nil)
	_, _ = fmt.Fprintf(buf, `{"id":%d`, item.id)
	for k, v := range item.attrs {
		_, _ = fmt.Fprintf(buf, `,%q:%s`, k, v)
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}
