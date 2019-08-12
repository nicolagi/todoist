package todoist_test

import (
	"encoding/json"
	"testing"

	"github.com/nicolagi/todoist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectPatch(t *testing.T) {
	testCases := []struct {
		id       int64                       // The project to update
		setter   func(*todoist.ProjectPatch) // A function to set attributes to update
		expected string                      // The expected JSON output
	}{
		{
			id:       0,
			setter:   func(*todoist.ProjectPatch) {},
			expected: `{"id":0}`,
		},
		{
			id: 1,
			setter: func(p *todoist.ProjectPatch) {
				p.WithName("foobar")
			},
			expected: `{"id":1,"name":"foobar"}`,
		},
		{
			id: 2,
			setter: func(p *todoist.ProjectPatch) {
				p.WithColor(7)
			},
			expected: `{"id":2,"color":7}`,
		},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			project := todoist.NewProjectPatch(tc.id)
			tc.setter(project)
			b, err := json.Marshal(project)
			require.Nil(t, err)
			assert.Equal(t, tc.expected, string(b))
		})
	}
}
