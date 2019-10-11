// +build wasm

package jsclient

import (
	"context"
	"crypto/ecdsa"
	"syscall/js"

	"github.com/quorumcontrol/messages/build/go/config"

	"github.com/quorumcontrol/messages/build/go/signatures"

	"github.com/ipfs/go-cid"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"

	"github.com/quorumcontrol/tupelo-go-sdk/wasm/jsstore"

	"github.com/gogo/protobuf/proto"

	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/client"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"

	"github.com/quorumcontrol/tupelo-go-sdk/wasm/pubsub"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/helpers"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/then"
)

// JSClient is a javascript bridging client
type JSClient struct {
	pubsub      remote.PubSub
	notaryGroup *types.NotaryGroup
}

func New(pubsub *pubsub.PubSubBridge, humanConfig *config.NotaryGroup) *JSClient {
	ngConfig, err := types.HumanConfigToConfig(humanConfig)
	if err != nil {
		panic(errors.Wrap(err, "error decoding human config"))
	}

	ng := types.NewNotaryGroupFromConfig(ngConfig)

	wrapped := remote.NewWrappedPubsub(pubsub)

	return &JSClient{
		pubsub:      wrapped,
		notaryGroup: ng,
	}
}

func jsTransactionsToTransactions(jsTransactions js.Value) ([]*transactions.Transaction, error) {
	transLength := jsTransactions.Length()
	transBits := make([][]byte, transLength)
	for i := 0; i < transLength; i++ {
		jsVal := jsTransactions.Index(i)
		transBits[i] = helpers.JsBufferToBytes(jsVal)
	}

	trans := make([]*transactions.Transaction, len(transBits))
	for i, bits := range transBits {
		tran := &transactions.Transaction{}
		err := proto.Unmarshal(bits, tran)
		if err != nil {
			return nil, errors.Wrap(err, "error unmarshaling")
		}
		trans[i] = tran
	}
	return trans, nil
}

func jsKeyBitsToPrivateKey(jsKeyBits js.Value) (*ecdsa.PrivateKey, error) {
	keybits := helpers.JsBufferToBytes(jsKeyBits)
	return crypto.ToECDSA(keybits)
}

func (jsc *JSClient) PlayTransactions(blockService js.Value, jsKeyBits js.Value, tip js.Value, jsTransactions js.Value) interface{} {
	t := then.New()
	go func() {
		trans, err := jsTransactionsToTransactions(jsTransactions)
		if err != nil {
			t.Reject(err.Error())
			return
		}

		key, err := jsKeyBitsToPrivateKey(jsKeyBits)
		if err != nil {
			t.Reject(err.Error())
			return
		}

		wrappedStore := jsstore.New(blockService)

		tip, err := helpers.JsCidToCid(tip)
		if err != nil {
			t.Reject(err.Error())
			return
		}

		resp, err := jsc.playTransactions(wrappedStore, tip, key, trans)
		if err != nil {
			t.Reject(err.Error())
			return
		}

		currState := &signatures.CurrentState{
			Signature: &resp.Signature,
		}

		respBits, err := proto.Marshal(currState)
		if err != nil {
			t.Reject(err.Error())
			return
		}
		t.Resolve(helpers.SliceToJSArray(respBits))
	}()

	return t
}

func (jsc *JSClient) playTransactions(store nodestore.DagStore, tip cid.Cid, treeKey *ecdsa.PrivateKey, transactions []*transactions.Transaction) (*consensus.AddBlockResponse, error) {
	ctx := context.TODO()

	cTree, err := chaintree.NewChainTree(
		ctx,
		dag.NewDag(ctx, tip, store),
		nil,
		consensus.DefaultTransactors,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error creating chaintree")
	}

	tree := consensus.NewSignedChainTreeFromChainTree(cTree)
	c := client.New(jsc.notaryGroup, tree.MustId(), jsc.pubsub)
	defer c.Stop()

	var remoteTip cid.Cid
	if !tree.IsGenesis() {
		remoteTip = tree.Tip()
	}

	return c.PlayTransactions(tree, treeKey, &remoteTip, transactions)
}

func VerifyCurrentState(humanConfig *config.NotaryGroup, state *signatures.CurrentState) *then.Then {
	t := then.New()
	go func() {
		ngConfig, err := types.HumanConfigToConfig(humanConfig)
		if err != nil {
			panic(errors.Wrap(err, "error decoding human config"))
		}

		ng := types.NewNotaryGroupFromConfig(ngConfig)
		valid, err := client.VerifyCurrentState(context.TODO(), ng, state)
		if err != nil {
			t.Reject(err.Error())
			return
		}
		t.Resolve(valid)
		return
	}()

	return t
}

func GenerateKey() *then.Then {
	t := then.New()
	go func() {
		key, err := crypto.GenerateKey()
		if err != nil {
			t.Reject(err.Error())
			return
		}
		privateBits := helpers.SliceToJSArray(crypto.FromECDSA(key))
		publicBits := helpers.SliceToJSArray(crypto.FromECDSAPub(&key.PublicKey))
		jsArray := js.Global().Get("Array").New(privateBits, publicBits)
		t.Resolve(jsArray)
	}()
	return t
}

func KeyFromPrivateBytes(jsBits js.Value) *then.Then {
	t := then.New()
	go func() {
		key, err := jsKeyBitsToPrivateKey(jsBits)
		if err != nil {
			t.Reject(err.Error())
			return
		}
		privateBits := helpers.SliceToJSArray(crypto.FromECDSA(key))
		publicBits := helpers.SliceToJSArray(crypto.FromECDSAPub(&key.PublicKey))
		jsArray := js.Global().Get("Array").New(privateBits, publicBits)
		t.Resolve(jsArray)
	}()
	return t
}

func PassPhraseKey(jsPhrase, jsSalt js.Value) *then.Then {
	t := then.New()
	go func() {
		phrase := helpers.JsBufferToBytes(jsPhrase)
		salt := helpers.JsBufferToBytes(jsSalt)
		key, err := consensus.PassPhraseKey(phrase, salt)
		if err != nil {
			t.Reject(err.Error())
			return
		}
		privateBits := helpers.SliceToJSArray(crypto.FromECDSA(key))
		publicBits := helpers.SliceToJSArray(crypto.FromECDSAPub(&key.PublicKey))
		jsArray := js.Global().Get("Array").New(privateBits, publicBits)
		t.Resolve(jsArray)
	}()
	return t
}

func jsToPubKey(jsPubKeyBits js.Value) (*ecdsa.PublicKey, error) {
	pubbits := helpers.JsBufferToBytes(jsPubKeyBits)
	pubKey, err := crypto.UnmarshalPubkey(pubbits)
	if err != nil {
		return nil, errors.Wrap(err, "error unmarshaling public key")
	}
	return pubKey, nil
}

func EcdsaPubkeyToDid(jsPubKeyBits js.Value) *then.Then {
	t := then.New()
	go func() {
		pubKey, err := jsToPubKey(jsPubKeyBits)
		if err != nil {
			t.Reject(err.Error())
			return
		}
		t.Resolve(consensus.EcdsaPubkeyToDid(*pubKey))
	}()
	return t
}

func EcdsaPubkeyToAddress(jsPubKeyBits js.Value) *then.Then {
	t := then.New()
	go func() {
		pubKey, err := jsToPubKey(jsPubKeyBits)
		if err != nil {
			t.Reject(err.Error())
			return
		}
		t.Resolve(crypto.PubkeyToAddress(*pubKey).String())
	}()
	return t
}

// NewEmptyTree is a little departurue from the normal Go SDK, it's a helper for JS to create
// a new blank ChainTree given a private key. It will populate the node store and return the tip
// of the new Dag (so javascript can reconstitute the chaintree on its side.)
func NewEmptyTree(jsBlockService js.Value, jsPublicKeyBits js.Value) *then.Then {
	t := then.New()
	go func() {
		store := jsstore.New(jsBlockService)
		treeKey, err := crypto.UnmarshalPubkey(helpers.JsBufferToBytes(jsPublicKeyBits))
		if err != nil {
			t.Reject(err.Error())
			return
		}
		did := consensus.EcdsaPubkeyToDid(*treeKey)
		dag := consensus.NewEmptyTree(did, store)
		t.Resolve(helpers.CidToJSCID(dag.Tip))
	}()
	return t
}

func JsConfigToHumanConfig(jsBits js.Value) (*config.NotaryGroup, error) {
	bits := helpers.JsBufferToBytes(jsBits)
	config := &config.NotaryGroup{}
	err := proto.Unmarshal(bits, config)
	return config, err
}

// func TokenPayloadForTransaction(chain *chaintree.ChainTree, tokenName *TokenName, sendTokenTxId string, sendTxSig *signatures.Signature) (*transactions.TokenPayload, error) {
func TokenPayloadForTransaction(jsBlockService js.Value, jsTip js.Value, tokenName js.Value, sendTokenTxId js.Value, jsSendTxSigBits js.Value) *then.Then {
	t := then.New()
	ctx := context.TODO()
	go func() {
		wrappedStore := jsstore.New(jsBlockService)
		tip, err := helpers.JsCidToCid(jsTip)
		if err != nil {
			t.Reject(err.Error())
			return
		}

		tree, err := chaintree.NewChainTree(
			ctx,
			dag.NewDag(ctx, tip, wrappedStore),
			nil,
			consensus.DefaultTransactors,
		)
		if err != nil {
			t.Reject(err.Error())
			return
		}

		sigBits := helpers.JsBufferToBytes(jsSendTxSigBits)
		sig := &signatures.Signature{}
		err = proto.Unmarshal(sigBits, sig)
		if err != nil {
			t.Reject(err.Error())
			return
		}

		canonicalTokenName := consensus.TokenNameFromString(tokenName.String())

		payload, err := consensus.TokenPayloadForTransaction(tree, &canonicalTokenName, sendTokenTxId.String(), sig)
		if err != nil {
			t.Reject(err.Error())
			return
		}

		bits, err := proto.Marshal(payload)
		if err != nil {
			t.Reject(err.Error())
			return
		}
		t.Resolve(helpers.SliceToJSArray(bits))
	}()

	return t
}
