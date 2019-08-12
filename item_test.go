package todoist_test

import (
	"encoding/json"
	"testing"

	"github.com/nicolagi/todoist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemPatchHappy(t *testing.T) {
	testCases := []struct {
		id       int64                    // item to patch
		setter   func(*todoist.ItemPatch) // function to set attributes in the patch
		expected string                   // expected JSON output
	}{
		{
			id:       0,
			setter:   func(*todoist.ItemPatch) {},
			expected: `{"id":0}`,
		},
		{
			id: 2,
			setter: func(item *todoist.ItemPatch) {
				item.WithProjectID(45)
			},
			expected: `{"id":2,"project_id":45}`,
		},
		{
			id: 3,
			setter: func(item *todoist.ItemPatch) {
				item.WithContent("something")
			},
			expected: `{"id":3,"content":"something"}`,
		},
		{
			id: 4,
			setter: func(item *todoist.ItemPatch) {
				item.WithContent(`it "quoted" the quote`)
			},
			expected: `{"id":4,"content":"it \"quoted\" the quote"}`,
		},
		{
			id: 5,
			setter: func(item *todoist.ItemPatch) {
				item.WithLabels()
			},
			expected: `{"id":5,"labels":[]}`,
		},
		{
			id: 6,
			setter: func(item *todoist.ItemPatch) {
				item.WithLabels(todoist.NewID(654))
			},
			expected: `{"id":6,"labels":[654]}`,
		},
		{
			id: 7,
			setter: func(item *todoist.ItemPatch) {
				item.WithLabels(todoist.NewID(654), todoist.NewID(88))
			},
			expected: `{"id":7,"labels":[654,88]}`,
		},
		{
			id: 8,
			setter: func(item *todoist.ItemPatch) {
				item.WithLabels(todoist.NewID(654), todoist.NewTemporaryID("eighty"))
			},
			expected: `{"id":8,"labels":[654,"eighty"]}`,
		},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			item := todoist.NewItemPatch(tc.id)
			tc.setter(item)
			b, err := json.Marshal(item)
			require.Nil(t, err)
			assert.Equal(t, tc.expected, string(b))
		})
	}
}

func TestItemPatchError(t *testing.T) {
	patch := todoist.NewItemPatch(13).WithLabels(todoist.NewID(0))
	b, err := json.Marshal(patch)
	assert.Nil(t, b)
	assert.NotNil(t, err)
}
