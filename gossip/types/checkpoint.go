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

func WrapCheckpoint(c *gossip.Checkpoint) *wrappedCheckpoint {
	sw := safewrap.SafeWrap{}
	n := sw.WrapObject(c)

	return &wrappedCheckpoint{
		value:   c,
		wrapped: n,
	}
}

type wrappedCheckpoint struct {
	value   *gossip.Checkpoint
	wrapped *cbornode.Node
}

func (c *wrappedCheckpoint) CID() cid.Cid {
	return c.Wrapped().Cid()
}

func (c *wrappedCheckpoint) Value() *gossip.Checkpoint {
	return c.value
}

func (c *wrappedCheckpoint) Wrapped() *cbornode.Node {
	return c.wrapped
}

func (c *wrappedCheckpoint) Length() int {
	return len(c.value.AddBlockRequests)
}
