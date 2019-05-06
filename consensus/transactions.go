package consensus

import (
	"fmt"
	"strings"

	cid "github.com/ipfs/go-cid"
	cbornode "github.com/ipfs/go-ipld-cbor"
	"github.com/quorumcontrol/storage"

	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/dag"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/messages/transactions"
)

const (
	TreePathForAuthentications = "_tupelo/authentications"
	TreePathForTokens          = "_tupelo/tokens"
	TreePathForStake           = "_tupelo/stake"
	TreePathForData            = "data"

	TokenMintLabel    = "mints"
	TokenSendLabel    = "sends"
	TokenReceiveLabel = "receives"
)

func init() {
	typecaster.AddType(Token{})
	typecaster.AddType(TokenMint{})
	typecaster.AddType(TokenSend{})
	typecaster.AddType(TokenReceive{})
	typecaster.AddType(StakePayload{})
	cbornode.RegisterCborType(Token{})
	cbornode.RegisterCborType(TokenMint{})
	cbornode.RegisterCborType(TokenSend{})
	cbornode.RegisterCborType(TokenReceive{})
	cbornode.RegisterCborType(StakePayload{})
}

func complexType(obj interface{}) bool {
	switch obj.(type) {
	// These are the built in type of go (excluding map) plus cid.Cid
	// Use SetAsLink if attempting to set map
	case bool, byte, complex64, complex128, error, float32, float64, int, int8, int16, int32, int64, string, uint, uint16, uint32, uint64, uintptr, cid.Cid, *bool, *byte, *complex64, *complex128, *error, *float32, *float64, *int, *int8, *int16, *int32, *int64, *string, *uint, *uint16, *uint32, *uint64, *uintptr, *cid.Cid, []bool, []byte, []complex64, []complex128, []error, []float32, []float64, []int, []int8, []int16, []int32, []int64, []string, []uint, []uint16, []uint32, []uint64, []uintptr, []cid.Cid, []*bool, []*byte, []*complex64, []*complex128, []*error, []*float32, []*float64, []*int, []*int8, []*int16, []*int32, []*int64, []*string, []*uint, []*uint16, []*uint32, []*uint64, []*uintptr, []*cid.Cid:
		return false
	default:
		return true
	}
}

func DecodePath(path string) ([]string, error) {
	trimmed := strings.TrimPrefix(path, "/")

	if trimmed == "" {
		return []string{}, nil
	}

	split := strings.Split(trimmed, "/")
	for _, component := range split {
		if component == "" {
			return nil, fmt.Errorf("malformed path string containing repeated separator: %s", path)
		}
	}

	return split, nil
}

// SetDataTransaction just sets a path in tree/data to arbitrary data.
func SetDataTransaction(_ string, tree *dag.Dag, transaction *transactions.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payloadWrapper := &transactions.Transaction_SetDataPayload{}
	err := typecaster.ToType(transaction.Payload, payloadWrapper)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error casting payload: %v", err)}
	}

	payload := payloadWrapper.SetDataPayload

	path, err := DecodePath(payload.Path)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error decoding path: %v", err)}
	}

	// SET_DATA always sets inside tree/data
	dataPath, _ := DecodePath(TreePathForData)
	path = append(dataPath, path...)

	if complexType(payload.Value) {
		newTree, err = tree.SetAsLink(path, payload.Value)
	} else {
		newTree, err = tree.Set(path, payload.Value)
	}
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error setting: %v", err)}
	}

	return newTree, true, nil
}

// SetOwnershipTransaction changes the ownership of a tree by adding a public key array to /_tupelo/authentications
func SetOwnershipTransaction(_ string, tree *dag.Dag, transaction *transactions.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payloadWrapper := &transactions.Transaction_SetOwnershipPayload{}
	err := typecaster.ToType(transaction.Payload, payloadWrapper)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error casting payload: %v", err)}
	}

	payload := payloadWrapper.SetOwnershipPayload

	path, err := DecodePath(TreePathForAuthentications)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error decoding path: %v", err)}
	}

	newTree, err = tree.SetAsLink(path, payload.Authentication)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error setting: %v", err)}
	}

	return newTree, true, nil
}

type TokenName struct {
	ChainTreeDID string
	LocalName    string
}

func (tn *TokenName) String() string {
	return strings.Join([]string{tn.ChainTreeDID, tn.LocalName}, ":")
}

func tokenNameFromString(tokenName string) TokenName {
	components := strings.Split(tokenName, ":")
	ctDIDComponents := components[:len(components)-1]
	localName := components[len(components)-1]
	ctDID := strings.Join(ctDIDComponents, ":")
	return TokenName{ChainTreeDID: ctDID, LocalName: localName}
}

func CanonicalTokenName(tree *dag.Dag, defaultChainTreeDID, tokenName string, requireDefault bool) (*TokenName, error) {
	if strings.HasPrefix(tokenName, "did:tupelo:") {
		tn := tokenNameFromString(tokenName)
		if requireDefault && tn.ChainTreeDID != defaultChainTreeDID {
			return nil, fmt.Errorf("invalid chaintree DID in token name")
		}
		return &tn, nil
	}

	tokensPath, err := DecodePath(TreePathForTokens)
	if err != nil {
		return nil, fmt.Errorf("error decoding tokens path: %v", err)
	}

	tokensObj, remaining, err := tree.Resolve(tokensPath)
	if err != nil {
		return nil, fmt.Errorf("error resolving tokens path: %v", err)
	}
	if len(remaining) > 0 {
		// probably just haven't established any tokens yet
		return &TokenName{ChainTreeDID: defaultChainTreeDID, LocalName: tokenName}, nil
	}

	tokens, ok := tokensObj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("tokens node was a %T; expected map[string]interface{}", tokensObj)
	}

	var matchedToken *TokenName
	for token := range tokens {
		tn := tokenNameFromString(token)
		if tn.LocalName == tokenName {
			if matchedToken != nil {
				return nil, fmt.Errorf("ambiguous token names found for %s; please provide full name", tokenName)
			}
			matchedToken = &tn
		}
	}

	if matchedToken != nil {
		return matchedToken, nil
	}

	return &TokenName{ChainTreeDID: defaultChainTreeDID, LocalName: tokenName}, nil
}

type TokenMonetaryPolicy struct {
	Maximum uint64
}

type EstablishTokenPayload struct {
	Name           string
	MonetaryPolicy TokenMonetaryPolicy
}

func EstablishTokenTransaction(chainTreeDID string, tree *dag.Dag, transaction *transactions.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payload := &EstablishTokenPayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error typecasting payload: %v", err)}
	}

	tokenName, err := CanonicalTokenName(tree, chainTreeDID, payload.Name, true)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting canonical token name for %s: %v", payload.Name, err)}
	}

	ledger := NewTreeLedger(tree, tokenName.String())

	tokenExists, err := ledger.TokenExists()
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error checking for existence of token \"%s\"", tokenName)}
	}
	if tokenExists {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error, token \"%s\" already exists", tokenName)}
	}

	newTree, err = ledger.EstablishToken(payload.MonetaryPolicy)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: err.Error()}
	}

	return newTree, true, nil
}

func MintTokenTransaction(chainTreeDID string, tree *dag.Dag, transaction *transactions.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payload := &transactions.MintTokenPayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error typecasting payload: %v", err)}
	}

	tokenName, err := CanonicalTokenName(tree, chainTreeDID, payload.Name, true)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting canonical token name for %s: %v", payload.Name, err)}
	}

	ledger := NewTreeLedger(tree, tokenName.String())

	newTree, err = ledger.MintToken(payload.Amount)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error minting token: %v", err)}
	}

	return newTree, true, nil
}

func SendTokenTransaction(chainTreeDID string, tree *dag.Dag, transaction *transactions.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payloadWrapper := &transactions.Transaction_SendTokenPayload{}
	err := typecaster.ToType(transaction.Payload, payloadWrapper)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error casting payload: %v", err)}
	}

	payload := payloadWrapper.SendTokenPayload
	err = typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error typecasting payload: %v", err)}
	}

	tokenName, err := CanonicalTokenName(tree, chainTreeDID, payload.Name, false)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error getting canonical token name for %s: %v", payload.Name, err)}
	}

	ledger := NewTreeLedger(tree, tokenName.String())

	newTree, err = ledger.SendToken(payload.Id, payload.Destination, payload.Amount)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error sending token: %v", err)}
	}

	return newTree, true, nil
}

// Returns the first node in tree linked to by a value of parentNode
// (i.e. a CID value) and the parentNode key it was found under.
// Useful for finding token & send nodes in ReceiveToken leaves.
func findFirstLinkedNode(tree *dag.Dag, parentNode map[string]interface{}) (key string, node *cbornode.Node, err error) {
	for k, v := range parentNode {
		nodeCid, ok := v.(cid.Cid)
		if !ok {
			continue
		}

		node, err := tree.Get(nodeCid)
		if err == nil && node != nil {
			return k, node, nil
		}
	}
	return "", nil, fmt.Errorf("no linked nodes were found in the DAG")
}

func getSenderDagFromReceive(payload *transactions.ReceiveTokenPayload) (*dag.Dag, chaintree.CodedError) {
	tipCid, err := cid.Cast(payload.Tip)
	if err != nil {
		return nil, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error casting tip to CID: %v", err)}
	}

	leaves := payload.Leaves
	nodes := make([]*cbornode.Node, 0)
	sw := safewrap.SafeWrap{}
	for _, l := range leaves {
		cborNode := sw.Decode(l)
		if sw.Err != nil {
			return nil, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error decoding CBOR node: %v", sw.Err)}
		}

		// make sure tip is first
		if cborNode.Cid() == tipCid {
			nodes = append([]*cbornode.Node{cborNode}, nodes...)
		} else {
			nodes = append(nodes, cborNode)
		}
	}

	nodeStore := nodestore.NewStorageBasedStore(storage.NewMemStorage())
	senderDag, err := dag.NewDagWithNodes(nodeStore, nodes...)
	if err != nil {
		return nil, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error recreating sender leaves DAG: %v", err)}
	}

	return senderDag, nil
}

func getTokenNameFromReceive(senderDag *dag.Dag) (string, chaintree.CodedError) {
	treePath, err := DecodePath(TreePathForTokens)
	if err != nil {
		return "", &ErrorCode{Code: 999, Memo: fmt.Sprintf("error decoding tree path for tokens: %v", err)}
	}
	tokensPath := append([]string{"tree"}, treePath...)

	uncastTokens, remaining, err := senderDag.Resolve(tokensPath)
	if err != nil {
		return "", &ErrorCode{Code: 999, Memo: fmt.Sprintf("error resolving tokens: %v", err)}
	}
	if len(remaining) > 0 {
		return "", &ErrorCode{Code: 999, Memo: fmt.Sprintf("error resolving tokens: remaining path elements: %v", remaining)}
	}

	tokens, ok := uncastTokens.(map[string]interface{})
	if !ok {
		return "", &ErrorCode{Code: 999, Memo: "error casting tokens map"}
	}

	tokenName, _, err := findFirstLinkedNode(senderDag, tokens)
	if err != nil {
		return "", &ErrorCode{Code: 999, Memo: fmt.Sprintf("error finding token node: %v", err)}
	}

	return tokenName, nil
}

func getSendTokenFromReceive(senderDag *dag.Dag, tokenName string) (*TokenSend, chaintree.CodedError) {
	treePath, err := DecodePath(TreePathForTokens)
	if err != nil {
		return nil, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error decoding tree path for tokens: %v", err)}
	}
	tokensPath := append([]string{"tree"}, treePath...)

	tokenSendsPath := append(tokensPath, tokenName, TokenSendLabel)
	uncastTokenSends, remaining, err := senderDag.Resolve(tokenSendsPath)
	if err != nil {
		return nil, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error resolving token sends: %v", err)}
	}
	if len(remaining) > 0 {
		return nil, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error resolving token sends: remaining path elements: %v", remaining)}
	}

	tokenSends, ok := uncastTokenSends.([]interface{})
	if !ok {
		return nil, &ErrorCode{Code: 999, Memo: "error casting token sends"}
	}

	tokenSendsMap := make(map[string]interface{})
	for i, ts := range tokenSends {
		tokenSendsMap[string(i)] = ts
	}

	_, tokenSendNode, err := findFirstLinkedNode(senderDag, tokenSendsMap)
	if err != nil {
		return nil, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error find token send node: %v", err)}
	}

	tokenSend := TokenSend{}
	err = cbornode.DecodeInto(tokenSendNode.RawData(), &tokenSend)
	if err != nil {
		return nil, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error decoding token send node: %v", err)}
	}

	return &tokenSend, nil
}

func ReceiveTokenTransaction(_ string, tree *dag.Dag, transaction *transactions.Transaction) (newTree *dag.Dag, valid bool, codedError chaintree.CodedError) {
	payload := &transactions.ReceiveTokenPayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error typecasting payload: %v", err)}
	}

	tipCid, err := cid.Cast(payload.Tip)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error casting tip to CID: %v", err)}
	}

	senderDag, codedErr := getSenderDagFromReceive(payload)
	if codedErr != nil {
		return nil, false, codedErr
	}

	// verify tip matches root from leaves
	if tipCid != senderDag.Tip {
		return nil, false, &ErrorCode{Code: 999, Memo: "invalid tip and/or leaves"}
	}

	tokenName, codedErr := getTokenNameFromReceive(senderDag)
	if codedErr != nil {
		return nil, false, codedErr
	}

	tokenSend, codedErr := getSendTokenFromReceive(senderDag, tokenName)
	if codedErr != nil {
		return nil, false, codedErr
	}

	tokenAmount := tokenSend.Amount

	// update token ledger
	ledger := NewTreeLedger(tree, tokenName)

	newTree, err = ledger.ReceiveToken(payload.SendTokenTransactionId, tokenAmount)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: err.Error()}
	}

	return newTree, true, nil
}

type StakePayload struct {
	GroupId string
	Amount  uint64
	DstKey  PublicKey
	VerKey  PublicKey
}

// THIS IS A pre-ALPHA TRANSACTION AND NO RULES ARE ENFORCED! Anyone can stake and join a group with no consequences.
// additionally, it only allows staking a single group at the moment
func StakeTransaction(_ string, tree *dag.Dag, transaction *transactions.Transaction) (newTree *dag.Dag, valid bool, codedErr chaintree.CodedError) {
	payload := &StakePayload{}
	err := typecaster.ToType(transaction.Payload, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error casting payload: %v", err)}
	}

	path, err := DecodePath(TreePathForStake)
	if err != nil {
		return nil, false, &ErrorCode{Code: ErrUnknown, Memo: fmt.Sprintf("error decoding path: %v", err)}
	}

	newTree, err = tree.SetAsLink(path, payload)
	if err != nil {
		return nil, false, &ErrorCode{Code: 999, Memo: fmt.Sprintf("error setting: %v", err)}
	}

	return newTree, true, nil
}
