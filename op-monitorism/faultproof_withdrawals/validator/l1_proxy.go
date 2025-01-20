package validator

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/dispute"
	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/l1"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// WithdrawalProvenExtensionEvent represents an event for a proven withdrawal.
type WithdrawalProvenExtensionEvent struct {
	WithdrawalHash [32]byte       // Hash of the withdrawal.
	ProofSubmitter common.Address // Address of the proof submitter.
	Raw            Raw            // Raw event data.
}

func (wpee *WithdrawalProvenExtensionEvent) String() string {
	return fmt.Sprintf("WithdrawalProvenExtensionEvent{ WithdrawalHash: 0x%x, ProofSubmitter: 0x%x, Raw: %v}", wpee.WithdrawalHash, wpee.ProofSubmitter, wpee.Raw)
}

type DisputeGameEvent struct {
	EventRef    *WithdrawalProvenExtensionEvent
	DisputeGame *DisputeGame
}

func (dge *DisputeGameEvent) String() string {
	return fmt.Sprintf("DisputeGameEvents{EventRef: %v, DisputeGame: %v}", dge.EventRef, dge.DisputeGame)
}

// DisputeGameRef holds data about a submitted proof.
type DisputeGameRef struct {
	disputeGameProxyAddress   common.Address // Address of the dispute game proxy.
	disputeGameProxyTimestamp uint64         // Timestamp of the dispute game proxy.
}

func (dgr *DisputeGameRef) String() string {
	return fmt.Sprintf("DisputeGameRef{disputeGameProxyAddress: 0x%x, disputeGameProxyTimestamp: %d}", dgr.disputeGameProxyAddress, dgr.disputeGameProxyTimestamp)
}

type DisputeGameClaimData struct {
	RootClaim     [32]byte // The root claim associated with the dispute game.
	L2blockNumber *big.Int // The L2 block number related to the game.
	L2ChainID     *big.Int // The L2 chain ID associated with the game.
}

func (dgcd *DisputeGameClaimData) String() string {
	return fmt.Sprintf("DisputeGameClaimData{RootClaim: 0x%x, L2blockNumber: %d, L2ChainID: %d}", dgcd.RootClaim, dgcd.L2blockNumber, dgcd.L2ChainID)
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

type DisputeGame struct {
	DisputeGameRef       *DisputeGameRef
	DisputeGameClaimData *DisputeGameClaimData
	CreatedAt            uint64 // Timestamp when the game was created.
	ResolvedAt           uint64 // Timestamp when the game was resolved.
	GameStatus           GameStatus
	IsGameBlacklisted    bool
}

func (dg *DisputeGame) String() string {
	return fmt.Sprintf("DisputeGame{DisputeGameRef: %v, DisputeGameClaimData: %v, CreatedAt: %d, ResolvedAt: %d, GameStatus: %v, IsGameBlacklisted: %v}", dg.DisputeGameRef, dg.DisputeGameClaimData, dg.CreatedAt, dg.ResolvedAt, dg.GameStatus, dg.IsGameBlacklisted)
}

type L1Proxy struct {
	ctx             *context.Context
	l1GethClient    *ethclient.Client
	optimismPortal2 *l1.OptimismPortal2
	ConnectionState *L1ConnectionState
}

type L1ConnectionState struct {
	ProxyConnection       uint64
	ProxyConnectionFailed uint64
}

func NewL1Proxy(ctx *context.Context, l1GethURL string, optimismPortalAddress common.Address) (*L1Proxy, error) {
	connectionState := &L1ConnectionState{
		ProxyConnection:       0,
		ProxyConnectionFailed: 0,
	}

	l1Client, err := ethclient.Dial(l1GethURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1: %w", err)
	}

	connectionState.ProxyConnection++
	optimismPortal2, err := l1.NewOptimismPortal2(optimismPortalAddress, l1Client)
	if err != nil {
		connectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to create new OptimismPortal2 instance error:%w", err)
	}

	return &L1Proxy{
		ctx:             ctx,
		l1GethClient:    l1Client,
		optimismPortal2: optimismPortal2,
		ConnectionState: connectionState,
	}, nil
}

func (l1Proxy *L1Proxy) GetDisputeGamesEvents(start uint64, end uint64) ([]DisputeGameEvent, error) {
	provenWithdrawalsExtension1Events, err := l1Proxy.getProvenWithdrawalsExtension1Events(start, &end)
	if err != nil {
		return nil, fmt.Errorf("failed to get proven withdrawals extension1 events error:%w", err)
	}

	disputeGamesEvents := make([]DisputeGameEvent, 0)
	for _, event := range provenWithdrawalsExtension1Events {
		disputeGameRef, err := l1Proxy.getSubmittedProofsDataFromWithdrawalProvenExtensionEvent(&event)
		if err != nil {
			return nil, fmt.Errorf("failed to get submitted proofs data from withdrawal proven extension1 event error:%w", err)
		}

		disputeGame, err := l1Proxy.getDisputeGameProxyFromAddress(disputeGameRef)
		if err != nil {
			return nil, fmt.Errorf("failed to get dispute game proxy from address error:%w", err)
		}

		disputeGamesEvents = append(disputeGamesEvents, DisputeGameEvent{
			EventRef:    &event,
			DisputeGame: disputeGame,
		})
	}

	return disputeGamesEvents, nil
}

// GetProvenWithdrawalsExtension1Events retrieves proven withdrawal extension 1 events within the specified block range.
// It returns a slice of WithdrawalProvenExtensionEvent along with any error encountered.
func (l1Proxy *L1Proxy) getProvenWithdrawalsExtension1Events(start uint64, end *uint64) ([]WithdrawalProvenExtensionEvent, error) {

	l1Proxy.ConnectionState.ProxyConnection++
	filterOpts := &bind.FilterOpts{Context: *l1Proxy.ctx, Start: start, End: end}
	iterator, err := l1Proxy.optimismPortal2.FilterWithdrawalProvenExtension1(filterOpts, nil, nil)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get proven withdrawals extension1 iterator error:%w", err)
	}

	events := make([]WithdrawalProvenExtensionEvent, 0)
	for iterator.Next() {
		event := iterator.Event
		events = append(events, WithdrawalProvenExtensionEvent{
			WithdrawalHash: event.WithdrawalHash,
			ProofSubmitter: event.ProofSubmitter,
			Raw: Raw{
				BlockNumber: event.Raw.BlockNumber,
				TxHash:      event.Raw.TxHash,
			},
		})
	}

	return events, nil
}

func (l1Proxy *L1Proxy) getSubmittedProofsDataFromWithdrawalProvenExtensionEvent(event *WithdrawalProvenExtensionEvent) (*DisputeGameRef, error) {

	l1Proxy.ConnectionState.ProxyConnection++
	opts := &bind.CallOpts{
		Context: *l1Proxy.ctx,
	}

	gameProxyStruct, err := l1Proxy.optimismPortal2.ProvenWithdrawals(opts, event.WithdrawalHash, event.ProofSubmitter)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get proven withdrawal for withdrawal hash:%x proof submitter:%x error:%w", event.WithdrawalHash, event.ProofSubmitter, err)
	}

	return &DisputeGameRef{
		disputeGameProxyAddress:   gameProxyStruct.DisputeGameProxy,
		disputeGameProxyTimestamp: gameProxyStruct.Timestamp,
	}, nil
}

// GetDisputeGameProxyFromAddress retrieves the FaultDisputeGameProxy from the specified address.
// It fetches the game details and caches the result for future use.
func (l1Proxy *L1Proxy) getDisputeGameProxyFromAddress(disputeGameRef *DisputeGameRef) (*DisputeGame, error) {

	disputeGameProxyAddress := disputeGameRef.disputeGameProxyAddress

	l1Proxy.ConnectionState.ProxyConnection++
	faultDisputeGame, err := dispute.NewFaultDisputeGame(disputeGameProxyAddress, l1Proxy.l1GethClient)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to bind to dispute game: %w", err)
	}

	l1Proxy.ConnectionState.ProxyConnection++
	rootClaim, err := faultDisputeGame.RootClaim(nil)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get root claim for game: %w", err)
	}

	l1Proxy.ConnectionState.ProxyConnection++
	l2blockNumber, err := faultDisputeGame.L2BlockNumber(nil)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get l2 block number for game: %w", err)
	}

	l1Proxy.ConnectionState.ProxyConnection++
	l2ChainID, err := faultDisputeGame.L2ChainId(nil)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get l2 chain id for game: %w", err)
	}

	l1Proxy.ConnectionState.ProxyConnection++
	gameStatus, err := faultDisputeGame.Status(nil)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get game status: %w", err)
	}

	l1Proxy.ConnectionState.ProxyConnection++
	createdAt, err := faultDisputeGame.CreatedAt(nil)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get game created at: %w", err)
	}

	l1Proxy.ConnectionState.ProxyConnection++
	resolvedAt, err := faultDisputeGame.ResolvedAt(nil)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get game resolved at: %w", err)
	}

	l1Proxy.ConnectionState.ProxyConnection++
	isBlacklisted, err := l1Proxy.optimismPortal2.DisputeGameBlacklist(nil, disputeGameProxyAddress)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get dispute game blacklist status: %w", err)
	}

	return &DisputeGame{
		DisputeGameRef: disputeGameRef,
		DisputeGameClaimData: &DisputeGameClaimData{
			RootClaim:     rootClaim,
			L2blockNumber: l2blockNumber,
			L2ChainID:     l2ChainID,
		},
		CreatedAt:         createdAt,
		ResolvedAt:        resolvedAt,
		GameStatus:        GameStatus(gameStatus),
		IsGameBlacklisted: isBlacklisted,
	}, nil
}

func (l1Proxy *L1Proxy) GetDisputeGameProxyUpdates(disputeGame *DisputeGame) (*DisputeGame, error) {

	disputeGameProxyAddress := disputeGame.DisputeGameRef.disputeGameProxyAddress

	l1Proxy.ConnectionState.ProxyConnection++
	faultDisputeGame, err := dispute.NewFaultDisputeGame(disputeGameProxyAddress, l1Proxy.l1GethClient)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to bind to dispute game: %w", err)
	}

	l1Proxy.ConnectionState.ProxyConnection++
	gameStatus, err := faultDisputeGame.Status(nil)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get game status: %w", err)
	}

	l1Proxy.ConnectionState.ProxyConnection++
	resolvedAt, err := faultDisputeGame.ResolvedAt(nil)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get game resolved at: %w", err)
	}

	l1Proxy.ConnectionState.ProxyConnection++
	isBlacklisted, err := l1Proxy.optimismPortal2.DisputeGameBlacklist(nil, disputeGameProxyAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get dispute game blacklist status: %w", err)
	}

	disputeGame.GameStatus = GameStatus(gameStatus)
	disputeGame.ResolvedAt = resolvedAt
	disputeGame.IsGameBlacklisted = isBlacklisted

	return disputeGame, nil
}

func (l1Proxy *L1Proxy) LatestHeight() (uint64, error) {
	l1Proxy.ConnectionState.ProxyConnection++
	block, err := l1Proxy.l1GethClient.BlockByNumber(*l1Proxy.ctx, nil)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}

	return block.NumberU64(), nil
}

func (l1Proxy *L1Proxy) BlockByNumber(blockNumber *big.Int) (*types.Block, error) {
	l1Proxy.ConnectionState.ProxyConnection++
	block, err := l1Proxy.l1GethClient.BlockByNumber(*l1Proxy.ctx, blockNumber)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("L1Proxy failed to get block by number: %d %w", blockNumber, err)
	}

	return block, nil
}

func (l1Proxy *L1Proxy) ChainID() (*big.Int, error) {
	l1Proxy.ConnectionState.ProxyConnection++
	chainID, err := l1Proxy.l1GethClient.ChainID(*l1Proxy.ctx)
	if err != nil {
		l1Proxy.ConnectionState.ProxyConnectionFailed++
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	return chainID, nil
}

func (l1Proxy *L1Proxy) Close() {
	l1Proxy.l1GethClient.Close()
}
