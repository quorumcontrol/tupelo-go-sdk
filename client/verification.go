package client

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/bls"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"golang.org/x/xerrors"
)

func VerifyCurrentState(ctx context.Context, group *types.NotaryGroup, state *signatures.TreeState) (bool, error) {
	sp, _ := opentracing.StartSpanFromContext(ctx, "verifyCurrentState")
	defer sp.Finish()

	if state.Signature == nil {
		return false, xerrors.Errorf("no signature supplied")
	}

	var verKeys [][]byte

	signers := group.AllSigners()
	var signerCount uint64
	for i, cnt := range state.Signature.Signers {
		if cnt > 0 {
			signerCount++
			verKey := signers[i].VerKey.Bytes()
			newKeys := make([][]byte, cnt)
			for j := uint32(0); j < cnt; j++ {
				newKeys[j] = verKey
			}
			verKeys = append(verKeys, newKeys...)
		}
	}
	if signerCount < group.QuorumCount() {
		return false, nil
	}
	isVerified, err := bls.VerifyMultiSig(state.Signature.Signature, consensus.GetSignable(state), verKeys)
	if err != nil {
		sp.SetTag("error", true)
		return false, xerrors.Errorf("error verifying: %w", err)
	}
	sp.SetTag("verified", isVerified)
	return true, nil
}
