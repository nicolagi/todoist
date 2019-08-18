package main

import "github.com/nicolagi/todoist"

type itemsByDue []*todoist.Item

func (items itemsByDue) Len() int {
	return len(items)
}

func (items itemsByDue) Swap(i, j int) {
	items[i], items[j] = items[j], items[i]
}

func (items itemsByDue) Less(i, j int) bool {
	a, b := items[i].Due, items[j].Due
	if a == nil && b == nil {
		return items[i].ID < items[j].ID
	}
	if a != nil && b == nil {
		return true
	}
	if a == nil && b != nil {
		return false
	}
	if a.Time().Unix() == b.Time().Unix() {
		return items[i].ID < items[j].ID
	}
	return a.Time().Before(b.Time())
}

type notesByPosted []*todoist.Note

func (notes notesByPosted) Len() int {
	return len(notes)
}

func (notes notesByPosted) Swap(i, j int) {
	notes[i], notes[j] = notes[j], notes[i]
}

func (notes notesByPosted) Less(i, j int) bool {
	return notes[i].Time().Before(notes[j].Time())
}

type itemsByChildOrder []*todoist.Item

func (items itemsByChildOrder) Len() int {
	return len(items)
}

func (items itemsByChildOrder) Swap(i, j int) {
	items[i], items[j] = items[j], items[i]
}

func (items itemsByChildOrder) Less(i, j int) bool {
	return items[i].ChildOrder < items[j].ChildOrder
}

type projectsByChildOrder []*todoist.Project

func (projects projectsByChildOrder) Len() int {
	return len(projects)
}

func (projects projectsByChildOrder) Swap(i, j int) {
	projects[i], projects[j] = projects[j], projects[i]
}

func (projects projectsByChildOrder) Less(i, j int) bool {
	return projects[i].ChildOrder < projects[j].ChildOrder
}
