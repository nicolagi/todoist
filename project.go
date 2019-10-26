package todoist

import (
	"bytes"
	"fmt"
	"strconv"
)

// Project partially describes a project.  It only includes a subset of the fields available in Todoist.  It must
// only be used to parse API responses.  Use ProjectPatch for adding or updating a project (see also QueueProjectAdd,
// QueueProjectUpdate).
type Project struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	ChildOrder int    `json:"child_order"`
	IsDeleted  int    `json:"is_deleted"`
	IsArchived int    `json:"is_archived"`
}

// ProjectPatch holds a subset of attributes for a new or existing Todoist project, meant for an add or update command.
type ProjectPatch struct {
	id    int64
	attrs map[string]string
}

func NewProjectPatch(id int64) *ProjectPatch {
	var project ProjectPatch
	project.id = id
	project.attrs = make(map[string]string)
	return &project
}

func (project *ProjectPatch) WithName(value string) *ProjectPatch {
	project.attrs["name"] = fmt.Sprintf("%q", value)
	return project
}

func (project *ProjectPatch) WithColor(value int) *ProjectPatch {
	project.attrs["color"] = strconv.Itoa(value)
	return project
}

func (project *ProjectPatch) WithChildOrder(value int) *ProjectPatch {
	project.attrs["child_order"] = strconv.Itoa(value)
	return project
}

// MarshalJSON implements json.Marshaler.
func (project *ProjectPatch) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	_, _ = fmt.Fprintf(buf, `{"id":%d`, project.id)
	for k, v := range project.attrs {
		_, _ = fmt.Fprintf(buf, `,%q:%s`, k, v)
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}
