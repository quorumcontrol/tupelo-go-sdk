package types

import (
	"testing"

	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
	"github.com/quorumcontrol/tupelo-go-sdk/testnotarygroup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActorAddress(t *testing.T) {
	ts := testnotarygroup.NewTestSet(t, 2)
	signer1 := NewLocalSigner(ts.PubKeys[0], ts.SignKeys[0])
	signer2 := NewLocalSigner(ts.PubKeys[1], ts.SignKeys[1])
	require.NotEmpty(t, signer1.ActorAddress(signer2.DstKey))
	peer1, err := p2p.PeerFromEcdsaKey(ts.PubKeys[0])
	require.Nil(t, err)
	peer2, err := p2p.PeerFromEcdsaKey(ts.PubKeys[1])
	require.Nil(t, err)
	assert.Equal(t, peer2.Pretty()+"-"+peer1.Pretty(), signer1.ActorAddress(signer2.DstKey))
}
