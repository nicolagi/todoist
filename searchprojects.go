package todoist

import "strings"

type projectPredicate func(*Project) bool

type ProjectScan struct {
	client     *Client
	predicates []projectPredicate
}

func (s *ProjectScan) WithIsArchived(value int) *ProjectScan {
	s.predicates = append(s.predicates, func(p *Project) bool {
		return p.IsArchived == value
	})
	return s
}

func (s *ProjectScan) WithIsDeleted(value int) *ProjectScan {
	s.predicates = append(s.predicates, func(p *Project) bool {
		return p.IsDeleted == value
	})
	return s
}

// WithName looks for projects containing the given substring, case-insensitive.
func (s *ProjectScan) WithName(needle string) *ProjectScan {
	needle = strings.ToLower(needle)
	s.predicates = append(s.predicates, func(p *Project) bool {
		return strings.Contains(strings.ToLower(p.Name), needle)
	})
	return s
}

func (s *ProjectScan) Results() []*Project {
	var results []*Project
	for _, project := range s.client.data.Projects {
		if s.match(project) {
			results = append(results, project)
		}
	}
	return results
}

func (s *ProjectScan) match(project *Project) bool {
	for _, match := range s.predicates {
		if !match(project) {
			return false
		}
	}
	return true
}

func (c *Client) SearchProjects() *ProjectScan {
	return &ProjectScan{
		client: c,
	}
}
