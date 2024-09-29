package validator

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/bindings/l2"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/ethclient"
)

type L2ToL1MessagePasserHelper struct {
	l2Client            *ethclient.Client
	l2ToL1MessagePasser *l2.L2ToL1MessagePasser

	ctx context.Context
}

func NewL2ToL1MessagePasserHelper(ctx context.Context, l2Client *ethclient.Client) (*L2ToL1MessagePasserHelper, error) {

	l2ToL1MessagePasser, err := l2.NewL2ToL1MessagePasser(predeploys.L2ToL1MessagePasserAddr, l2Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to dispute game factory: %w", err)
	}

	return &L2ToL1MessagePasserHelper{
		l2Client:            l2Client,
		l2ToL1MessagePasser: l2ToL1MessagePasser,

		ctx: ctx,
	}, nil
}

func (l2l1 *L2ToL1MessagePasserHelper) WithdrawalExistsOnL2(withdrawalHash [32]byte) (bool, error) {
	return l2l1.l2ToL1MessagePasser.L2ToL1MessagePasserCaller.SentMessages(nil, withdrawalHash)
}
