package todoist_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/nicolagi/todoist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIDMarshal(t *testing.T) {
	checkMarshal := func(in todoist.ID, out string) {
		b, err := json.Marshal(in)
		require.Nil(t, err)
		assert.Equal(t, out, string(b))
	}

	checkMarshalErr := func(in todoist.ID, expected error) {
		b, err := json.Marshal(in)
		require.Nil(t, b)
		assert.True(t, errors.Is(err, expected))
	}

	checkMarshal(todoist.NewID(42), "42")
	checkMarshal(todoist.NewTemporaryID("foobar"), `"foobar"`)
	checkMarshalErr(todoist.NewID(0), todoist.ErrZeroID)
	checkMarshalErr(todoist.NewTemporaryID(""), todoist.ErrZeroID)

	item := todoist.NewItemPatch(1).WithLabels(todoist.NewID(10), todoist.NewTemporaryID("ten"))
	b, err := json.Marshal(item)
	require.Nil(t, err)
	assert.Equal(t, `{"id":1,"labels":[10,"ten"]}`, string(b))
}

func TestIDUnmarshal(t *testing.T) {
	checkUnmarshal := func(in string, out todoist.ID) {
		var actual todoist.ID
		err := json.Unmarshal([]byte(in), &actual)
		require.Nil(t, err)
		assert.Equal(t, out, actual)
	}

	checkUnmarshalErr := func(in string, expected error) {
		var actual todoist.ID
		err := json.Unmarshal([]byte(in), &actual)
		t.Logf("Error is: %v", err)
		assert.True(t, errors.Is(err, expected))
	}

	checkUnmarshal("42", todoist.NewID(42))
	checkUnmarshal(`"foobar"`, todoist.NewTemporaryID("foobar"))
	checkUnmarshalErr("0", todoist.ErrZeroID)
	checkUnmarshalErr(`""`, todoist.ErrZeroID)
}
