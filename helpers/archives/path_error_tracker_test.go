package archives

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNilsArentActionable(t *testing.T) {
	var genericNil error
	var typedNil *os.PathError
	tracker := newPathErrorTracker()

	assert.False(t, tracker.actionable(genericNil), "Untyped nils should not be actionable")
	assert.False(t, tracker.actionable(typedNil), "PathError typed nils should not be actionable")
}

func TestPathErrorIsActionableTheFirstTimeOnly(t *testing.T) {
	pathErr1 := &os.PathError{Op: "anything"}
	pathErr2 := &os.PathError{Op: "anything"}
	pathErr3 := &os.PathError{Op: "something else"}
	tracker := newPathErrorTracker()

	assert.True(t, tracker.actionable(pathErr1), "Should be actionable the first time an Op is seen")
	assert.False(t, tracker.actionable(pathErr2), "Should not be actionable if the same Op is seen in a different instance")
	assert.False(t, tracker.actionable(pathErr1), "Should not be actionable if the same instance is passed again")
	assert.True(t, tracker.actionable(pathErr3), "Another Op should be actionable")
}

func TestNonPathErrorsAlwaysActionable(t *testing.T) {
	nonPathErrs := []error{errors.New("one"), errors.New("two")}
	nonPathErrs = append(nonPathErrs, nonPathErrs...) // try each error twice
	tracker := newPathErrorTracker()

	for i, err := range nonPathErrs {
		assert.True(t, tracker.actionable(err), "#%d should be actionable", i)
	}
}
