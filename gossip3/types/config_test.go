package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	assert.NotEmpty(t, c.ValidatorGenerators)
	assert.NotEmpty(t, c.Transactions)
}
