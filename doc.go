// The todoist package contains a Todoist client that uses a subset of the Todoist Sync API v8 documented at
// https://developer.todoist.com/sync/v8. The client will be extended to support more functionality as required
// by consumers; at the time of writing the only consumer is the acme user interface in the cmd/todoist
// subdirectory.
//
// The client maintains lists of resources (projects, items, notes, labels) and all lookup and search operations
// scan through these lists to find the entities of interest. While this is obviously inefficient it is pointless to
// change implementation to improve performance as projects and tasks inventories should be small and the current
// only use case is the acme user interface in cmd/todoist.
//
// The only two client methods that make remote calls are Push and Pull. The former sends to the server the commands
// that were previously enqueued by the client, in bulk, while the latter fetches all changes that happened since
// the previous time it was called, including locally initiated changes (the first time, it will download all the data).
//
// Methods that query the data, e.g., ItemByID or SearchProjects, use the local copy of the data.  Methods that
// modify the data, e.g., QueueItemAdd, locally enqueue the changes to be later sent upstream by Push.
package todoist // import "github.com/nicolagi/todoist"
