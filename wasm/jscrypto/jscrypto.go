// +build wasm

package jscrypto

import (
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/helpers"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/then"
)

func Sign(msg []byte, keyBits []byte) *then.Then {
	t := then.New()
	go func() {
		hsh := sha256.Sum256(msg)
		key, err := crypto.ToECDSA(keyBits)
		if err != nil {
			t.Reject(err.Error())
			return
		}
		bits, err := crypto.Sign(hsh[:], key)
		if err != nil {
			t.Reject(err.Error())
			return
		}
		t.Resolve(helpers.SliceToJSBuffer(bits))
	}()
	return t
}
