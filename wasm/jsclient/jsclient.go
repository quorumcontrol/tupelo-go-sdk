// +build wasm

package jsclient

import (
	"syscall/js"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
	"github.com/quorumcontrol/tupelo-go-sdk/wasm/then"
)

var hardcodedHumanConfig = &types.HumanConfig{
	ID: "localrun",
	Signers: []types.HumanPublicKeySet{
		{
			VerKeyHex:  "0x753a9c092a0268b597035c06970a5f6347ffc223fc0e0df3be20a4f0d2265be64df855badf98c7319c80ee06ac510af068696456331d93e2bdab51350e1089df29d21c6da14aa6365bf8314c5110af69caa341c731ed308c8e4401f6d93e38ce2354c183f8eff143b5e7e9e8c3470c13f1f04640baa7d66e3a2ff0630dbd5c60",
			DestKeyHex: "0x0401b8b0fea7cdf569c59ad74cca3e45fbdb38d5992b3203a61355b1c172c09963c89bd806a3179f324796c4a4d2e566811c5f4e75b518a11a41a48d95af654894",
		},
		{
			VerKeyHex:  "0x09e09d6126c20b1771791f38a44ae2decd6902af386999b08937a3329bc6cdd43440ff2761f95859d282bbf45e5fdbbf7fa279c5608abc76176b450e332118a86d37eabf94283bef4578ea2d7c57f16f90a87945af8452e86f579fa8a725c7106409f1e6d8c4362b3b495663428a4d2bb41fa321e85e8887c7928677652da3e2",
			DestKeyHex: "0x040c4e57d4eb4d1733915f78fe419922824d9b24da686aa257a7396e94a90e46bc73abd1a6a6b9360c77d169875b15f8f0efba0016484082a68a3fc8b88a1dbb01",
		},
		{
			VerKeyHex:  "0x5f637890ddbbd1f8ae57e2777688d0b076443c7e55f8f908a9ec9d5f5fad5584323ec51cbcacaaa55dcbdfcb02027c3dc49bd008a97aa4d2714d03bc7b7b5de96f3e32ee75c04768cc8f833b0ba5cd3b0a0c9a89ba33f60364e9d2f364db3aaa6421d467db7d682af09b35b83abdc1dfb838a45fed17766c81ed7742f9227e3e",
			DestKeyHex: "0x04bbd6e887c52fddc2d888fcc2b8b366df76746e2732b1e897ae94634288352c992cd405206dc66a4ffd14a80742a01451b49a7d475553c190b440848b663d0a03",
		},
	},
}

// JSClient is a javascript bridging client
type JSClient struct {
}

func New() *JSClient {
	return &JSClient{}
}

func (jsc *JSClient) GenerateKey() *then.Then {
	t := then.New()
	go func() {
		key, err := crypto.GenerateKey()
		if err != nil {
			t.Reject(err.Error())
			return
		}
		bits := js.TypedArrayOf(crypto.FromECDSA(key))
		t.Resolve(bits)
	}()
	return t
}
