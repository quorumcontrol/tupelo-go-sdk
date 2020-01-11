package types

import (
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/safewrap"
)

func init() {
	cbornode.RegisterCborType(Checkpoint{})
}

type Checkpoint struct {
	Height           uint64
	AddBlockRequests []cid.Cid
	wrapped          *cbornode.Node
}

func (c *Checkpoint) CID() cid.Cid {
	return c.Wrapped().Cid()
}

func (c *Checkpoint) Wrapped() *cbornode.Node {
	if c.wrapped != nil {
		return c.wrapped
	}
	sw := safewrap.SafeWrap{}
	n := sw.WrapObject(c)
	c.wrapped = n
	return n
}

func (c *Checkpoint) Length() int {
	return len(c.AddBlockRequests)
}
