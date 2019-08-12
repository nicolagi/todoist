package todoist

import (
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type Due struct {
	// From v8 API doc: Due date in the format of YYYY-MM-DD (RFC 3339). For recurring dates, the date of the
	// current iteration.
	Date string `json:"date"`

	// From v8 API doc: Always set to null.
	Timezone string `json:"timezone"`

	// From v8 API doc: Human-readable representation of due date. String always represents the due object in
	// user’s timezone. Look at our reference to see which formats are supported.
	String string `json:"string"`

	// From v8 API doc: Lang which has to be used to parse the content of the string attribute. Used by clients
	// and on the server side to properly process due dates when date object is not set, and when dealing with
	// recurring tasks. Valid languages are: en, da, pl, zh, ko, de, pt, ja, it, fr, sv, ru, es, nl.}
	Lang string `json:"lang"`

	IsRecurring bool `json:"is_recurring"`

	// This will be lazily populated by parsing the Date property.
	time time.Time
}

func (due Due) Time() time.Time {
	if !due.time.IsZero() {
		return due.time
	}
	date := due.Date
	if !strings.Contains(date, "T") {
		date += "T23:59:59Z"
	}
	var err error
	due.time, err = time.Parse(time.RFC3339, date)
	if err != nil {
		log.WithFields(log.Fields{
			"cause": err,
			"date":  due.Date,
		}).Warning("Could not parse time, has Todoist changed format?")
	}
	return due.time
}
