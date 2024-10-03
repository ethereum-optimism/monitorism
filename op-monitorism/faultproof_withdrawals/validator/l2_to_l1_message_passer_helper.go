package validator

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/l2"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/ethclient"
)

// L2ToL1MessagePasserHelper assists in interacting with the L2 to L1 message passer.
type L2ToL1MessagePasserHelper struct {
	l2Client            *ethclient.Client       // The L2 Ethereum client.
	l2ToL1MessagePasser *l2.L2ToL1MessagePasser // The L2 to L1 message passer contract instance.

	ctx context.Context // Context for managing cancellation and timeouts.
}

// NewL2ToL1MessagePasserHelper initializes a new L2ToL1MessagePasserHelper.
// It binds to the L2 to L1 message passer contract and returns the helper instance.
func NewL2ToL1MessagePasserHelper(ctx context.Context, l2Client *ethclient.Client) (*L2ToL1MessagePasserHelper, error) {
	l2ToL1MessagePasser, err := l2.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, l2Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to L2ToL1MessagePasser: %w", err)
	}

	return &L2ToL1MessagePasserHelper{
		l2Client:            l2Client,
		l2ToL1MessagePasser: l2ToL1MessagePasser,
		ctx:                 ctx,
	}, nil
}

// WithdrawalExistsOnL2 checks if a withdrawal message with the given hash exists on L2.
// It returns true if the message exists, otherwise returns false along with any error encountered.
func (l2l1 *L2ToL1MessagePasserHelper) WithdrawalExistsOnL2(withdrawalHash eth.Bytes32) (bool, error) {
	return l2l1.l2ToL1MessagePasser.L2ToL1MessagePasserCaller.SentMessages(nil, withdrawalHash)
}
