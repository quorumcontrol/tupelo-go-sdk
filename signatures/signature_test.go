package signatures

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEcdsaAddress(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.Nil(t, err)

	// With no conditions it's the same as a normal key
	o := &Ownership{
		Type:      KeyTypeSecp256k1,
		PublicKey: crypto.FromECDSAPub(&key.PublicKey),
	}
	addr, err := o.Address()
	require.Nil(t, err)
	assert.Equal(t, addr.String(), crypto.PubkeyToAddress(key.PublicKey).String())

	// with conditions, it changes the addr
	o.Conditions = "true"
	conditionalAddr, err := o.Address()
	require.Nil(t, err)
	assert.NotEqual(t, conditionalAddr.String(), crypto.PubkeyToAddress(key.PublicKey).String())
	assert.Len(t, conditionalAddr, 20) // same length as an addr

	// changing the conditions changes the addr
	o.Conditions = "false"
	conditionalAddr2, err := o.Address()
	require.Nil(t, err)
	assert.NotEqual(t, conditionalAddr2.String(), conditionalAddr.String())
	assert.Len(t, conditionalAddr2, 20)
}

func TestBLSAddr(t *testing.T) {
	key := bls.MustNewSignKey()

	// With no conditions it's the same as a normal key
	o := &Ownership{
		Type:      KeyTypeBLSGroupSig,
		PublicKey: key.MustVerKey().Bytes(),
	}
	addr, err := o.Address()
	require.Nil(t, err)
	assert.Equal(t, addr.String(), bytesToAddress(o.PublicKey).String())

	// with conditions, it changes the addr
	o.Conditions = "true"
	conditionalAddr, err := o.Address()
	require.Nil(t, err)
	assert.NotEqual(t, conditionalAddr.String(), bytesToAddress(o.PublicKey).String())
	assert.Len(t, conditionalAddr, 20) // same length as an addr

	// changing the conditions changes the addr
	o.Conditions = "false"
	conditionalAddr2, err := o.Address()
	require.Nil(t, err)
	assert.NotEqual(t, conditionalAddr2.String(), conditionalAddr.String())
	assert.Len(t, conditionalAddr2, 20)
}
