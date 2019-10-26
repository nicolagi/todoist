package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"9fans.net/go/acme"
	"github.com/nicolagi/todoist"
	log "github.com/sirupsen/logrus"
)

type windowMode int

const (
	modeItem        windowMode = iota // /todo/items/$id
	modeNewItem                       // /todo/items/new
	modeProject                       // /todo/projects/$id
	modeNewProject                    // /todo/projects/new
	modeAllProjects                   // /todo/projects/all
	modeSearch                        // /todo/search/$expr
	modeCalendar                      // /todo/calendar
)

func (mode windowMode) String() string {
	switch mode {
	case modeItem:
		return "item"
	case modeNewItem:
		return "newItem"
	case modeProject:
		return "project"
	case modeNewProject:
		return "newProject"
	case modeAllProjects:
		return "allProjects"
	case modeSearch:
		return "search"
	case modeCalendar:
		return "calendar"
	default:
		log.WithField("mode", int(mode)).Error("Missing mode string, returning as number")
		return fmt.Sprintf("%d", int(mode))
	}
}

var all struct {
	sync.Mutex
	m map[*acme.Win]*window
}

type window struct {
	*acme.Win

	mode windowMode

	projectID int64  // For modeProject, modeItem, modeNewItem
	itemID    int64  // For modeItem
	expr      string // For modeSearch

	// If false, sort by item.ItemOrder, as in the web app.  Only used for project mode, search mode, and all
	// projects mode.
	sortAlphabetically bool
}

// resetTag is used when a new window is created, or when transitioning a window from new item (project) mode to
// item (project) mode.
func (w *window) resetTag() {
	var tag string
	switch w.mode {
	case modeItem:
		tag = " Projects Calendar New Get Put PutDel Complete Zap "
	case modeNewItem:
		tag = " Projects Calendar Put PutDel "
	case modeProject:
		tag = " Projects Calendar New Get Put PutDel Sort Zap "
	case modeNewProject:
		tag = " Projects Calendar Put PutDel "
	case modeAllProjects:
		tag = " Calendar New Get Put PutDel Sort Search Zap "
	case modeSearch:
		tag = " Projects Calendar Get Sort Search Zap "
	case modeCalendar:
		tag = " Projects Get Search Zap "
	}
	_ = w.Ctl("cleartag")
	_ = w.Fprintf("tag", tag)
}

// exit is called after the window's event loop is over, i.e., the window has been closed in acme.  If it's the
// last window, we try to save the data before terminating the process.
func (w *window) exit() {
	all.Lock()
	defer all.Unlock()
	if all.m[w.Win] == w {
		delete(all.m, w.Win)
	}
	if len(all.m) == 0 {
		if err := client.Dump(); err != nil {
			log.WithField("cause", err).Warning("Could not dump data locally")
		}
		os.Exit(0)
	}
}

// newWindow creates a window in acme without a specific purpose, and registers it in the global map of windows.
func newWindow(pathname string) *window {
	all.Lock()
	defer all.Unlock()
	if all.m == nil {
		all.m = make(map[*acme.Win]*window)
	}

	logEntry := log.WithField("path", pathname)
	aw, err := acme.New()
	if err != nil {
		logEntry.WithField("cause", err).Warning("Could not create acme window")
		time.Sleep(10 * time.Millisecond)
		aw, err = acme.New()
		if err != nil {
			logEntry.WithField("cause", err).Fatal("Could not create acme window again")
		}
	}
	aw.SetErrorPrefix(pathname)
	_ = aw.Name(pathname)

	w := &window{Win: aw}
	all.m[w.Win] = w
	return w
}

func newAllProjectsWindow() {
	title := "/todo/projects/all"
	if acme.Show(title) != nil {
		return
	}
	w := newWindow(title)
	w.mode = modeAllProjects
	w.resetTag()
	go w.load()
	go w.loop()
}

func newSearchWindow(expr string) {
	title := "/todo/search/" + expr
	if acme.Show(title) != nil {
		return
	}
	w := newWindow(title)
	w.mode = modeSearch
	w.expr = expr
	w.resetTag()
	go w.load()
	go w.loop()
}

func newProjectWindow(id int64) {
	var title string
	if id != 0 {
		title = fmt.Sprintf("/todo/projects/%d", id)
	} else {
		title = "/todo/projects/new"
	}
	if acme.Show(title) != nil {
		return
	}
	w := newWindow(title)
	if id != 0 {
		w.projectID = id
		w.mode = modeProject
	} else {
		w.mode = modeNewProject
	}
	w.resetTag()
	go w.load()
	go w.loop()
}

func newItemWindow(itemID, projectID int64) {
	var title string
	if itemID != 0 {
		title = fmt.Sprintf("/todo/items/%d", itemID)
	} else {
		title = "/todo/items/new"
	}
	if acme.Show(title) != nil {
		return
	}
	w := newWindow(title)
	if itemID != 0 {
		w.mode = modeItem
		w.itemID = itemID
	} else {
		w.mode = modeNewItem
	}
	w.projectID = projectID
	w.resetTag()
	go w.load()
	go w.loop()
}

func newCalendarWindow() {
	title := "/todo/calendar"
	if acme.Show(title) != nil {
		return
	}
	w := newWindow(title)
	w.mode = modeCalendar
	w.resetTag()
	go w.load()
	go w.loop()
}

// Look is invoked via button-3 click in acme. We need to see if we can open other windows from the current
// one, e.g., if text contains an item id or a project id. Should return true if we were able to handle
// the action, otherwise return false to defer to other handlers (to, e.g., open a URL in the browser).
func (w *window) Look(text string) bool {
	switch w.mode {
	case modeAllProjects:
		if id, err := strconv.ParseInt(text, 10, 64); err == nil {
			if _, ok := client.ProjectByID(id); ok {
				newProjectWindow(id)
				return true
			}
		}
	case modeItem:
		if projects := client.SearchProjects().WithName(text).Results(); len(projects) != 0 {
			newProjectWindow(projects[0].ID)
			return true
		}
	case modeProject, modeSearch, modeCalendar:
		id, err := strconv.ParseInt(text, 10, 64)
		if err == nil {
			if item, ok := client.ItemByID(id); ok {
				newItemWindow(id, item.ProjectID)
				return true
			}
		}
	}
	return false
}

func (w *window) load() {
	if err := client.Pull(); err != nil {
		w.Errf("load: pull: %v", err)
	}
	var buf bytes.Buffer
	var err error
	switch w.mode {
	case modeNewItem:
		err = printNewItemForProject(&buf, w.projectID)
	case modeNewProject:
		// Leave buffer empty.
	case modeItem:
		err = printItemByID(&buf, w.itemID)
	case modeProject:
		err = printProjectByID(&buf, w.projectID)
	case modeSearch:
		err = printSearch(&buf, w.expr)
	case modeAllProjects:
		err = printAllProjects(&buf)
	case modeCalendar:
		err = printCalendar(&buf)
	}
	w.Clear()
	if err != nil {
		_, _ = w.Write("body", []byte(err.Error()))
	} else if w.mode != modeProject && w.mode != modeSearch {
		_, _ = w.Write("body", buf.Bytes())
		_ = w.Ctl("clean")
	} else {
		w.PrintTabbed(buf.String())
		_ = w.Ctl("clean")
	}

	if err == nil && (w.mode == modeItem || w.mode == modeNewItem) {
		_ = w.Addr("#9") // Past "Content: "
	} else {
		_ = w.Addr("0")
	}
	_ = w.Ctl("dot=addr")
	_ = w.Ctl("show")
}

func (w *window) sort() {
	if err := w.Addr("0/^[0-9]/,"); err != nil {
		w.Err("nothing to sort")
	}
	var less func(string, string) bool
	if !w.sortAlphabetically {
		less = func(a, b string) bool { return lineNumber(a) < lineNumber(b) }
	} else {
		less = func(a, b string) bool { return skipField(a) < skipField(b) }
	}
	if err := w.Sort(less); err != nil {
		w.Errf("Could not sort: %v", err.Error())
	}
	_ = w.Addr("0")
	_ = w.Ctl("dot=addr")
	_ = w.Ctl("show")
}

func lineNumber(s string) int {
	n := 0
	for j := 0; j < len(s) && '0' <= s[j] && s[j] <= '9'; j++ {
		n = n*10 + int(s[j]-'0')
	}
	return n
}

func skipField(s string) string {
	i := strings.Index(s, "\t")
	if i < 0 {
		return s
	}
	for i < len(s) && s[i] == '\t' {
		i++
	}
	return s[i:]
}

// Execute is triggered by button-2 click in acme.
func (w *window) Execute(cmd string) bool {
	if strings.HasPrefix(cmd, "Search ") {
		expr := strings.TrimSpace(strings.TrimPrefix(cmd, "Search "))
		newSearchWindow(expr)
		return true
	}
	if cmd == "Zap" { // Try to infer argument
		switch w.mode {
		case modeProject:
			cmd += fmt.Sprintf(" %d", w.projectID)
		case modeItem:
			cmd += fmt.Sprintf(" %d", w.itemID)
		}
	}
	if strings.HasPrefix(cmd, "Zap ") {
		what := strings.TrimSpace(strings.TrimPrefix(cmd, "Zap "))
		if label := client.LabelByName(what); label != nil {
			client.QueueLabelDelete(label.ID)
			if err := client.Push(); err != nil {
				w.Errf("Could not delete %v", label)
			}
			return true
		}
		id, err := strconv.ParseInt(what, 10, 64)
		if err != nil {
			return false
		}
		if note, ok := client.NoteByID(id); ok {
			client.QueueNoteDelete(id)
			if err := client.Push(); err != nil {
				w.Errf("Could not delete %v", note)
			} else {
				onNoteZapped(note.ItemID)
			}
			return true
		} else if item, ok := client.ItemByID(id); ok {
			client.QueueItemDelete(id)
			if err := client.Push(); err != nil {
				w.Errf("Could not delete %v", item)
			} else {
				onItemZapped(id, item.ProjectID)
			}
			return true
		} else if project, ok := client.ProjectByID(id); ok {
			client.QueueProjectArchive(id)
			if err := client.Push(); err != nil {
				w.Errf("Could not archive %v", project)
			} else {
				onProjectZapped(id)
			}
			return true
		}
		return false
	}
	switch cmd {
	case "Projects":
		newAllProjectsWindow()
		return true
	case "Calendar":
		newCalendarWindow()
		return true
	case "Get":
		w.load()
		return true
	case "Put", "PutDel":
		del := cmd == "PutDel"
		if w.mode == modeNewProject {
			tempID, err := func() (string, error) {
				name, err := w.ReadAll("body")
				if err != nil {
					return "", err
				}
				pp := todoist.NewProjectPatch(0).WithColor(0).WithName(strings.TrimSpace(string(name))).WithChildOrder(1)
				tempID := client.QueueProjectAdd(pp)
				return tempID, client.Push()
			}()
			if err != nil {
				w.Errf("Failed adding project: %v", err)
			} else {
				if id, ok := client.PermanentID(tempID); ok {
					_ = w.Name("/todo/projects/%d", id)
					w.mode = modeProject
					w.projectID = id
					_ = w.Ctl("clean")
					w.resetTag()
				} else {
					_ = w.Name("/todo/projects/%s", tempID)
					_ = w.Ctl("clean")
				}
				if del {
					_ = w.Del(true)
				}
				onProjectPut()
			}
		} else if w.mode == modeNewItem {
			item := todoist.NewItemPatch(0).WithProjectID(w.projectID).WithChildOrder(1)
			note := todoist.NewNotePatch(0)
			if err := w.populateItem(item, note); err != nil {
				w.Errf("Failed parsing edited window: %v", err)
			} else {
				tempID := client.QueueItemAdd(item)
				if !note.Empty() {
					client.QueueNoteAdd(note.WithItemID(todoist.NewTemporaryID(tempID)))
				}
				if err := client.Push(); err != nil {
					w.Errf("Failed adding item: %v", err)
				} else {
					if id, ok := client.PermanentID(tempID); ok {
						_ = w.Name("/todo/items/%d", id)
						w.mode = modeItem
						w.itemID = id
						w.resetTag()
						if del {
							_ = w.Del(true)
						}
						onItemPut(id, w.projectID)
					} else {
						_ = w.Name("/todo/items/%s", tempID)
						_ = w.Ctl("clean")
					}
				}
			}
		} else if w.mode == modeAllProjects {
			err := func() error {
				var reorder todoist.ReorderCommand
				data, err := w.ReadAll("body")
				if err != nil {
					return err
				}
				lines := strings.Split(string(data), "\n")
				for i, line := range lines {
					fields := strings.Fields(line)
					if len(fields) == 0 {
						continue
					}
					id, err := strconv.ParseInt(fields[0], 10, 64)
					if err != nil {
						log.WithField("line", line).Warning("Ignoring line that does not start with a number")
						continue
					}
					p, ok := client.ProjectByID(id)
					if !ok {
						log.WithField("line", line).Warning("Ignoring line that refers to an unknown project")
						continue
					}
					if p.ChildOrder != i {
						reorder.Add(id, i)
					}
					if name := strings.TrimSpace(strings.Join(fields[1:], " ")); len(name) > 0 {
						if p.Name != name {
							client.QueueProjectUpdate(todoist.NewProjectPatch(id).WithColor(0).WithName(name))
						}
					}
				}
				if !reorder.Empty() {
					client.QueueProjectReorder(&reorder)
				}
				return client.Push()
			}()
			if err != nil {
				w.Errf("Could not reorder projects and/or update their names: %v", err)
			} else {
				_ = w.Ctl("clean")
				if del {
					_ = w.Del(true)
				}
				onAllProjectsPut()
			}
		} else if w.mode == modeProject {
			err := func() error {
				projectID := todoist.NewID(w.projectID)
				var reorder todoist.ReorderCommand
				data, err := w.ReadAll("body")
				if err != nil {
					return err
				}
				lines := strings.Split(string(data), "\n")
				for i, line := range lines {
					fields := strings.Fields(line)
					if len(fields) == 0 {
						continue
					}
					id, err := strconv.ParseInt(fields[0], 10, 64)
					if err != nil {
						log.WithField("line", line).Warning("Ignoring line that does not start with a number")
						continue
					}
					item, ok := client.ItemByID(id)
					if !ok {
						log.WithField("line", line).Warning("Ignoring line that refers to an unknown item")
						continue
					}
					if item.ProjectID != w.projectID {
						client.QueueItemMove(todoist.NewID(item.ID), projectID)
					}
					if item.ChildOrder != i {
						reorder.Add(id, i)
					}
				}
				if !reorder.Empty() {
					client.QueueItemReorder(&reorder)
				}
				return client.Push()
			}()
			if err != nil {
				w.Errf("Could not update reorder items/move items across projects: %v", err)
			} else {
				_ = w.Ctl("clean")
				if del {
					_ = w.Del(true)
				}
				onProjectPut()
			}
		} else if w.mode == modeItem {
			item := todoist.NewItemPatch(w.itemID)
			note := todoist.NewNotePatch(0).WithItemID(todoist.NewID(w.itemID))
			if err := w.populateItem(item, note); err != nil {
				w.Errf("Failed parsing edited window: %v", err)
				return true
			}
			client.QueueItemUpdate(item)
			if !note.Empty() {
				client.QueueNoteAdd(note)
			}
			if err := client.Push(); err != nil {
				w.Errf("Could not update item: %v", err)
			} else {
				if del {
					_ = w.Del(true)
				}
				onItemPut(w.itemID, w.projectID)
			}
		} else {
			w.Errf("Put forbidden for this window mode: %v", w.mode)
		}
		return true
	case "Del":
		_ = w.Del(false)
		return true
	case "New":
		if w.mode == modeProject || w.mode == modeItem {
			newItemWindow(0, w.projectID)
		} else if w.mode == modeAllProjects {
			newProjectWindow(0)
		} else {
			w.Errf("Trying to create a new entity in a window with mode: %v", w.mode)
		}
		return true
	case "Sort":
		if w.mode == modeProject || w.mode == modeAllProjects {
			w.sortAlphabetically = !w.sortAlphabetically
			w.sort()
		} else {
			w.Errf("Window mode does not allow sorting: %v", w.mode)
		}
		return true
	case "Complete":
		if w.mode == modeItem {
			if item, ok := client.ItemByID(w.itemID); ok {
				client.QueueItemClose(w.itemID)
				if err := client.Push(); err != nil {
					w.Errf("Could not complete item: %v", err)
				} else {
					// Same reaction to complete and delete.
					onItemZapped(item.ID, item.ProjectID)
				}
			} else {
				w.Errf("Item not found: %d", w.itemID)
			}
		} else {
			w.Errf("Complete only works in item mode, mode is %v", w.mode)
		}
		return true
	default:
		return false
	}
}

// Reads up the body and parses it to update properties in the passed item object.
func (w *window) populateItem(item *todoist.ItemPatch, note *todoist.NotePatch) error {
	data, err := w.ReadAll("body")
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Content:") {
			c := strings.TrimSpace(line[len("Content:"):])
			// Hard to imagine one intends to make the content empty.
			if len(c) > 0 {
				item.WithContent(c)
			}
		} else if strings.HasPrefix(line, "Labels:") {
			var labels []todoist.ID
			for _, name := range strings.Fields(line[len("Labels:"):]) {
				label := client.LabelByName(name)
				if label != nil {
					labels = append(labels, todoist.NewID(label.ID))
				} else {
					tid := client.QueueLabelAdd(todoist.NewLabelPatch(0).WithName(name))
					labels = append(labels, todoist.NewTemporaryID(tid))
				}
			}
			item.WithLabels(labels...)
		} else if strings.HasPrefix(line, "Note:") {
			if content := strings.TrimSpace(line[len("Note:"):]); content != "" {
				note.WithContent(content)
			}
		} else if strings.HasPrefix(line, "Due:") {
			// Format should be "2019-08-03" or "2019-08-03T15:30:00Z" (due day, due day and time,
			// respectively.)
			if date := strings.TrimSpace(line[len("Due:"):]); date != "" {
				item.WithDue(date)
			}
		}
	}
	return nil
}

func (w *window) loop() {
	defer w.exit()
	w.EventLoop(w)
}

func onAllProjectsPut() {
	all.Lock()
	defer all.Unlock()
	for _, w := range all.m {
		switch w.mode {
		case modeAllProjects, modeProject:
			// Not needed right now for project windows, as the project window does not show the
			// project name, but it might in the future.
			w.load()
		}
	}
}

func onItemPut(itemID, projectID int64) {
	all.Lock()
	defer all.Unlock()
	for _, w := range all.m {
		switch w.mode {
		case modeSearch, modeCalendar:
			w.load()
		case modeProject:
			if w.projectID == projectID {
				w.load()
			}
		case modeItem:
			if w.itemID == itemID {
				_ = w.Ctl("clean")
				w.load()
			}
		}
	}
}

func onProjectPut() {
	all.Lock()
	defer all.Unlock()
	for _, w := range all.m {
		switch w.mode {
		case modeAllProjects:
			// Only needed for project creation, really; but someday we might be able to change the
			// project name from the project window.
			w.load()
		case modeProject:
			// Only actually needed if an item has been moved away from the window's project to the
			// project that was put.
			w.load()
		}
	}
}

func onProjectZapped(projectID int64) {
	all.Lock()
	defer all.Unlock()
	for _, w := range all.m {
		switch w.mode {
		case modeAllProjects, modeSearch, modeCalendar:
			w.load()
		case modeItem, modeNewItem, modeProject:
			if w.projectID == projectID {
				_ = w.Del(true)
			}
		}
	}
}

func onItemZapped(itemID, projectID int64) {
	all.Lock()
	defer all.Unlock()
	for _, w := range all.m {
		if w.mode == modeSearch || w.mode == modeCalendar {
			w.load()
		}
		if w.mode == modeProject && w.projectID == projectID {
			w.load()
		}
		if w.mode == modeItem && w.itemID == itemID {
			_ = w.Del(true)
		}
	}
}

func onNoteZapped(itemID int64) {
	all.Lock()
	defer all.Unlock()
	for _, w := range all.m {
		if w.mode == modeItem && w.itemID == itemID {
			w.load()
		}
	}
}
