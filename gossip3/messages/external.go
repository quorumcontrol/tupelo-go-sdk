//go:generate msgp

package messages

import (
	"encoding/binary"
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-client/gossip3/serializer"
)

func init() {
	serializer.RegisterEncodable(Ping{})
	serializer.RegisterEncodable(Pong{})
	serializer.RegisterEncodable(Store{})
	serializer.RegisterEncodable(CurrentState{})
	serializer.RegisterEncodable(Signature{})
	serializer.RegisterEncodable(Transaction{})
	serializer.RegisterEncodable(GetTip{})
	serializer.RegisterEncodable(ActorPID{})
	serializer.RegisterEncodable(TipSubscription{})
}

type DestinationSettable interface {
	SetDestination(*ActorPID)
	GetDestination() *ActorPID
}

type Ping struct {
	Msg string
}

type Pong struct {
	Msg string
}

type Store struct {
	Key        []byte
	Value      []byte
	SkipNotify bool `msg:"-"`
}

type TipSubscription struct {
	Unsubscribe bool
	ObjectID    []byte
	TipValue    []byte
}

type CurrentState struct {
	Signature *Signature
}

func (cs *CurrentState) CommittedKey() []byte {
	return []byte(cs.Signature.ConflictSetID())
}

func (cs *CurrentState) CurrentKey() []byte {
	return append(cs.Signature.ObjectID)
}

func (cs *CurrentState) MustBytes() []byte {
	bits, err := cs.MarshalMsg(nil)
	if err != nil {
		panic(fmt.Errorf("error marshaling current state: %v", err))
	}
	return bits
}

type GetTip struct {
	ObjectID []byte
}

type Signature struct {
	TransactionID []byte
	ObjectID      []byte
	PreviousTip   []byte
	Height        uint64
	NewTip        []byte
	View          uint64
	Cycle         uint64
	Signers       []byte // this is a marshaled BitArray from github.com/Workiva/go-datastructures
	Signature     []byte
}

func (sig *Signature) GetSignable() []byte {
	return append(append(sig.ObjectID, append(sig.PreviousTip, sig.NewTip...)...), append(uint64ToBytes(sig.View), uint64ToBytes(sig.Cycle)...)...)
}

func (sig *Signature) ConflictSetID() string {
	return ConflictSetID(sig.ObjectID, sig.Height)
}

func uint64ToBytes(id uint64) []byte {
	a := make([]byte, 8)
	binary.BigEndian.PutUint64(a, id)
	return a
}

type Transaction struct {
	ObjectID    []byte
	PreviousTip []byte
	Height      uint64
	NewTip      []byte
	Payload     []byte
	State       [][]byte
}

func (t *Transaction) ConflictSetID() string {
	return ConflictSetID(t.ObjectID, t.Height)
}

func ConflictSetID(objectID []byte, height uint64) string {
	return string(crypto.Keccak256(append(objectID, uint64ToBytes(height)...)))
}

type ActorPID struct {
	Address string
	Id      string
}

func ToActorPid(a *actor.PID) *ActorPID {
	if a == nil {
		return nil
	}
	return &ActorPID{
		Address: a.Address,
		Id:      a.Id,
	}
}

func FromActorPid(a *ActorPID) *actor.PID {
	return actor.NewPID(a.Address, a.Id)
}
