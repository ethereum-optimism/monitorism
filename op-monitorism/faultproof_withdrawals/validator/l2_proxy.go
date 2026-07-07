package validator

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

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
