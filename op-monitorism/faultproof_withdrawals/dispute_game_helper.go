package faultproof_withdrawals

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/dispute"
	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/l1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

type DisputeGameHelper struct {
	//objects
	l1Client           *ethclient.Client
	l2Client           *ethclient.Client
	rpc_l2Client       *rpc.Client
	optimismPortal2    *l1.OptimismPortal2
	disputeGameFactory *dispute.DisputeGameFactory
	ctx                context.Context
	gameCache          *lru.Cache
	rootProofCache     *lru.Cache
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

type DisputeGameFactoryCoordinates struct {
	GameType                  uint32
	GameIndex                 uint64
	disputeGameProxyAddress   common.Address
	disputeGameProxyTimestamp uint64
}

type DisputeFactoryGameHelper struct {
	//objects
	l1Client                 *ethclient.Client
	optimismPortal2          *l1.OptimismPortal2
	DisputeGameFactoryCaller dispute.DisputeGameFactoryCaller
}

type DisputeGameFactoryIterator struct {
	DisputeGameFactoryCaller      *dispute.DisputeGameFactoryCaller
	currentIndex                  uint64
	gameCount                     uint64
	init                          bool
	DisputeGameFactoryCoordinates *DisputeGameFactoryCoordinates
}

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

func NewDisputeGameHelper(ctx context.Context, l1Client *ethclient.Client, l2Client *ethclient.Client, optimismPortal2 *l1.OptimismPortal2) (*DisputeGameHelper, error) {

	gameCache, err := lru.New(1000)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}
	rootProofCache, err := lru.New(1000)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}
	rpcClient := l2Client.Client()

	disputeGameFactoryAddress, err := optimismPortal2.DisputeGameFactory(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute game factory address: %w", err)
	}
	disputeGameFactory, err := dispute.NewDisputeGameFactory(disputeGameFactoryAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to dispute game factory: %w", err)
	}

	return &DisputeGameHelper{
		optimismPortal2:    optimismPortal2,
		disputeGameFactory: disputeGameFactory,
		l1Client:           l1Client,
		l2Client:           l2Client,
		rpc_l2Client:       rpcClient,
		ctx:                ctx,
		gameCache:          gameCache,
		rootProofCache:     rootProofCache,
	}, nil
}

func (op *DisputeGameHelper) GetDisputeGameFromAddress(disputeGameProxyAddress common.Address) (*DisputeGame, error) {

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

func (op *DisputeGameHelper) GetRootProofFromTrustedL2Node(l2blockNumber *big.Int) ([32]byte, error) {

	ret, found := op.rootProofCache.Get(l2blockNumber)
	if !found {
		// log.Info("Cache Miss", "l2blockNumber", l2blockNumber)

		var result OutputResponse
		l2blockNumberHex := hexutil.EncodeBig(l2blockNumber)

		err := op.rpc_l2Client.CallContext(op.ctx, &result, "optimism_outputAtBlock", l2blockNumberHex)
		if err != nil {
			return [32]byte{}, fmt.Errorf("failed to get output at block for game block:%v : %w", l2blockNumberHex, err)
		}
		trustedRootProof, err := stringToBytes32(result.OutputRoot)
		if err != nil {
			return [32]byte{}, fmt.Errorf("failed to convert output root to bytes32: %w", err)
		}
		ret = trustedRootProof
	}

	return ret.([32]byte), nil
}

func (op *DisputeGameHelper) IsValidOutputRoot(gameClaim [32]byte, l2blockNumber *big.Int) (bool, error) {
	trustedRootClaim, err := op.GetRootProofFromTrustedL2Node(l2blockNumber)
	if err != nil {
		return false, fmt.Errorf("failed to get root proof from trusted l2 node: %w", err)
	}
	return gameClaim == trustedRootClaim, nil
}

func (op *DisputeGameHelper) IsGameBlacklisted(disputeGame *DisputeGame) (bool, error) {

	isBlacklisted, err := op.optimismPortal2.DisputeGameBlacklist(nil, disputeGame.disputeGameProxyAddress)
	if err != nil {
		return false, fmt.Errorf("failed to get dispute game blacklist status: %w", err)
	}

	return isBlacklisted, err
}

func (op *DisputeGameHelper) IsGameStateINPROGRESS(disputeGame *DisputeGame) (bool, error) {
	gameStatus, err := disputeGame.faultDisputeGame.Status(nil)
	if err != nil {
		return false, fmt.Errorf("failed to get game status: %w", err)
	}
	return GameStatus(gameStatus) == IN_PROGRESS, nil
}

func (op *DisputeGameHelper) IsGameStateCHALLENGER_WINS(disputeGame *DisputeGame) (bool, error) {
	gameStatus, err := disputeGame.faultDisputeGame.Status(nil)
	if err != nil {
		return false, fmt.Errorf("failed to get game status: %w", err)
	}
	return GameStatus(gameStatus) == CHALLENGER_WINS, nil
}

func (op *DisputeGameHelper) IsGameStateDEFENDER_WINS(disputeGame *DisputeGame) (bool, error) {
	gameStatus, err := disputeGame.faultDisputeGame.Status(nil)
	if err != nil {
		return false, fmt.Errorf("failed to get game status: %w", err)
	}
	return GameStatus(gameStatus) == DEFENDER_WINS, nil
}

func NewDisputeGameFactoryHelper(ctx context.Context, l1Client *ethclient.Client, optimismPortal2 *l1.OptimismPortal2) (*DisputeFactoryGameHelper, error) {

	disputeGameFactoryAddress, err := optimismPortal2.DisputeGameFactory(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute game factory address: %w", err)
	}
	disputeGameFactory, err := dispute.NewDisputeGameFactory(disputeGameFactoryAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to dispute game factory: %w", err)
	}
	disputeGameFactoryCaller := disputeGameFactory.DisputeGameFactoryCaller

	return &DisputeFactoryGameHelper{
		optimismPortal2:          optimismPortal2,
		l1Client:                 l1Client,
		DisputeGameFactoryCaller: disputeGameFactoryCaller,
	}, nil
}

func (op *DisputeFactoryGameHelper) GetDisputeGameCoordinatesFromGameIndex(gameIndex uint64) (*DisputeGameFactoryCoordinates, error) {
	gameDetails, err := op.DisputeGameFactoryCaller.GameAtIndex(nil, big.NewInt(int64(gameIndex)))
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute game details: %w", err)
	}

	return &DisputeGameFactoryCoordinates{
		GameType:                  gameDetails.GameType,
		GameIndex:                 gameIndex,
		disputeGameProxyAddress:   gameDetails.Proxy,
		disputeGameProxyTimestamp: gameDetails.Timestamp,
	}, nil
}

func (op *DisputeFactoryGameHelper) GetDisputeGameCount() (uint64, error) {
	gameCountBigInt, err := op.DisputeGameFactoryCaller.GameCount(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get num dispute games: %w", err)
	}
	return gameCountBigInt.Uint64(), nil
}

func (op *DisputeFactoryGameHelper) GetDisputeGameIteratorFromDisputeGameFactory() (*DisputeGameFactoryIterator, error) {

	gameCountBigInt, err := op.DisputeGameFactoryCaller.GameCount(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get num dispute games: %w", err)
	}
	gameCount := gameCountBigInt.Uint64()

	return &DisputeGameFactoryIterator{
		DisputeGameFactoryCaller:      &op.DisputeGameFactoryCaller,
		currentIndex:                  0,
		gameCount:                     gameCount,
		DisputeGameFactoryCoordinates: nil,
	}, nil
}

func (it *DisputeGameFactoryIterator) RefreshElements() error {
	gameCountBigInt, err := it.DisputeGameFactoryCaller.GameCount(nil)
	if err != nil {
		return fmt.Errorf("failed to get num dispute games: %w", err)
	}
	it.gameCount = gameCountBigInt.Uint64()
	return nil
}

func (it *DisputeGameFactoryIterator) Next() bool {
	if it.currentIndex >= it.gameCount-1 {
		return false
	}

	var currentIndex uint64 = 0
	if it.init {
		currentIndex = it.currentIndex + 1
	}

	gameDetails, err := it.DisputeGameFactoryCaller.GameAtIndex(nil, big.NewInt(int64(currentIndex)))
	if err != nil {
		return false
	}

	it.init = true
	it.currentIndex = currentIndex

	it.DisputeGameFactoryCoordinates = &DisputeGameFactoryCoordinates{
		GameType:                  gameDetails.GameType,
		GameIndex:                 currentIndex,
		disputeGameProxyAddress:   gameDetails.Proxy,
		disputeGameProxyTimestamp: gameDetails.Timestamp,
	}

	return true
}
