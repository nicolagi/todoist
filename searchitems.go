package todoist

import "strings"

type itemPredicate func(*Item) bool

func negate(p itemPredicate) itemPredicate {
	return func(item *Item) bool {
		return !p(item)
	}
}

type ItemScan struct {
	client     *Client
	predicates []itemPredicate
}

// Not negates the last predicate added.  It will panic if no predicates were added.
func (s *ItemScan) Not() *ItemScan {
	i := len(s.predicates) - 1
	s.predicates[i] = negate(s.predicates[i])
	return s
}

// WithProjectID looks for items in any of the given project IDs, that is, arguments are ORed together.
func (s *ItemScan) WithProjectID(value ...int64) *ItemScan {
	s.predicates = append(s.predicates, func(item *Item) bool {
		for _, pid := range value {
			if item.ProjectID == pid {
				return true
			}
		}
		return false
	})
	return s
}

func (s *ItemScan) WithChecked(value int) *ItemScan {
	s.predicates = append(s.predicates, func(item *Item) bool {
		return item.Checked == value
	})
	return s
}

func (s *ItemScan) WithLabel(label int64) *ItemScan {
	s.predicates = append(s.predicates, func(item *Item) bool {
		for _, lid := range item.Labels {
			if lid == label {
				return true
			}
		}
		return false
	})
	return s
}

// WithContent looks for items containing the given substring.
func (s *ItemScan) WithContent(needle string) *ItemScan {
	s.predicates = append(s.predicates, func(item *Item) bool {
		return strings.Contains(item.Content, needle)
	})
	return s
}

func (s *ItemScan) WithDue() *ItemScan {
	s.predicates = append(s.predicates, func(item *Item) bool {
		return item.Due != nil
	})
	return s
}

func (s *ItemScan) Results() []*Item {
	var results []*Item
	for _, item := range s.client.data.Items {
		if s.match(item) {
			results = append(results, item)
		}
	}
	return results
}

func (s *ItemScan) match(item *Item) bool {
	for _, match := range s.predicates {
		if !match(item) {
			return false
		}
	}
	return true
}

func (c *Client) SearchItems() *ItemScan {
	return &ItemScan{
		client: c,
	}
}
