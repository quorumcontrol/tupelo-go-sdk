package types

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestTomlLoading(t *testing.T) {
	bits, err := ioutil.ReadFile("example.toml")
	require.Nil(t, err)
	c, err := TomlToConfig(string(bits))
	assert.Nil(t, err)
	assert.Len(t, c.Transactions, 1)
	assert.Len(t, c.ValidatorGenerators, 3)
	assert.Equal(t, "sometoken", c.TransactionToken)
	assert.Equal(t, "did:tupelo:exampleonly", c.ID)
	assert.Equal(t, uint64(100), c.BurnAmount)
	assert.Equal(t, "tupelo-pubsub-topics", c.TransactionTopic)
	assert.Equal(t, "tupelo-pubsub-commits", c.CommitTopic)
	assert.Equal(t, []string{"/ipv4/0.0.0.0/tcp/1024/ipfs/Qmyadayada"}, c.BootstrapAddresses)
	assert.Len(t, c.Signers, 2)
}

func TestDefaultValues(t *testing.T) {
	bits, err := ioutil.ReadFile("example_simple.toml")
	require.Nil(t, err)
	c, err := TomlToConfig(string(bits))
	require.Nil(t, err)
	assert.Len(t, c.Transactions, 7)
	assert.Len(t, c.ValidatorGenerators, 2)
	assert.Equal(t, "", c.TransactionToken)
	assert.Equal(t, "did:tupelo:simple", c.ID)
	assert.Equal(t, uint64(0), c.BurnAmount)
	assert.Equal(t, "tupelo-transaction-broadcast", c.TransactionTopic)
	assert.Equal(t, "tupelo-commits", c.CommitTopic)
	assert.Len(t, c.Signers, 0)
}

func TestErrorsWhenBlank(t *testing.T) {
	_, err := TomlToConfig("")
	require.NotNil(t, err)
}

func TestTomLoadingInvalidGenerator(t *testing.T) {
	tomlStr := `
	transactions = ["somethingnotthere"]
	`
	_, err := TomlToConfig(tomlStr)
	t.Logf("[good] error: %v", err)
	require.NotNil(t, err)
}
