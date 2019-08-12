package todoist

import (
	"bytes"
	"fmt"
)

// Label partially describes a label. (It only includes a subset of the fields available in Todoist.) Treat as
// read-only.
type Label struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ItemOrder int    `json:"item_order"`
	IsDeleted int    `json:"is_deleted"`
}

// LabelPatch is used to add or update labels (see, e.g., QueueLabelAdd, QueueLabelUpdate).
type LabelPatch struct {
	id    int64
	attrs map[string]string
}

func NewLabelPatch(id int64) *LabelPatch {
	label := new(LabelPatch)
	label.id = id
	label.attrs = make(map[string]string)
	return label
}

func (label *LabelPatch) WithName(value string) *LabelPatch {
	label.attrs["name"] = fmt.Sprintf("%q", value)
	return label
}

// MarshalJSON implements json.Marshaler.
func (label *LabelPatch) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	_, _ = fmt.Fprintf(buf, `{"id":%d`, label.id)
	for k, v := range label.attrs {
		_, _ = fmt.Fprintf(buf, `,%q:%s`, k, v)
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}
