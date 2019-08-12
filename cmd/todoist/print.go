package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/nicolagi/todoist"
)

var errNotFound = errors.New("entity not found")

func printAllProjects(w io.Writer) error {
	all := client.SearchProjects().WithIsArchived(0).WithIsDeleted(0).Results()
	sort.Sort(projectsByChildOrder(all))
	for _, p := range all {
		_, _ = fmt.Fprintf(w, "%v\t%v\n", p.ID, p.Name)
	}
	return nil
}

func printProjectByID(w io.Writer, id int64) error {
	items := client.SearchItems().WithProjectID(id).WithChecked(0).Results()
	sort.Sort(itemsByChildOrder(items))
	return printItems(w, items)
}

func printSearch(w io.Writer, expr string) error {
	search := client.SearchItems().WithChecked(0)
	for _, term := range strings.Split(expr, ":") {
		addSearchTerm(search, strings.TrimSpace(term))
	}
	return printItems(w, search.Results())
}

func printCalendar(w io.Writer) error {
	items := client.SearchItems().WithChecked(0).WithDue().Results()
	sort.Sort(itemsByDue(items))
	return printItems(w, items)
}

func relativeDurationFormat(d time.Duration) string {
	var buf bytes.Buffer
	t := d / (24 * time.Hour)
	if t != 0 {
		fmt.Fprintf(&buf, "%dd", t)
	}
	d -= t * 24 * time.Hour
	t = d / time.Hour
	if t != 0 {
		fmt.Fprintf(&buf, "%dh", t)
	}
	d -= t * time.Hour
	if buf.Len() == 0 {
		t = d / time.Minute
		if t != 0 {
			fmt.Fprintf(&buf, "%dm", t)
		}
	}
	return buf.String()
}

func printItems(w io.Writer, items []*todoist.Item) error {
	for _, i := range items {
		labelNames, err := getLabelNames(i.Labels)
		if err != nil {
			return fmt.Errorf("print items: %d: %w", i.ID, err)
		}
		dueIn := ""
		if i.Due != nil {
			dueIn = relativeDurationFormat(time.Until(i.Due.Time()))
		}
		_, _ = fmt.Fprintf(w, "%v\t%v\t%v\t%v\n", i.ID, strings.Join(labelNames, " "), dueIn, i.Content)
	}
	return nil
}

func printItemByID(w io.Writer, id int64) error {
	item, ok := client.ItemByID(id)
	if !ok {
		return fmt.Errorf("print item: %d: %w", id, errNotFound)
	}
	return printItem(w, item)
}

func printItem(w io.Writer, item *todoist.Item) error {
	projectName, err := getProjectName(item.ProjectID)
	if err != nil {
		return fmt.Errorf("print item: %d: %w", item.ID, err)
	}
	labelNames, err := getLabelNames(item.Labels)
	if err != nil {
		return fmt.Errorf("print item: %d: %w", item.ID, err)
	}
	_, _ = fmt.Fprintf(w, "Content: %s\n", item.Content)
	_, _ = fmt.Fprintf(w, "Project: %s\n", projectName)
	_, _ = fmt.Fprintf(w, "Labels: %s\n", strings.Join(labelNames, " "))
	if item.Due != nil {
		_, _ = fmt.Fprintf(w, "Due: %s\n", item.Due.Date)
	} else {
		_, _ = fmt.Fprint(w, "Due: \n")
	}
	_, _ = fmt.Fprint(w, "Note: \n")

	labels := client.SearchLabels().WithIsDeleted(0).Results()
	labelNames = nil
	for _, label := range labels {
		labelNames = append(labelNames, label.Name)
	}
	sort.Strings(labelNames)
	_, _ = fmt.Fprintf(w, "Available labels: %s\n", strings.Join(labelNames, " "))

	notes := client.SearchNotes().WithIsDeleted(0).WithItemID(item.ID).Results()
	sort.Sort(notesByPosted(notes))
	for _, note := range notes {
		_, _ = fmt.Fprintf(w, "\n%d @ %s\n\n%s\n", note.ID, note.Posted, note.Content)
	}

	return nil
}

func printNewItemForProject(w io.Writer, projectID int64) error {
	project, err := getProjectName(projectID)
	if err != nil {
		return fmt.Errorf("print new item for project %d: %w", projectID, err)
	}
	labels := client.SearchLabels().WithIsDeleted(0).Results()
	var labelNames []string
	for _, label := range labels {
		labelNames = append(labelNames, label.Name)
	}
	_, _ = fmt.Fprintf(w, `Content: 
Project: %s
Labels: 
Due: 
Note: 
Available labels: %s
`, project, strings.Join(labelNames, " "))
	return nil
}

func getProjectName(id int64) (string, error) {
	p, ok := client.ProjectByID(id)
	if !ok {
		return "", fmt.Errorf("project %d: %w", id, errNotFound)
	}
	return p.Name, nil
}

func getLabelNames(ids []int64) ([]string, error) {
	var names []string
	for _, id := range ids {
		label, ok := client.LabelByID(id)
		if !ok {
			return nil, fmt.Errorf("label %d: %w", id, errNotFound)
		}
		names = append(names, label.Name)
	}
	sort.Strings(names)
	return names, nil
}

func addSearchTerm(s *todoist.ItemScan, term string) {
	switch term[0] {
	case '-':
		addSearchTerm(s, term[1:])
		s.Not()
	case '@':
		label := client.LabelByName(term[1:])
		if label != nil {
			s.WithLabel(label.ID)
		} else {
			// This won't match any item.
			s.WithLabel(0)
		}
	case '#':
		var pids []int64
		for _, p := range client.SearchProjects().WithIsArchived(0).WithIsDeleted(0).WithName(term[1:]).Results() {
			pids = append(pids, p.ID)
		}
		s.WithProjectID(pids...)
	default:
		s.WithContent(term)
	}
}
