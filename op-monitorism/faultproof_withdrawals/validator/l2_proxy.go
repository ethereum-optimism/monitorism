package validator

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// sentMessagesSelector is the 4-byte selector of L2ToL1MessagePasser.sentMessages(bytes32).
var sentMessagesSelector = crypto.Keccak256([]byte("sentMessages(bytes32)"))[:4]

type L2Proxy struct {
	l2GethClient          *ethclient.Client
	chainID               *big.Int
	ctx                   context.Context
	l2OpGethBackupClients map[string]*ethclient.Client
	ConnectionError       map[string]uint64
	Connections           map[string]uint64
}

func NewL2Proxy(ctx context.Context, l2GethClientURL string, l2GethBackupClientsURLs map[string]string) (*L2Proxy, error) {
	l2GethClient, err := ethclient.Dial(l2GethClientURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l2: %w", err)
	}

	chainID, err := l2GethClient.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get l2 chain id: %w", err)
	}

	// if backup urls are provided, create a backup client for each
	var l2OpGethBackupClients map[string]*ethclient.Client
	var badClients map[string]string
	if len(l2GethBackupClientsURLs) > 0 {
		l2OpGethBackupClients, badClients, err = GethBackupClientsDictionary(ctx, l2GethBackupClientsURLs, chainID)
		if err != nil {
			return nil, fmt.Errorf("failed to create backup clients: %w. Bad clients: %v", err, badClients)
		}
	}

	return &L2Proxy{l2GethClient: l2GethClient, chainID: chainID, ctx: ctx, l2OpGethBackupClients: l2OpGethBackupClients, ConnectionError: make(map[string]uint64), Connections: make(map[string]uint64)}, nil
}

// get latest known L2 block number
func (l2Proxy *L2Proxy) BlockNumber() (uint64, error) {
	blockNumber, err := l2Proxy.l2GethClient.BlockNumber(l2Proxy.ctx)
	l2Proxy.Connections["default"]++
	if err != nil {
		l2Proxy.ConnectionError["default"]++
		return 0, fmt.Errorf("failed to get block number: %w", err)
	}
	return blockNumber, nil
}

// VerifyRootClaimFromHeader recomputes the L2 output root purely from the block
// header and compares it against the dispute game's root claim.
//
// Post-Isthmus, the header's WithdrawalsRoot IS the L2ToL1MessagePasser storage
// root, so the output root can be reconstructed without any historical trie state
// (no eth_getProof). Validation therefore works for arbitrarily old games and can
// never stall on pruned state.
//
// Pre-Isthmus blocks carry the empty-trie root in WithdrawalsRoot rather than the
// message-passer storage root, so they cannot be verified this way. Those are
// reported (preIsthmus=true) so the monitor can surface them for security triage
// rather than validate them here.
//
// Returns:
//   - trusted: true if the recomputed output root matches the root claim
//   - preIsthmus: true if the block predates Isthmus and cannot be header-verified
//   - error: any error encountered fetching the block
func (l2Proxy *L2Proxy) VerifyRootClaimFromHeader(blockNumber *big.Int, rootClaim [32]byte) (trusted bool, preIsthmus bool, err error) {
	block, err := l2Proxy.l2GethClient.BlockByNumber(l2Proxy.ctx, blockNumber)
	l2Proxy.Connections["default"]++
	if err != nil {
		l2Proxy.ConnectionError["default"]++
		return false, false, fmt.Errorf("failed to get block for game blockInt:%v error:%w", blockNumber, err)
	}

	header := block.Header()
	// Note: the Go field is WithdrawalsHash; its JSON/RLP tag is "withdrawalsRoot".
	// Pre-Isthmus the header does not commit to the message-passer storage root: it is
	// nil (pre-Canyon) or types.EmptyWithdrawalsHash (Canyon..Isthmus — op-geth's
	// EmptyWithdrawalsHash is the OP pre-Isthmus withdrawals-root sentinel).
	//
	// NOTE: this is a heuristic. The robust, chain-agnostic check is the rollup
	// config's isthmus_time compared against header.Time; the sentinel could in
	// principle collide with a genuinely-empty post-Isthmus message-passer storage
	// root on a brand-new low-activity chain. Revisit before relying on other chains.
	if header.WithdrawalsHash == nil || *header.WithdrawalsHash == types.EmptyWithdrawalsHash {
		return false, true, nil
	}

	outputRoot := eth.OutputRoot(&eth.OutputV0{
		StateRoot:                [32]byte(header.Root),
		MessagePasserStorageRoot: [32]byte(*header.WithdrawalsHash),
		BlockHash:                header.Hash(),
	})
	return bytes.Equal(outputRoot[:], rootClaim[:]), false, nil
}

// IsWithdrawalPresentAtHead reports whether the withdrawal hash is recorded in the
// L2ToL1MessagePasser's sentMessages mapping at the current L2 head.
//
// L2ToL1MessagePasser.sentMessages is append-only (only ever set to true, never
// cleared), so a withdrawal present at its dispute game's (possibly old, possibly
// pruned) L2 block is still present at head. Querying head therefore yields the same
// presence answer as the historical block WITHOUT any archive-state dependency —
// head state is never pruned, so this can never stall. A fabricated withdrawal that
// was never initiated on L2 is absent at head and gets flagged.
//
// This is an independent re-check of what the OptimismPortal enforces on-chain
// (defense-in-depth against a Portal inclusion-proof bug). It reads the trusted
// primary L2 node's public getter via eth_call, so no merkle proof (eth_getProof)
// is needed.
func (l2Proxy *L2Proxy) IsWithdrawalPresentAtHead(withdrawalHash [32]byte) (bool, error) {
	data := append(append([]byte{}, sentMessagesSelector...), withdrawalHash[:]...)
	to := predeploys.L2ToL1MessagePasserAddr

	l2Proxy.Connections["default"]++
	res, err := l2Proxy.l2GethClient.CallContract(l2Proxy.ctx, ethereum.CallMsg{To: &to, Data: data}, nil)
	if err != nil {
		l2Proxy.ConnectionError["default"]++
		return false, fmt.Errorf("failed to query sentMessages at head: %w", err)
	}

	// ABI bool: a 32-byte word, non-zero => true.
	for _, b := range res {
		if b != 0 {
			return true, nil
		}
	}
	return false, nil
}

func (l2Proxy *L2Proxy) GetTotalConnections() uint64 {
	totalConnections := uint64(0)
	for _, connection := range l2Proxy.Connections {
		totalConnections += connection
	}
	return totalConnections
}

func (l2Proxy *L2Proxy) GetTotalConnectionErrors() uint64 {
	totalConnectionErrors := uint64(0)
	for _, connectionError := range l2Proxy.ConnectionError {
		totalConnectionErrors += connectionError
	}
	return totalConnectionErrors
}
