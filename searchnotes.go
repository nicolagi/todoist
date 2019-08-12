package todoist

type notePredicate func(*Note) bool

type NoteScan struct {
	client     *Client
	predicates []notePredicate
}

func (s *NoteScan) WithIsDeleted(value int) *NoteScan {
	s.predicates = append(s.predicates, func(note *Note) bool {
		return note.IsDeleted == value
	})
	return s
}

func (s *NoteScan) WithItemID(item int64) *NoteScan {
	s.predicates = append(s.predicates, func(note *Note) bool {
		return note.ItemID == item
	})
	return s
}

func (s *NoteScan) Results() []*Note {
	var results []*Note
	for _, note := range s.client.data.Notes {
		if s.match(note) {
			results = append(results, note)
		}
	}
	return results
}

func (s *NoteScan) match(note *Note) bool {
	for _, match := range s.predicates {
		if !match(note) {
			return false
		}
	}
	return true
}

func (c *Client) SearchNotes() *NoteScan {
	return &NoteScan{
		client: c,
	}
}
