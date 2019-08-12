package todoist

import (
	"encoding/json"
	"errors"
)

// ErrZeroID is returned by marshalling or unmarshalling JSON. If one uses NewID passing non-zero values and
// NewTemporaryID passing non-empty strings to construct ID values, this error won't happen.
var ErrZeroID = errors.New("both permanent and temporary id are zero")

// ID represents either a numeric or a string identifier. This type is needed because, for example, an item's
// labels property is an array where items can be numbers (permanent ids) or strings (temporary ids).  Since you
// can not declare a slice with two types, we'll represent the labels property as a slice of *ID.  Custom JSON
// marshal and unmarshal methods will take care of choosing the right representation based on what field of the
// struct is actually set.
type ID struct {
	pid int64  // Permanent id
	tid string // Temporary id
}

func NewID(value int64) ID {
	return ID{pid: value}
}

func NewTemporaryID(value string) ID {
	return ID{tid: value}
}

// MarshalJSON implements json.Marshaler.
func (id ID) MarshalJSON() ([]byte, error) {
	if id.pid == 0 && id.tid == "" {
		return nil, ErrZeroID
	}
	if id.pid != 0 {
		return json.Marshal(id.pid)
	}
	return json.Marshal(id.tid)
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *ID) UnmarshalJSON(b []byte) error {
	if b[0] == '"' {
		if err := json.Unmarshal(b, &id.tid); err != nil {
			return err
		}
		if len(id.tid) == 0 {
			return ErrZeroID
		}
		return nil
	}
	if err := json.Unmarshal(b, &id.pid); err != nil {
		return err
	}
	if id.pid == 0 {
		return ErrZeroID
	}
	return nil
}
