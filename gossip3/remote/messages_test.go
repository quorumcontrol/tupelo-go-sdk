package remote

import (
	"bytes"
	"testing"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinylib/msgp/msgp"
)

type nullActor struct{}

func (na *nullActor) Receive(_ actor.Context) {}
func newNullActor() actor.Actor               { return &nullActor{} }

func TestWorksWithActorPID(t *testing.T) {
	act, err := actor.EmptyRootContext.SpawnNamed(actor.PropsFromProducer(newNullActor), "test-actorPID-encode")
	defer act.Stop()
	require.Nil(t, err)
	wd := &WireDelivery{
		Target: messages.ToActorPid(act),
	}

	t.Run("using MarshalMsg", func(t *testing.T) {
		marshaled, err := wd.MarshalMsg(nil)
		require.Nil(t, err)
		var newWd WireDelivery
		_, err = newWd.UnmarshalMsg(marshaled)
		require.Nil(t, err)

		recoveredPid := messages.FromActorPid(newWd.Target)
		assert.True(t, act.Equal(recoveredPid))
	})

	t.Run("using streaming interface", func(t *testing.T) {
		buf := make([]byte, 0)
		bufWriter := bytes.NewBuffer(buf)
		writer := msgp.NewWriter(bufWriter)
		err := wd.EncodeMsg(writer)
		require.Nil(t, err)

		err = writer.Flush()
		require.Nil(t, err)

		reader := msgp.NewReader(bufWriter)
		var newWd WireDelivery
		err = newWd.DecodeMsg(reader)
		require.Nil(t, err)
		recoveredPid := messages.FromActorPid(newWd.Target)
		assert.True(t, act.Equal(recoveredPid))
	})
}
