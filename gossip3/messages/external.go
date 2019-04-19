//go:generate msgp

package messages

import (
	"encoding/binary"
	"fmt"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-client/tracing"
)

func init() {
	RegisterMessage(&Error{})
	RegisterMessage(&Ping{})
	RegisterMessage(&Pong{})
	RegisterMessage(&Store{})
	RegisterMessage(&CurrentState{})
	RegisterMessage(&Signature{})
	RegisterMessage(&Transaction{})
	RegisterMessage(&GetTip{})
	RegisterMessage(&ActorPID{})
}

type DestinationSettable interface {
	SetDestination(*ActorPID)
	GetDestination() *ActorPID
}

// Error represents an error message.
type Error struct {
	Source string
	Code   int
	Memo   string
}

func (Error) TypeCode() int8 {
	return -1
}

type Ping struct {
	tracing.ContextHolder `msg:"-"`
	Msg                   string
}

func (Ping) TypeCode() int8 {
	return -2
}

type Pong struct {
	Msg string
}

func (Pong) TypeCode() int8 {
	return -3
}

type Store struct {
	Key        []byte
	Value      []byte
	SkipNotify bool `msg:"-"`
}

func (Store) TypeCode() int8 {
	return -4
}

type CurrentState struct {
	Signature *Signature
}

func (CurrentState) TypeCode() int8 {
	return -6
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

func (GetTip) TypeCode() int8 {
	return -7
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
	Type          string
}

func (Signature) TypeCode() int8 {
	return -8
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

func (Transaction) TypeCode() int8 {
	return -9
}

func (t *Transaction) ID() []byte {
	bits, err := t.MarshalMsg(nil)
	if err != nil {
		panic(fmt.Errorf("error marshaling: %v", err))
	}
	return crypto.Keccak256(bits)
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

func (ActorPID) TypeCode() int8 {
	return -10
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
