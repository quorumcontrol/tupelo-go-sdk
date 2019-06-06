package types

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTomlLoading(t *testing.T) {
	bits, err := ioutil.ReadFile("example.toml")
	require.Nil(t, err)
	c, err := TomlToConfig(string(bits))
	require.Nil(t, err)
	require.Len(t, c.Transactions, 1)
}
