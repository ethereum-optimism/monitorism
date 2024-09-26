package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/dispute"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	lru "github.com/hashicorp/golang-lru"
)

type FaultDisputeGameProxy struct {
	FaultDisputeGame *dispute.FaultDisputeGame
	DisputeGameData  *DisputeGameData
}

type DisputeGameData struct {
	ProxyAddress common.Address
	// game data
	RootClaim     [32]byte
	L2blockNumber *big.Int
	L2ChainID     *big.Int
	Status        GameStatus
	CreatedAt     uint64
	ResolvedAt    uint64
}

type FaultDisputeGameHelper struct {
	//objects
	l1Client  *ethclient.Client
	ctx       context.Context
	gameCache *lru.Cache
}

type OutputResponse struct {
	Version    string `json:"version"`
	OutputRoot string `json:"outputRoot"`
}

// Define the GameStatus type
type GameStatus uint8

// Define constants for the GameStatus using iota
const (
	// The game is currently in progress, and has not been resolved.
	IN_PROGRESS GameStatus = iota

	// The game has concluded, and the `rootClaim` was challenged successfully.
	CHALLENGER_WINS

	// The game has concluded, and the `rootClaim` could not be contested.
	DEFENDER_WINS
)

// Implement the Stringer interface for pretty printing
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

func (d DisputeGameData) String() string {
	return fmt.Sprintf("DisputeGame[ disputeGameProxyAddress=%v rootClaim=%s l2blockNumber=%s l2ChainID=%s status=%v createdAt=%v  resolvedAt=%v ]",
		d.ProxyAddress,
		common.BytesToHash(d.RootClaim[:]).Hex(),
		d.L2blockNumber.String(),
		d.L2ChainID.String(),
		d.Status,
		Timestamp(d.CreatedAt),
		Timestamp(d.ResolvedAt),
	)
}

func (p *FaultDisputeGameProxy) String() string {
	return fmt.Sprintf("FaultDisputeGameProxy[ DisputeGameData=%v ]", p.DisputeGameData)
}

const gameCacheSize = 1000

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

func (op *FaultDisputeGameHelper) GetDisputeGameProxyFromAddress(disputeGameProxyAddress common.Address) (FaultDisputeGameProxy, error) {

	ret, found := op.gameCache.Get(disputeGameProxyAddress)
	if !found {
		faultDisputeGame, err := dispute.NewFaultDisputeGame(disputeGameProxyAddress, op.l1Client)
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

		op.gameCache.Add(disputeGameProxyAddress, ret)

	}

	// return ret.(*FaultDisputeGameProxy), nil
	return *(ret.(*FaultDisputeGameProxy)), nil

}

func (op *FaultDisputeGameProxy) RefreshState() error {

	gameStatus, err := op.FaultDisputeGame.Status(nil)

	if err != nil {
		return fmt.Errorf("failed to get game status: %w", err)
	}

	op.DisputeGameData.Status = GameStatus(gameStatus)

	resolvedAt, err := op.FaultDisputeGame.ResolvedAt(nil)
	if err != nil {
		return fmt.Errorf("failed to get game resolved at: %w", err)
	}
	op.DisputeGameData.ResolvedAt = resolvedAt
	return nil
}
