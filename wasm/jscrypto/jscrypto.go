// +build wasm

package jscrypto

import (
	"context"
	"fmt"
	jshelpers "github.com/quorumcontrol/tupelo-go-sdk/wasm/helpers"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/then"
	"syscall/js"

	"github.com/quorumcontrol/messages/v2/build/go/signatures"
	sigfuncs "github.com/quorumcontrol/tupelo-go-sdk/signatures"

	"github.com/ethereum/go-ethereum/crypto"
)

func JSSignMessage(this js.Value, args []js.Value) interface{} {
	keyBits := jshelpers.JsBufferToBytes(args[0])
	msgBits := jshelpers.JsBufferToBytes(args[1])
	t := then.New()
	sigBits, err := SignMessage(keyBits, msgBits)
	if err != nil {
		t.Reject(err.Error())
		return t
	}

	t.Resolve(jshelpers.SliceToJSArray(sigBits))
	return t
}

func JSVerifyMessage(this js.Value, args []js.Value) interface{} {
	addr := args[0].String()
	msgBits := jshelpers.JsBufferToBytes(args[1])
	sigBits := jshelpers.JsBufferToBytes(args[2])
	t := then.New()
	isValid, err := VerifySignature(addr, msgBits, sigBits)
	if err != nil {
		t.Reject(err.Error())
		return t
	}

	t.Resolve(isValid)
	return t
}

func SignMessage(keyBits []byte, message []byte) ([]byte, error) {
	ctx := context.TODO()

	key, err := crypto.ToECDSA(keyBits)
	if err != nil {
		return nil, fmt.Errorf("error converting key: %w", err)
	}
	hsh := crypto.Keccak256(message)

	sig, err := sigfuncs.EcdsaSign(ctx, key, hsh)
	if err != nil {
		return nil, fmt.Errorf("error signing: %w", err)
	}

	bits, err := sig.Marshal()
	if err != nil {
		return nil, fmt.Errorf("error marshaling: %w", err)
	}
	return bits, nil
}

func VerifySignature(address string, message []byte, signatureBits []byte) (bool, error) {
	ctx := context.TODO()

	sig := &signatures.Signature{}
	err := sig.Unmarshal(signatureBits)
	if err != nil {
		return false, fmt.Errorf("error unmarshaling signature: %w", err)
	}

	hsh := crypto.Keccak256(message)

	err = sigfuncs.RestoreEcdsaPublicKey(ctx, sig, hsh)
	if err != nil {
		return false, fmt.Errorf("error restoring public key: %v", err)
	}

	sigAddr, err := sigfuncs.Address(sig.Ownership)
	if err != nil {
		return false, fmt.Errorf("error getting address from signature: %w", err)
	}
	if sigAddr.String() != address {
		return false, fmt.Errorf("unsigned by address %s != %s", sigAddr.String(), address)
	}

	return sigfuncs.Valid(ctx, sig, hsh, nil) // TODO: maybe we want to have a custom scope here?
}
