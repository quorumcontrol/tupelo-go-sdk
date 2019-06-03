package types

import (
	"context"
	"testing"

	"github.com/Workiva/go-datastructures/bitarray"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/testnotarygroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerKeysOf(t *testing.T) {
	ts := testnotarygroup.NewTestSet(t, 5)
	ng := NewNotaryGroup("TestVerKeysOf")

	for i, signKey := range ts.SignKeys {
		signer := NewLocalSigner(consensus.PublicKeyToEcdsaPub(&ts.PubKeys[i]), signKey)
		ng.AddSigner(signer)
	}

	arry := bitarray.NewSparseBitArray()
	err := arry.SetBit(0)
	require.Nil(t, err)
	err = arry.SetBit(3)
	require.Nil(t, err)

	resp, err := ng.VerKeysOf(arry)
	require.Nil(t, err)
	require.ElementsMatch(t, resp, [][]byte{ng.SignerAtIndex(0).VerKey.Bytes(), ng.SignerAtIndex(3).VerKey.Bytes()})
}

func TestNotaryGroupBlockValidators(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// notary group with default config:
	ng := NewNotaryGroup("TestNotaryGroupBlockValidators")

	validators, err := ng.BlockValidators(ctx)
	require.Nil(t, err)
	assert.NotEmpty(t, validators)
	assert.Len(t, validators, len(ng.config.ValidatorGenerators))
}
