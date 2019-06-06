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
	require.Len(t, c.ValidatorGenerators, 3)
	require.Equal(t, c.TransactionToken, "sometoken")
	require.Equal(t, c.ID, "did:tupelo:exampleonly")
	require.Equal(t, c.BurnAmount, uint64(100))
	require.Equal(t, c.TransactionTopic, "tupelo-pubsub-topics")
	require.Equal(t, c.CommitTopic, "tupelo-pubsub-commits")
}
