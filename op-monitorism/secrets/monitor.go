package secrets

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/monitorism/op-monitorism/secrets/bindings"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "secrets_mon"

	// ABI for CheckSecretsParams struct
	CheckSecretsParamsABI = `[{"constant":true,"inputs":[],"name":"getTuple","outputs":[{"components":[{"name":"delay","type":"uint256"},{"name":"secretHashMustExist","type":"bytes32"},{"name":"secretHashMustNotExist","type":"bytes32"}],"name":"","type":"tuple"}],"payable":false,"stateMutability":"view","type":"function"}]`
)

type Monitor struct {
	log log.Logger

	l1Client *ethclient.Client

	drippieAddress common.Address
	drippie        *bindings.Drippie
	drips          map[string]*bindings.DrippieDripConfig
	created        []string

	// Metrics
	highestBlockNumber     *prometheus.GaugeVec
	revealedSecrets        *prometheus.GaugeVec
	nodeConnectionFailures *prometheus.CounterVec
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	log.Info("creating secrets monitor...")

	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1: %w", err)
	}

	drippie, err := bindings.NewDrippie(cfg.DrippieAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to Drippie: %w", err)
	}

	return &Monitor{
		log: log,

		l1Client: l1Client,

		drippieAddress: cfg.DrippieAddress,
		drippie:        drippie,
		drips:          make(map[string]*bindings.DrippieDripConfig),

		// Metrics
		highestBlockNumber: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "highestBlockNumber",
			Help:      "observed l1 heights (checked and known)",
		}, []string{"type"}),
		revealedSecrets: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "revealedSecrets",
			Help:      "revealed secrets",
		}, []string{"type", "drip", "hash"}),
		nodeConnectionFailures: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "nodeConnectionFailures",
			Help:      "number of times node connection has failed",
		}, []string{"layer", "section"}),
	}, nil
}

func (m *Monitor) Run(ctx context.Context) {
	// Determine current L1 block height.
	latestL1Height, err := m.l1Client.BlockNumber(ctx)
	if err != nil {
		m.log.Error("failed to query latest block number", "err", err)
		m.nodeConnectionFailures.WithLabelValues("l1", "blockNumber").Inc()
		return
	}

	// Update metrics.
	m.highestBlockNumber.WithLabelValues("known").Set(float64(latestL1Height))

	// Set up the call options once.
	callOpts := bind.CallOpts{
		BlockNumber: big.NewInt(int64(latestL1Height)),
	}

	// Grab the number of created drips at the current block height.
	numCreated, err := m.drippie.GetDripCount(&callOpts)
	if err != nil {
		m.log.Error("failed to query Drippie for number of created drips", "err", err)
		m.nodeConnectionFailures.WithLabelValues("l1", "dripCount").Inc()
		return
	}

	// Add new drip names if the number of created drips has increased.
	if numCreated.Cmp(big.NewInt(int64(len(m.created)))) >= 0 {
		// Iterate through the new drip indices and add their names to the stored list.
		// TODO: You can optimize this with a multicall. Current code is good enough for now since we
		// don't expect a large number of drips to be created. If this changes, consider multicall to
		// batch the requests into a single call.
		for i := len(m.created); i < int(numCreated.Int64()); i++ {
			// Log so we know what's happening.
			m.log.Info("pulling name for new drip index", "index", i)

			// Grab the name of the drip at the current index.
			name, err := m.drippie.Created(&callOpts, big.NewInt(int64(i)))
			if err != nil {
				m.log.Error("failed to query Drippie for Drip name", "index", i, "err", err)
				m.nodeConnectionFailures.WithLabelValues("l1", "dripName").Inc()
				return
			}

			// Log the name of the new drip.
			m.log.Info("pulling config for new drip", "index", i, "name", name)

			// Add the name to the list of created drips.
			m.created = append(m.created, name)

			// Get the drip configuration.
			drip, err := m.drippie.Drips(&callOpts, name)
			if err != nil {
				m.log.Error("failed to query Drippie for Drip", "name", name, "err", err)
				m.nodeConnectionFailures.WithLabelValues("l1", "drips").Inc()
				return
			}

			// Bind to the DripCheck contract.
			dripcheck, err := bindings.NewCheckSecretsCaller(drip.Config.Dripcheck, m.l1Client)
			if err != nil {
				m.log.Error("failed to bind to DripCheck", "name", name, "address", drip.Config.Dripcheck, "err", err)
				m.nodeConnectionFailures.WithLabelValues("l1", "dripCheck").Inc()
				return
			}

			// Get the name of the DripCheck contract.
			checkname, err := dripcheck.Name(&callOpts)
			if err != nil {
				m.log.Error("failed to get name of DripCheck", "name", name, "address", drip.Config.Dripcheck, "err", err)
				m.nodeConnectionFailures.WithLabelValues("l1", "dripCheckName").Inc()
				return
			}

			// If the DripCheck contract is a CheckSecrets contract, store the config.
			if checkname == "CheckSecrets" {
				m.log.Info("DripCheck is a CheckSecrets contract", "name", name, "checkname", checkname)
				m.drips[name] = &drip.Config
			}
		}
	} else {
		// Should not happen, log an error and reset the created drips.
		m.log.Error("number of created drips decreased", "old", len(m.created), "new", numCreated)
		m.created = nil
		return
	}

	// Iterate through all stored drips and check if the secrets have been revealed.
	// TODO: You can optimize this with a multicall. Current code is good enough for now since we
	// don't expect a large number of drips to be created. If this changes, consider multicall to
	// batch the requests into a single call.
	for name, config := range m.drips {
		// Check if this drip has been archived.
		status, err := m.drippie.GetDripStatus(&callOpts, name)
		if err != nil {
			m.log.Error("failed to query Drippie for Drip status", "name", name, "err", err)
			m.nodeConnectionFailures.WithLabelValues("l1", "dripStatus").Inc()
			return
		}

		// If the drip has been paused or archived then we can skip checking it for now.
		if status == 1 || status == 3 {
			continue
		}

		// Bind to the DripCheck contract.
		dripcheck, err := bindings.NewCheckSecretsCaller(config.Dripcheck, m.l1Client)
		if err != nil {
			m.log.Error("failed to bind to DripCheck", "name", name, "address", config.Dripcheck, "err", err)
			m.nodeConnectionFailures.WithLabelValues("l1", "dripCheck").Inc()
			return
		}

		// Get the ABI reader to read the CheckSecretsParams struct.
		checkparams := new(bindings.CheckSecretsParams)
		abidef, err := abi.JSON(strings.NewReader(CheckSecretsParamsABI))
		if err != nil {
			m.log.Error("failed to parse CheckSecretsParams ABI", "err", err)
			return
		}

		// Unpack the CheckSecretsParams struct.
		err = abidef.UnpackIntoInterface(&checkparams, "getTuple", config.Checkparams)
		if err != nil {
			m.log.Error("failed to unpack CheckSecretsParams", "err", err)
			return
		}

		// Check if the initation secret exists.
		secretHex1 := common.Bytes2Hex(checkparams.SecretHashMustExist[:])
		m.log.Info("checking initiation secret", "name", name, "hash", secretHex1)
		exists1, err := dripcheck.RevealedSecrets(&callOpts, checkparams.SecretHashMustExist)
		if err != nil {
			m.log.Error("failed to query CheckSecrets for initiation secret", "name", name, "err", err)
			m.nodeConnectionFailures.WithLabelValues("l1", "checkSecrets").Inc()
			return
		}

		// Update metrics.
		if exists1.Cmp(big.NewInt(0)) > 0 {
			m.log.Info("revealed initation secret", "name", name, "hash", secretHex1)
			m.revealedSecrets.WithLabelValues("initiation", name, secretHex1).Set(1)
		}

		// Check if the cancellation secret exists.
		secretHex2 := common.Bytes2Hex(checkparams.SecretHashMustNotExist[:])
		m.log.Info("checking cancellation secret", "name", name, "hash", secretHex2)
		exists2, err := dripcheck.RevealedSecrets(&callOpts, checkparams.SecretHashMustNotExist)
		if err != nil {
			m.log.Error("failed to query CheckSecrets for cancellation secret", "name", name, "err", err)
			m.nodeConnectionFailures.WithLabelValues("l1", "checkSecrets").Inc()
			return
		}

		// Update metrics.
		if exists2.Cmp(big.NewInt(0)) > 0 {
			m.log.Info("revealed cancellation secret", "name", name, "hash", secretHex2)
			m.revealedSecrets.WithLabelValues("cancellation", name, secretHex2).Set(1)
		}
	}
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	return nil
}
