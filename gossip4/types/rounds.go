package types

import (
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/v2/build/go/signatures"
)

func init() {
	cbornode.RegisterCborType(CompletedRound{})
	cbornode.RegisterCborType(RoundConfirmation{})
}

type CompletedRound struct {
	Height     uint64
	Checkpoint cid.Cid
	State      cid.Cid
	wrapped    *cbornode.Node
}

func (r *CompletedRound) CID() cid.Cid {
	return r.Wrapped().Cid()
}

func (r *CompletedRound) Wrapped() *cbornode.Node {
	if r.wrapped != nil {
		return r.wrapped
	}
	sw := safewrap.SafeWrap{}
	n := sw.WrapObject(r)
	r.wrapped = n
	return n
}

type RoundConfirmation struct {
	CompletedRound cid.Cid
	Signature      *signatures.Signature
	wrapped        *cbornode.Node
}

func (rc *RoundConfirmation) Data() []byte {
	return rc.Wrapped().RawData()
}

func (rc *RoundConfirmation) Wrapped() *cbornode.Node {
	if rc.wrapped != nil {
		return rc.wrapped
	}
	sw := safewrap.SafeWrap{}
	n := sw.WrapObject(rc)
	rc.wrapped = n
	return n
}
