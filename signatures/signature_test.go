package signatures

import (
	"fmt"
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

func TestEcdsaKeyRestore(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	msg := crypto.Keccak256([]byte("hi hi"))

	sigBits, err := crypto.Sign(msg, key)
	require.Nil(t, err)
	sig := &Signature{
		Ownership: &Ownership{
			Type: KeyTypeSecp256k1,
		},
		Signature: sigBits,
	}
	err = sig.RestorePublicKey(msg)
	require.Nil(t, err)
	assert.Len(t, sig.PublicKey, 65)
	assert.Equal(t, crypto.FromECDSAPub(&key.PublicKey), sig.PublicKey)
}

func TestEcdsaSigning(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	msg := crypto.Keccak256([]byte("hi hi"))

	sigBits, err := crypto.Sign(msg, key)
	require.Nil(t, err)
	sig := &Signature{
		Ownership: &Ownership{
			Type: KeyTypeSecp256k1,
		},
		Signature: sigBits,
	}
	sig.RestorePublicKey(msg)
	verified, err := sig.Valid(msg, nil)
	require.Nil(t, err)
	assert.True(t, verified)
}

func TestEcdsaSigningWithConditions(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	msg := crypto.Keccak256([]byte("hi hi"))

	sigBits, err := crypto.Sign(msg, key)
	require.Nil(t, err)
	sig := &Signature{
		Ownership: &Ownership{
			Type:       KeyTypeSecp256k1,
			Conditions: "false",
		},
		Signature: sigBits,
	}
	sig.RestorePublicKey(msg)
	verified, err := sig.Valid(msg, nil)
	require.Nil(t, err)
	// Conditions returned false so it should not verify
	assert.False(t, verified)

	sig.Conditions = "true"
	verified, err = sig.Valid(msg, nil)
	require.Nil(t, err)
	// Conditions are now TRUE so should verify
	assert.True(t, verified)
}

func TestHashPreimageConditions(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.Nil(t, err)
	msg := crypto.Keccak256([]byte("hi hi"))

	preImage := "secrets!"
	hsh := crypto.Keccak256Hash([]byte(preImage)).String()

	sigBits, err := crypto.Sign(msg, key)
	require.Nil(t, err)
	sig := &Signature{
		Ownership: &Ownership{
			Type:       KeyTypeSecp256k1,
			Conditions: fmt.Sprintf(`(== hashed-preimage "%s")`, hsh),
		},
		Signature: sigBits,
		PreImage:  "not the right one",
	}
	sig.RestorePublicKey(msg)
	verified, err := sig.Valid(msg, nil)
	require.Nil(t, err)
	// Conditions returned false so it should not verify
	assert.False(t, verified)

	sig.PreImage = preImage
	verified, err = sig.Valid(msg, nil)
	require.Nil(t, err)
	// Conditions are now TRUE so should verify
	assert.True(t, verified)
}

func BenchmarkConditions(b *testing.B) {
	key, err := crypto.GenerateKey()
	require.Nil(b, err)
	msg := crypto.Keccak256([]byte("hi hi"))

	preImage := "secrets!"
	hsh := crypto.Keccak256Hash([]byte(preImage)).String()

	sigBits, err := crypto.Sign(msg, key)
	require.Nil(b, err)
	sig := &Signature{
		Ownership: &Ownership{
			Type:       KeyTypeSecp256k1,
			Conditions: fmt.Sprintf(`(== hashed-preimage "%s")`, hsh),
		},
		Signature: sigBits,
		PreImage:  preImage,
	}
	sig.RestorePublicKey(msg)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err = sig.Valid(msg, nil)
	}
	require.Nil(b, err)
}
