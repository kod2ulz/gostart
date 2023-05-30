package collections_test

import (
	"testing"

	"github.com/kod2ulz/gostart/collections"
	"github.com/stretchr/testify/assert"
)

func TestCanMakeMapOfAny(t *testing.T) {
	out := collections.MapOf[string, int]("one", 1, "two", 2)
	assert.Equal(t, 2, len(out))
	assert.Equal(t, 1, out["one"])
	assert.Equal(t, 2, out["two"])
}
