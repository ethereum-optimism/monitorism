package validator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/dispute"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	lru "github.com/hashicorp/golang-lru"
)

// FaultDisputeGameProxy represents a proxy for the fault dispute game.
type FaultDisputeGameProxy struct {
	FaultDisputeGame *dispute.FaultDisputeGame // The underlying fault dispute game.
	DisputeGameData  *DisputeGameData          // Data related to the dispute game.
}

// DisputeGameData holds the details of a dispute game.
type DisputeGameData struct {
	ProxyAddress                      common.Address // The address of the dispute game proxy.
	RootClaim                         [32]byte       // The root claim associated with the dispute game.
	L2blockNumber                     *big.Int       // The L2 block number related to the game.
	L2ChainID                         *big.Int       // The L2 chain ID associated with the game.
	Status                            GameStatus     // The current status of the game.
	CreatedAt                         uint64         // Timestamp when the game was created.
	ResolvedAt                        uint64         // Timestamp when the game was resolved.
	IsL2BlockNumberKnownToTrustedNode bool           // Whether the L2 block number is known to a trusted node.
	IsGameBlackListed                 bool           // Whether the game is blacklisted
	TrustedNodeRootClaim              [32]byte       // The root claim known to the trusted node.
}

// FaultDisputeGameHelper assists in interacting with fault dispute games.
type FaultDisputeGameHelper struct {
	// objects
	l1Client  *ethclient.Client // The L1 Ethereum client.
	ctx       context.Context   // Context for managing cancellation and timeouts.
	gameCache *lru.Cache        // Cache for storing game proxies.
}

// OutputResponse represents the response structure for output-related data.
type OutputResponse struct {
	Version    string `json:"version"`    // The version of the output.
	OutputRoot string `json:"outputRoot"` // The output root associated with the response.
}

// GameStatus represents the status of a dispute game.
type GameStatus uint8

// Define constants for the GameStatus using iota.
const (
	IN_PROGRESS     GameStatus = iota // The game is currently in progress and has not been resolved.
	CHALLENGER_WINS                   // The game has concluded, and the root claim was challenged successfully.
	DEFENDER_WINS                     // The game has concluded, and the root claim could not be contested.
)

// String implements the Stringer interface for pretty printing the GameStatus.
func (gs GameStatus) String() string {
	switch gs {
	case IN_PROGRESS:
		return "IN_PROGRESS"
	case CHALLENGER_WINS:
		return "CHALLENGER_WINS"
	case DEFENDER_WINS:
		return "DEFENDER_WINS"
	default:
		return "UNKNOWN"
	}
}

// String provides a string representation of DisputeGameData.
func (d DisputeGameData) String() string {
	return fmt.Sprintf("DisputeGame[ disputeGameProxyAddress: %v rootClaim: %s trustedNodeRootClaim: %s l2blockNumber: %s l2ChainID: %s status: %v createdAt: %v resolvedAt: %v isL2BlockNumberKnownToTrustedNode: %v isGameBlackListed: %v ]",
		d.ProxyAddress,
		common.BytesToHash(d.RootClaim[:]),
		common.BytesToHash(d.TrustedNodeRootClaim[:]),
		d.L2blockNumber.String(),
		d.L2ChainID.String(),
		d.Status,
		Timestamp(d.CreatedAt),
		Timestamp(d.ResolvedAt),
		d.IsL2BlockNumberKnownToTrustedNode,
		d.IsGameBlackListed,
	)
}

// String provides a string representation of the FaultDisputeGameProxy.
func (p *FaultDisputeGameProxy) String() string {
	return fmt.Sprintf("FaultDisputeGameProxy[ DisputeGameData=%v ]", p.DisputeGameData)
}

const gameCacheSize = 1000

// NewFaultDisputeGameHelper initializes a new FaultDisputeGameHelper.
// It creates a cache for storing game proxies and returns the helper instance.
func NewFaultDisputeGameHelper(ctx context.Context, l1Client *ethclient.Client) (*FaultDisputeGameHelper, error) {
	gameCache, err := lru.New(gameCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return &FaultDisputeGameHelper{
		l1Client:  l1Client,
		ctx:       ctx,
		gameCache: gameCache,
	}, nil
}

// GetDisputeGameProxyFromAddress retrieves the FaultDisputeGameProxy from the specified address.
// It fetches the game details and caches the result for future use.
func (fd *FaultDisputeGameHelper) GetDisputeGameProxyFromAddress(disputeGameProxyAddress common.Address) (FaultDisputeGameProxy, error) {
	ret, found := fd.gameCache.Get(disputeGameProxyAddress)
	if !found {
		faultDisputeGame, err := dispute.NewFaultDisputeGame(disputeGameProxyAddress, fd.l1Client)
		if err != nil {
			return FaultDisputeGameProxy{}, fmt.Errorf("failed to bind to dispute game: %w", err)
		}

		rootClaim, err := faultDisputeGame.RootClaim(nil)
		if err != nil {
			return FaultDisputeGameProxy{}, fmt.Errorf("failed to get root claim for game: %w", err)
		}
		l2blockNumber, err := faultDisputeGame.L2BlockNumber(nil)
		if err != nil {
			return FaultDisputeGameProxy{}, fmt.Errorf("failed to get l2 block number for game: %w", err)
		}

		l2ChainID, err := faultDisputeGame.L2ChainId(nil)
		if err != nil {
			return FaultDisputeGameProxy{}, fmt.Errorf("failed to get l2 chain id for game: %w", err)
		}

		gameStatus, err := faultDisputeGame.Status(nil)
		if err != nil {
			return FaultDisputeGameProxy{}, fmt.Errorf("failed to get game status: %w", err)
		}

		createdAt, err := faultDisputeGame.CreatedAt(nil)
		if err != nil {
			return FaultDisputeGameProxy{}, fmt.Errorf("failed to get game created at: %w", err)
		}

		resolvedAt, err := faultDisputeGame.ResolvedAt(nil)
		if err != nil {
			return FaultDisputeGameProxy{}, fmt.Errorf("failed to get game resolved at: %w", err)
		}

		ret = &FaultDisputeGameProxy{
			DisputeGameData: &DisputeGameData{
				ProxyAddress:  disputeGameProxyAddress,
				RootClaim:     rootClaim,
				L2blockNumber: l2blockNumber,
				L2ChainID:     l2ChainID,
				Status:        GameStatus(gameStatus),
				CreatedAt:     createdAt,
				ResolvedAt:    resolvedAt,
			},
			FaultDisputeGame: faultDisputeGame,
		}

		fd.gameCache.Add(disputeGameProxyAddress, ret)
	}

	return *(ret.(*FaultDisputeGameProxy)), nil
}

// RefreshState updates the state of the FaultDisputeGameProxy.
// It retrieves the current status and resolved timestamp of the game.
func (fd *FaultDisputeGameProxy) RefreshState() error {
	if fd.FaultDisputeGame == nil {
		return fmt.Errorf("dispute game is nil")
	}

	if fd.DisputeGameData.Status != IN_PROGRESS {
		return nil
	}

	gameStatus, err := fd.FaultDisputeGame.Status(nil)
	if err != nil {
		return fmt.Errorf("failed to get game status: %w", err)
	}

	fd.DisputeGameData.Status = GameStatus(gameStatus)

	resolvedAt, err := fd.FaultDisputeGame.ResolvedAt(nil)
	if err != nil {
		return fmt.Errorf("failed to get game resolved at: %w", err)
	}
	fd.DisputeGameData.ResolvedAt = resolvedAt
	return nil
}
