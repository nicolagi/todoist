package todoist

import (
	"bytes"
	"encoding/json"
	"fmt"

	uuid "github.com/nu7hatch/gouuid"
)

// These constants are among the possible values for the type property of a command.
const (
	itemAdd     = "item_add"
	itemUpdate  = "item_update"
	itemDelete  = "item_delete"
	itemClose   = "item_close"
	itemMove    = "item_move"
	itemReorder = "item_reorder"

	labelAdd    = "label_add"
	labelUpdate = "label_update"
	labelDelete = "label_delete"

	projectAdd     = "project_add"
	projectUpdate  = "project_update"
	projectDelete  = "project_delete"
	projectArchive = "project_archive"
	projectReorder = "project_reorder"

	noteAdd    = "note_add"
	noteUpdate = "note_update"
	noteDelete = "note_delete"
)

// entityOrderAssignment can be used for items and projects alike.
type entityOrderAssignment struct {
	ID         int64 `json:"id"`
	ChildOrder int   `json:"child_order"`
}

// ReorderCommand is for reordering both projects and items.
type ReorderCommand struct {
	entity string
	args   []entityOrderAssignment
}

func (reorder *ReorderCommand) Add(id int64, childOrder int) {
	reorder.args = append(reorder.args, entityOrderAssignment{
		ID:         id,
		ChildOrder: childOrder,
	})
}

func (reorder *ReorderCommand) Empty() bool {
	return len(reorder.args) == 0
}

type idContainer struct {
	ID int64 `json:"id"`
}

// MarshalJSON implements json.Marshaler.
func (reorder *ReorderCommand) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(reorder.args)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	_, _ = fmt.Fprintf(buf, "{%q:", reorder.entity)
	buf.Write(b)
	buf.WriteRune('}')
	return buf.Bytes(), nil
}

// command represents a Todoist command according to the Sync API documentation.
type command struct {
	Type string `json:"type"`

	// Needed only when adding entities.
	TempID string `json:"temp_id,omitempty"`

	// Identifies the command for idempotency and to get its response from the response for a batch of commands.
	UUID string `json:"uuid"`

	// This can be a project patch, an item patch, a label patch, a project reorder object, etc.
	// commands
	// It can be many more things but they're not implemented in this package.
	Args interface{} `json:"args"`
}

func newCommand(cmdType string, args interface{}) *command {
	u, _ := uuid.NewV4()
	c := &command{Type: cmdType, UUID: u.String(), Args: args}
	switch cmdType {
	case itemAdd, labelAdd, noteAdd, projectAdd:
		u, _ := uuid.NewV4()
		c.TempID = u.String()
	default:
	}
	return c
}

func (c *Client) QueueItemAdd(item *ItemPatch) (temporaryID string) {
	add := newCommand(itemAdd, item)
	c.commands = append(c.commands, add)
	return add.TempID
}

func (c *Client) QueueItemUpdate(item *ItemPatch) {
	c.commands = append(c.commands, newCommand(itemUpdate, item))
}

func (c *Client) QueueItemDelete(id int64) {
	c.commands = append(c.commands, newCommand(itemDelete, idContainer{ID: id}))
}

func (c *Client) QueueItemClose(id int64) {
	c.commands = append(c.commands, newCommand(itemClose, idContainer{ID: id}))
}

// itemMoveCommand represents a command to move an item to another project.  (The project id property can not be
// set as part of an item update (which would achieve moving the project). This is just how the Todoist APIs work.)
type itemMoveCommand struct {
	ID        ID `json:"id"`
	ProjectID ID `json:"project_id"`
}

func (c *Client) QueueItemMove(item, project ID) {
	c.commands = append(c.commands, newCommand(itemMove, &itemMoveCommand{
		ID:        item,
		ProjectID: project,
	}))
}

func (c *Client) QueueItemReorder(reorder *ReorderCommand) {
	reorder.entity = "items"
	c.commands = append(c.commands, newCommand(itemReorder, reorder))
}

func (c *Client) QueueLabelAdd(label *LabelPatch) (temporaryID string) {
	add := newCommand(labelAdd, label)
	c.commands = append(c.commands, add)
	return add.TempID
}

func (c *Client) QueueLabelUpdate(label *LabelPatch) {
	c.commands = append(c.commands, newCommand(labelUpdate, label))
}

func (c *Client) QueueLabelDelete(id int64) {
	c.commands = append(c.commands, newCommand(labelDelete, idContainer{ID: id}))
}

func (c *Client) QueueProjectAdd(project *ProjectPatch) (temporaryID string) {
	add := newCommand(projectAdd, project)
	c.commands = append(c.commands, add)
	return add.TempID
}

func (c *Client) QueueProjectUpdate(project *ProjectPatch) {
	c.commands = append(c.commands, newCommand(projectUpdate, project))
}

func (c *Client) QueueProjectArchive(id int64) {
	c.commands = append(c.commands, newCommand(projectArchive, idContainer{ID: id}))
}

func (c *Client) QueueProjectDelete(id int64) {
	c.commands = append(c.commands, newCommand(projectDelete, idContainer{ID: id}))
}

func (c *Client) QueueProjectReorder(reorder *ReorderCommand) {
	reorder.entity = "projects"
	c.commands = append(c.commands, newCommand(projectReorder, reorder))
}

func (c *Client) QueueNoteAdd(note *NotePatch) (temporaryID string) {
	add := newCommand(noteAdd, note)
	c.commands = append(c.commands, add)
	return add.TempID
}

func (c *Client) QueueNoteUpdate(note *NotePatch) {
	c.commands = append(c.commands, newCommand(noteUpdate, note))
}

func (c *Client) QueueNoteDelete(id int64) {
	c.commands = append(c.commands, newCommand(noteDelete, idContainer{ID: id}))
}
