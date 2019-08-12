package todoist

type labelPredicate func(*Label) bool

type LabelScan struct {
	client     *Client
	predicates []labelPredicate
}

func (s *LabelScan) WithIsDeleted(value int) *LabelScan {
	s.predicates = append(s.predicates, func(label *Label) bool {
		return label.IsDeleted == value
	})
	return s
}

func (s *LabelScan) Results() []*Label {
	var results []*Label
	for _, label := range s.client.data.Labels {
		if s.match(label) {
			results = append(results, label)
		}
	}
	return results
}

func (s *LabelScan) match(label *Label) bool {
	for _, match := range s.predicates {
		if !match(label) {
			return false
		}
	}
	return true
}

func (c *Client) SearchLabels() *LabelScan {
	return &LabelScan{
		client: c,
	}
}
