package consensus

import (
	"testing"

	"github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
)

func mustWrap(t testing.TB, obj interface{}) *cbornode.Node {
	sw := safewrap.SafeWrap{}
	wrapped := sw.WrapObject(obj)
	require.Nil(t, sw.Err)
	return wrapped
}

func treeMapToNodes(t testing.TB, tree map[string]interface{}) []*cbornode.Node {
	var nodes []*cbornode.Node

	wrappable := make(map[string]interface{}, len(tree))

	for key, val := range tree {
		switch v := val.(type) {
		case map[string]interface{}:
			nodes = treeMapToNodes(t, v)
			wrappable[key] = nodes[0].Cid()
		case []map[string]interface{}:
			cids := make([]cid.Cid, len(v))
			for i, m := range v {
				nodes = append(nodes, treeMapToNodes(t, m)...)
				cids[i] = nodes[len(nodes) - 1].Cid()
			}
			wrappable[key] = cids
		default:
			wrappable[key] = v
		}
	}

	wrapped := mustWrap(t, wrappable)
	nodes = append([]*cbornode.Node{wrapped}, nodes...)

	return nodes
}

// Turns arbitrarily-nested map[string]interface{}'s into DAGs to aid test tree
// construction.
func NewTestTree(t testing.TB, tree map[string]interface{}) *dag.Dag {
	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	treeNodes := treeMapToNodes(t, tree)

	dagTree, err := dag.NewDagWithNodes(nodeStore, treeNodes...)
	require.Nil(t, err)

	return dagTree
}

func TestNewTreeLedger(t *testing.T) {
	ledger := NewTreeLedger(nil, "test-token")
	assert.Equal(t, "test-token", ledger.tokenName)
}

func TestTreeLedger_tokenPath(t *testing.T) {
	ledger := NewTreeLedger(nil, "test-token")
	tokenPath, err := ledger.tokenPath()
	require.Nil(t, err)
	assert.Equal(t, []string{"_tupelo", "tokens", "test-token"}, tokenPath)
}

func TestTreeLedger_tokenTransactionCidsForType(t *testing.T) {
	testTreeNodes := map[string]interface{}{
		"_tupelo": map[string]interface{}{
			"tokens": map[string]interface{}{
				"test-token": map[string]interface{}{
					TokenMintLabel: []map[string]interface{}{
						{"amount": 1},
						{"amount": 2},
						{"amount": 3},
					},
					TokenSendLabel: []map[string]interface{}{
						{"amount": 4},
						{"amount": 2},
					},
					TokenReceiveLabel: []map[string]interface{}{
						{"amount": 10},
					},
				},
			},
		},
	}
	testTree := NewTestTree(t, testTreeNodes)

	ledger := NewTreeLedger(testTree, "test-token")

	mintTransactions, err := ledger.tokenTransactionCidsForType(TokenMintLabel)
	require.Nil(t, err)
	assert.Equal(t, 3, len(mintTransactions))

	sendTransactions, err := ledger.tokenTransactionCidsForType(TokenSendLabel)
	require.Nil(t, err)
	assert.Equal(t, 2, len(sendTransactions))

	receiveTransactions, err := ledger.tokenTransactionCidsForType(TokenReceiveLabel)
	require.Nil(t, err)
	assert.Equal(t, 1, len(receiveTransactions))
}

func TestTreeLedger_tokenTransactionCids(t *testing.T) {
	testTreeNodes := map[string]interface{}{
		"_tupelo": map[string]interface{}{
			"tokens": map[string]interface{}{
				"test-token": map[string]interface{}{
					TokenMintLabel: []map[string]interface{}{
						{"amount": 1},
						{"amount": 2},
						{"amount": 3},
					},
					TokenSendLabel: []map[string]interface{}{
						{"amount": 4},
						{"amount": 2},
					},
					TokenReceiveLabel: []map[string]interface{}{
						{"amount": 10},
					},
				},
			},
		},
	}
	testTree := NewTestTree(t, testTreeNodes)

	ledger := NewTreeLedger(testTree, "test-token")

	transactions, err := ledger.tokenTransactionCids()
	require.Nil(t, err)

	assert.Equal(t, 3, len(transactions))
	assert.Equal(t, 3, len(transactions[TokenMintLabel]))
	assert.Equal(t, 2, len(transactions[TokenSendLabel]))
	assert.Equal(t, 1, len(transactions[TokenReceiveLabel]))
}

func TestTreeLedger_sumTokenTransactions(t *testing.T) {
	testTreeNodes := map[string]interface{}{
		"_tupelo": map[string]interface{}{
			"tokens": map[string]interface{}{
				"test-token": map[string]interface{}{
					TokenMintLabel: []map[string]interface{}{
						{"amount": 1},
						{"amount": 2},
						{"amount": 3},
					},
					TokenSendLabel: []map[string]interface{}{
						{"amount": 4},
						{"amount": 20},
					},
					TokenReceiveLabel: []map[string]interface{}{
						{"amount": 50},
					},
				},
			},
		},
	}
	testTree := NewTestTree(t, testTreeNodes)

	ledger := NewTreeLedger(testTree, "test-token")

	transactionCids, err := ledger.tokenTransactionCidsForType(TokenMintLabel)
	require.Nil(t, err)
	sum, err := ledger.sumTokenTransactions(transactionCids)
	require.Nil(t, err)

	assert.Equal(t, uint64(6), sum)

	transactionCids, err = ledger.tokenTransactionCidsForType(TokenSendLabel)
	require.Nil(t, err)
	sum, err = ledger.sumTokenTransactions(transactionCids)
	require.Nil(t, err)

	assert.Equal(t, uint64(24), sum)

	transactionCids, err = ledger.tokenTransactionCidsForType(TokenReceiveLabel)
	require.Nil(t, err)
	sum, err = ledger.sumTokenTransactions(transactionCids)
	require.Nil(t, err)

	assert.Equal(t, uint64(50), sum)
}

func TestTreeLedger_Balance(t *testing.T) {
	testTreeNodes := map[string]interface{}{
		"_tupelo": map[string]interface{}{
			"tokens": map[string]interface{}{
				"test-token": map[string]interface{}{
					TokenBalanceLabel: uint64(32),
					TokenMintLabel:    []map[string]interface{}{
						{"amount": 1},
						{"amount": 2},
						{"amount": 3},
					},
					TokenSendLabel:    []map[string]interface{}{
						{"amount": 4},
						{"amount": 20},
					},
					TokenReceiveLabel: []map[string]interface{}{
						{"amount": 50},
					},
				},
			},
		},
	}
	testTree := NewTestTree(t, testTreeNodes)

	ledger := NewTreeLedger(testTree, "test-token")

	balance, err := ledger.Balance()
	require.Nil(t, err)

	assert.Equal(t, uint64(32), balance)
}

func TestTreeLedger_TokenExists(t *testing.T) {
	testTreeNodes := map[string]interface{}{
		"_tupelo": map[string]interface{}{
			"tokens": map[string]interface{}{
				"test-token": map[string]interface{}{
					TokenMintLabel: []map[string]interface{}{
						{"amount": 1},
						{"amount": 2},
						{"amount": 3},
					},
					TokenSendLabel: []map[string]interface{}{
						{"amount": 4},
						{"amount": 2},
					},
					TokenReceiveLabel: []map[string]interface{}{
						{"amount": 10},
					},
				},
			},
		},
	}
	testTree := NewTestTree(t, testTreeNodes)

	ledger := NewTreeLedger(testTree, "test-token")

	testTokenExists, err := ledger.TokenExists()
	require.Nil(t, err)

	assert.True(t, testTokenExists)

	ledger = NewTreeLedger(testTree, "other")

	otherTokenExists, err := ledger.TokenExists()
	require.Nil(t, err)

	assert.False(t, otherTokenExists)
}

func TestTreeLedger_createToken(t *testing.T) {
	testTreeNodes := map[string]interface{}{
		"_tupelo": map[string]interface{}{
			"tokens": map[string]interface{}{
				"test-token": map[string]interface{}{
					TokenMintLabel: []map[string]interface{}{
						{"amount": 1},
						{"amount": 2},
						{"amount": 3},
					},
					TokenSendLabel: []map[string]interface{}{
						{"amount": 4},
						{"amount": 2},
					},
					TokenReceiveLabel: []map[string]interface{}{
						{"amount": 10},
					},
				},
			},
		},
	}
	testTree := NewTestTree(t, testTreeNodes)

	ledger := NewTreeLedger(testTree, "other")

	otherTokenExists, err := ledger.TokenExists()
	require.Nil(t, err)

	assert.False(t, otherTokenExists)

	newTree, err := ledger.createToken()
	require.Nil(t, err)

	ledger = NewTreeLedger(newTree, "other")

	otherTokenExists, err = ledger.TokenExists()
	require.Nil(t, err)

	assert.True(t, otherTokenExists)

	balance, err := ledger.Balance()
	require.Nil(t, err)

	assert.Zero(t, balance)
}

func TestTreeLedger_EstablishToken(t *testing.T) {
	testTreeNodes := map[string]interface{}{
		"_tupelo": map[string]interface{}{
			"tokens": map[string]interface{}{
				"test-token": map[string]interface{}{
					TokenMintLabel: []map[string]interface{}{
						{"amount": 1},
						{"amount": 2},
						{"amount": 3},
					},
					TokenSendLabel: []map[string]interface{}{
						{"amount": 4},
						{"amount": 2},
					},
					TokenReceiveLabel: []map[string]interface{}{
						{"amount": 10},
					},
				},
			},
		},
	}
	testTree := NewTestTree(t, testTreeNodes)

	ledger := NewTreeLedger(testTree, "other")

	otherTokenExists, err := ledger.TokenExists()
	require.Nil(t, err)

	assert.False(t, otherTokenExists)

	newTree, err := ledger.EstablishToken(TokenMonetaryPolicy{Maximum: uint64(42)})
	require.Nil(t, err)

	ledger = NewTreeLedger(newTree, "other")

	otherTokenExists, err = ledger.TokenExists()
	require.Nil(t, err)

	assert.True(t, otherTokenExists)

	// cannot mint more than monetary policy allows
	_, err = ledger.MintToken(uint64(43))
	assert.NotNil(t, err)
}

func TestTreeLedger_MintToken(t *testing.T) {
	testTreeNodes := map[string]interface{}{
		"_tupelo": map[string]interface{}{
			"tokens": map[string]interface{}{
				"test-token": map[string]interface{}{
					MonetaryPolicyLabel: TokenMonetaryPolicy{Maximum: uint64(100)},
					TokenBalanceLabel:   uint64(10),
					TokenMintLabel:      []map[string]interface{}{
						{"amount": 1},
						{"amount": 2},
						{"amount": 3},
					},
					TokenSendLabel:      []map[string]interface{}{
						{"amount": 4},
						{"amount": 2},
					},
					TokenReceiveLabel:   []map[string]interface{}{
						{"amount": 10},
					},
				},
			},
		},
	}
	testTree := NewTestTree(t, testTreeNodes)

	ledger := NewTreeLedger(testTree, "test-token")

	newTree, err := ledger.MintToken(uint64(4))
	require.Nil(t, err)

	ledger = NewTreeLedger(newTree, "test-token")

	balance, err := ledger.Balance()
	require.Nil(t, err)

	assert.Equal(t, uint64(14), balance)
}

func TestTreeLedger_SendToken(t *testing.T) {
	testTreeNodes := map[string]interface{}{
		"_tupelo": map[string]interface{}{
			"tokens": map[string]interface{}{
				"test-token": map[string]interface{}{
					TokenBalanceLabel: uint64(10),
					TokenMintLabel:    []map[string]interface{}{
						{"amount": 1},
						{"amount": 2},
						{"amount": 3},
					},
					TokenSendLabel:    []map[string]interface{}{
						{"amount": 4},
						{"amount": 2},
					},
					TokenReceiveLabel: []map[string]interface{}{
						{"amount": 10},
					},
				},
			},
		},
	}
	testTree := NewTestTree(t, testTreeNodes)

	ledger := NewTreeLedger(testTree, "test-token")

	newTree, err := ledger.SendToken("test-tx-id", "did:tupelo:testchaintree", uint64(5))
	require.Nil(t, err)

	ledger = NewTreeLedger(newTree, "test-token")

	balance, err := ledger.Balance()
	require.Nil(t, err)

	assert.Equal(t, uint64(5), balance)
}

func TestTreeLedger_ReceiveToken(t *testing.T) {
	recipientTreeNodes := map[string]interface{}{
		"_tupelo": map[string]interface{}{
			"tokens": map[string]interface{}{},
		},
	}

	recipientTree := NewTestTree(t, recipientTreeNodes)

	recipientLedger := NewTreeLedger(recipientTree, "test-token")

	recipientTree, err := recipientLedger.ReceiveToken("test-send-token-2", uint64(5))
	require.Nil(t, err)

	// TODO: Finish this
}
