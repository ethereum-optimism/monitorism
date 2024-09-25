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
	IN_PROGRESS GameStatus = iota
	CHALLENGER_WINS
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
	return fmt.Sprintf("DisputeGame[ disputeGameProxyAddress=%v rootClaim=%s l2blockNumber=%s l2ChainID=%s ]",
		d.ProxyAddress,
		common.BytesToHash(d.RootClaim[:]).Hex(),
		d.L2blockNumber.String(),
		d.L2ChainID.String(),
	)
}

func NewFaultDisputeGameHelper(ctx context.Context, l1Client *ethclient.Client) (*FaultDisputeGameHelper, error) {

	gameCache, err := lru.New(1000)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	return &FaultDisputeGameHelper{
		l1Client:  l1Client,
		ctx:       ctx,
		gameCache: gameCache,
	}, nil
}

func (op *FaultDisputeGameHelper) GetDisputeGameProxyFromAddress(disputeGameProxyAddress common.Address) (*FaultDisputeGameProxy, error) {

	ret, found := op.gameCache.Get(disputeGameProxyAddress)
	if !found {
		// log.Info("Cache Miss", "disputeGameProxyAddress", disputeGameProxyAddress)
		faultDisputeGame, err := dispute.NewFaultDisputeGame(disputeGameProxyAddress, op.l1Client)
		if err != nil {
			return nil, fmt.Errorf("failed to bind to dispute game: %w", err)
		}

		rootClaim, err := faultDisputeGame.RootClaim(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get root claim for game: %w", err)
		}
		l2blockNumber, err := faultDisputeGame.L2BlockNumber(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get l2 block number for game: %w", err)
		}

		l2ChainID, err := faultDisputeGame.L2ChainId(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get l2 chain id for game: %w", err)
		}

		gameStatus, err := faultDisputeGame.Status(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get game status: %w", err)
		}

		ret = &FaultDisputeGameProxy{
			DisputeGameData: &DisputeGameData{
				ProxyAddress:  disputeGameProxyAddress,
				RootClaim:     rootClaim,
				L2blockNumber: l2blockNumber,
				L2ChainID:     l2ChainID,
				Status:        GameStatus(gameStatus),
			},
			FaultDisputeGame: faultDisputeGame,
		}

		op.gameCache.Add(disputeGameProxyAddress, ret)

	}

	return ret.(*FaultDisputeGameProxy), nil
}

func (op *FaultDisputeGameProxy) RefreshState() error {
	gameStatus, err := op.FaultDisputeGame.Status(nil)
	if err != nil {
		return fmt.Errorf("failed to get game status: %w", err)
	}
	op.DisputeGameData.Status = GameStatus(gameStatus)
	return nil
}
