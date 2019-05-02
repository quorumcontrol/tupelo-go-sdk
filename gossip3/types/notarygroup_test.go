package types

import (
	"testing"

	"github.com/Workiva/go-datastructures/bitarray"
	"github.com/quorumcontrol/tupelo-go-sdk/testnotarygroup"
	"github.com/stretchr/testify/require"
)

func TestVerKeysOf(t *testing.T) {
	ts := testnotarygroup.NewTestSet(t, 5)
	ng := NewNotaryGroup("TestVerKeysOf")

	for i, signKey := range ts.SignKeys {
		signer := NewLocalSigner(ts.PubKeys[i].ToEcdsaPub(), signKey)
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
