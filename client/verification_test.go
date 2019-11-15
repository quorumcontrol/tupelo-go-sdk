package client

import (
	"context"
	"testing"

	sigfuncs "github.com/quorumcontrol/tupelo-go-sdk/signatures"

	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
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
	signer1 := types.NewLocalSigner(ts.PubKeys[0], ts.SignKeys[0])
	signer2 := types.NewLocalSigner(ts.PubKeys[1], ts.SignKeys[1])
	signer3 := types.NewLocalSigner(ts.PubKeys[2], ts.SignKeys[2])

	ng.AddSigner(signer1)
	ng.AddSigner(signer2)
	ng.AddSigner(signer3)

	sw := &safewrap.SafeWrap{}
	currState := &signatures.TreeState{
		ObjectId:  []byte("did:tupelo:fake"),
		NewTip:    sw.WrapObject(true).Cid().Bytes(),
		Signature: &signatures.Signature{},
	}
	signable := consensus.GetSignable(currState)

	isValid, err := VerifyCurrentState(ctx, ng, currState)
	require.Nil(t, err)
	assert.False(t, isValid)

	// now go ahead and sign it
	sig1, err := sigfuncs.BLSSign(ts.SignKeys[0], signable, 3, 0)
	require.Nil(t, err)

	sig2, err := sigfuncs.BLSSign(ts.SignKeys[1], signable, 3, 1)
	require.Nil(t, err)

	sig3, err := sigfuncs.BLSSign(ts.SignKeys[2], signable, 3, 2)
	require.Nil(t, err)

	sig, err := sigfuncs.AggregateBLSSignatures([]*signatures.Signature{sig1, sig2, sig3})
	require.Nil(t, err)

	currState.Signature = sig

	isValid, err = VerifyCurrentState(ctx, ng, currState)
	require.Nil(t, err)
	assert.True(t, isValid)
}
