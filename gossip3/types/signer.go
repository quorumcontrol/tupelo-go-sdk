package types

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/p2p"
)

type Signer struct {
	ID     string
	DstKey *ecdsa.PublicKey
	VerKey *bls.VerKey

	SignKey *bls.SignKey
	Actor   *actor.PID
}

func NewLocalSigner(dstKey *ecdsa.PublicKey, signKey *bls.SignKey) *Signer {
	pubKey := consensus.BlsKeyToPublicKey(signKey.MustVerKey())
	return &Signer{
		ID:      consensus.PublicKeyToAddr(&pubKey),
		SignKey: signKey,
		VerKey:  signKey.MustVerKey(),
		DstKey:  dstKey,
	}
}

func NewRemoteSigner(dstKey *ecdsa.PublicKey, verKey *bls.VerKey) *Signer {
	pubKey := consensus.BlsKeyToPublicKey(verKey)
	return &Signer{
		ID:     consensus.PublicKeyToAddr(&pubKey),
		VerKey: verKey,
		DstKey: dstKey,
	}
}

// ActorName returns the default name that should be used for the spawned
// actor of this signer.
func (s *Signer) ActorName() string {
	return "tupelo-" + s.ID
}

func (s *Signer) ActorAddress(localKey *ecdsa.PublicKey) string {
	fromID, err := p2p.PeerFromEcdsaKey(localKey)
	if err != nil {
		panic(fmt.Sprintf("error getting peer from ecdsa key: %v", err))
	}
	id, err := p2p.PeerFromEcdsaKey(s.DstKey)
	if err != nil {
		panic(fmt.Sprintf("error getting peer from ecdsa key: %v", err))
	}
	return NewRoutableAddress(fromID.Pretty(), id.Pretty()).String()
}
