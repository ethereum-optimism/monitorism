package faultproof_withdrawals

import (
	"context"

	"github.com/ethereum-optimism/monitorism/op-monitorism/faultproof_withdrawals/validator"
)

type WithdrawalValidator struct {
	ctx     *context.Context
	L1Proxy validator.L1ProxyInterface
	L2Proxy validator.L2ProxyInterface
}

// NewWithdrawalValidator creates a new Validator instance with the provided L1 and L2 proxies.
func NewWithdrawalValidator(ctx *context.Context, l1Proxy validator.L1ProxyInterface, l2Proxy validator.L2ProxyInterface) (*WithdrawalValidator, error) {
	return &WithdrawalValidator{
		L1Proxy: l1Proxy,
		L2Proxy: l2Proxy,
		ctx:     ctx,
	}, nil
}

func (v *WithdrawalValidator) GetRange(blockStart uint64, blockEnd uint64) ([]validator.WithdrawalValidationRef, error) {
	disputeGamesEvents, err := v.L1Proxy.GetDisputeGamesEvents(blockStart, blockEnd)
	if err != nil {
		return nil, err
	}

	withdrawalValidations := make([]validator.WithdrawalValidationRef, 0)
	for _, disputeGameEvent := range disputeGamesEvents {

		withdrawalValidation, err := v.L2Proxy.GetWithdrawalValidation(disputeGameEvent)
		if err != nil {
			return nil, err
		}

		withdrawalValidations = append(withdrawalValidations, *withdrawalValidation)
	}

	return withdrawalValidations, nil
}
