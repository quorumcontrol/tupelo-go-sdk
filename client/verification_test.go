package client

import (
	"context"
	"testing"

	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/testnotarygroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyCurrentState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := testnotarygroup.NewTestSet(t, 3)
	ng := types.NewNotaryGroup("testverification")
	signer1 := types.NewLocalSigner(consensus.PublicKeyToEcdsaPub(&ts.PubKeys[0]), ts.SignKeys[0])
	signer2 := types.NewLocalSigner(consensus.PublicKeyToEcdsaPub(&ts.PubKeys[1]), ts.SignKeys[1])
	signer3 := types.NewLocalSigner(consensus.PublicKeyToEcdsaPub(&ts.PubKeys[2]), ts.SignKeys[2])

	ng.AddSigner(signer1)
	ng.AddSigner(signer2)
	ng.AddSigner(signer3)

	sw := &safewrap.SafeWrap{}
	currState := &signatures.CurrentState{
		Signature: &signatures.Signature{
			ObjectId: []byte("did:tupelo:fake"),
			NewTip:   sw.WrapObject(true).Cid().Bytes(),
		},
	}
	signable := consensus.GetSignable(currState.Signature)

	isValid, err := VerifyCurrentState(ctx, ng, currState)
	require.Nil(t, err)
	assert.False(t, isValid)

	// now go ahead and sign it
	signers := []uint32{1, 1, 1}
	currState.Signature.Signers = signers

	sig1, err := ts.SignKeys[0].Sign(signable)
	require.Nil(t, err)

	sig2, err := ts.SignKeys[1].Sign(signable)
	require.Nil(t, err)

	sig3, err := ts.SignKeys[2].Sign(signable)
	require.Nil(t, err)
	sig, err := bls.SumSignatures([][]byte{sig1, sig2, sig3})
	require.Nil(t, err)

	currState.Signature.Signature = sig

	isValid, err = VerifyCurrentState(ctx, ng, currState)
	require.Nil(t, err)
	assert.True(t, isValid)
}
