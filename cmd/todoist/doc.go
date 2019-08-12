// The todoist program is an acme user interface to Todoist (https://todoist.com).
//
// The API token is expected at the file lib/todoist/token within the user's home directory.
//
// When launched, it creates an initial window listing all projects. Operation of the window via middle-click and
// right-click should be fairly intuitive to an acme user so I mostly won't document it.
//
// Be careful with the Zap command as it will delete items. With projects, it will archive rather
// than delete. You can also delete notes by 2-button-swiping "Zap 1234" where 1234 is a note id.
//
// Example arguments to Search: All items labeled "next":  @next.  All items labeled "bug" containing the string
// "foobar":  @bug:foobar.  All items labeled "feature" but not labeled maybe:  @feature:-@maybe.  All items in
// projects containing the string foobar:  #foobar.
//
// So, in summary, prepending minus negates a condition; the colon combines conditions (i.e., represents the
// boolean AND); the @ symbol introduces a condition on the item label; the # symbol introduces a condition on
// the item project name, while the default condition looks for substring in items.
package main // import "github.com/nicolagi/todoist/cmd/todoist"
