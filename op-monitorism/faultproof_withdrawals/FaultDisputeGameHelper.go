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

type DisputeGame struct {
	disputeGameProxyAddress common.Address
	//game object
	faultDisputeGame *dispute.FaultDisputeGame

	// game data
	rootClaim     [32]byte
	l2blockNumber *big.Int
	l2ChainID     *big.Int
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

// Define the GameStatus type
type GameCorrectness uint8

// Define constants for the GameStatus using iota
const (
	UNKNOWN GameCorrectness = iota
	CORRECT
	INCORRECT
)

// Implement the Stringer interface for pretty printing
func (gs GameCorrectness) String() string {
	switch gs {
	case UNKNOWN:
		return "UNKNOWN"
	case CORRECT:
		return "CORRECT"
	case INCORRECT:
		return "INCORRECT"
	default:
		return "UNKNOWN"
	}
}

func (d DisputeGame) String() string {
	return fmt.Sprintf("DisputeGame[ disputeGameProxyAddress=%v rootClaim=%s l2blockNumber=%s l2ChainID=%s ]",
		d.disputeGameProxyAddress,
		common.BytesToHash(d.rootClaim[:]).Hex(),
		d.l2blockNumber.String(),
		d.l2ChainID.String(),
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

func (op *FaultDisputeGameHelper) GetDisputeGameFromAddress(disputeGameProxyAddress common.Address) (*DisputeGame, error) {

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

		ret = &DisputeGame{
			disputeGameProxyAddress: disputeGameProxyAddress,
			rootClaim:               rootClaim,
			l2blockNumber:           l2blockNumber,
			l2ChainID:               l2ChainID,
			faultDisputeGame:        faultDisputeGame,
		}

		op.gameCache.Add(disputeGameProxyAddress, ret)

	}

	return ret.(*DisputeGame), nil
}

func (op *FaultDisputeGameHelper) IsGameStateINPROGRESS(disputeGame *DisputeGame) (bool, error) {
	gameStatus, err := disputeGame.faultDisputeGame.Status(nil)
	if err != nil {
		return false, fmt.Errorf("failed to get game status: %w", err)
	}
	return GameStatus(gameStatus) == IN_PROGRESS, nil
}

func (op *FaultDisputeGameHelper) IsGameStateCHALLENGER_WINS(disputeGame *DisputeGame) (bool, error) {
	gameStatus, err := disputeGame.faultDisputeGame.Status(nil)
	if err != nil {
		return false, fmt.Errorf("failed to get game status: %w", err)
	}
	return GameStatus(gameStatus) == CHALLENGER_WINS, nil
}

func (op *FaultDisputeGameHelper) IsGameStateDEFENDER_WINS(disputeGame *DisputeGame) (bool, error) {
	gameStatus, err := disputeGame.faultDisputeGame.Status(nil)
	if err != nil {
		return false, fmt.Errorf("failed to get game status: %w", err)
	}
	return GameStatus(gameStatus) == DEFENDER_WINS, nil
}
