package types

import (
	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/messages/build/go/gossip"
)

func init() {
	cbornode.RegisterCborType(gossip.Checkpoint{})
}

func WrapCheckpoint(c *gossip.Checkpoint) *WrappedCheckpoint {
	sw := safewrap.SafeWrap{}
	n := sw.WrapObject(c)

	return &WrappedCheckpoint{
		value:   c,
		wrapped: n,
	}
}

type WrappedCheckpoint struct {
	value   *gossip.Checkpoint
	wrapped *cbornode.Node
}

func (c *WrappedCheckpoint) CID() cid.Cid {
	return c.Wrapped().Cid()
}

func (c *WrappedCheckpoint) Value() *gossip.Checkpoint {
	return c.value
}

func (c *WrappedCheckpoint) Wrapped() *cbornode.Node {
	return c.wrapped
}

func (c *WrappedCheckpoint) Length() int {
	return len(c.value.AddBlockRequests)
}

func (c *WrappedCheckpoint) AddBlockRequests() [][]byte {
	return c.value.AddBlockRequests
}
