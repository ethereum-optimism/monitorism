package multisig

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ethereum-optimism/monitorism/op-monitorism/multisig/bindings"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "multisig_mon"
	SafeNonceABI     = "nonce()"

	OPTokenEnvName = "OP_SERVICE_ACCOUNT_TOKEN"

	// Item names follow a `ready-<nonce>.json` format
	PresignedNonceTitlePrefix = "ready-"
	PresignedNonceTitleSuffix = ".json"
)

var (
	SafeNonceSelector = crypto.Keccak256([]byte(SafeNonceABI))[:4]
)

type Monitor struct {
	log log.Logger

	l1Client *ethclient.Client

	optimismPortalAddress common.Address
	optimismPortal        *bindings.OptimismPortalCaller
	nickname              string

	onePassToken string
	onePassVault *string
	safeAddress  *common.Address

	// metrics
	safeNonce                 *prometheus.GaugeVec
	latestPresignedPauseNonce *prometheus.GaugeVec
	pausedState               *prometheus.GaugeVec
	unexpectedRpcErrors       *prometheus.CounterVec
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1 rpc: %w", err)
	}

	optimismPortal, err := bindings.NewOptimismPortalCaller(cfg.OptimismPortalAddress, l1Client)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to the OptimismPortal: %w", err)
	}

	if cfg.OnePassVault != nil && len(os.Getenv(OPTokenEnvName)) == 0 {
		return nil, fmt.Errorf("%s ENV name must be set for 1Pass integration", OPTokenEnvName)
	}

	if cfg.OnePassVault == nil {
		log.Warn("one pass integration is not configured")
	}
	if cfg.SafeAddress == nil {
		log.Warn("safe integration is not configured")
	}

	return &Monitor{
		log:      log,
		l1Client: l1Client,

		optimismPortal:        optimismPortal,
		optimismPortalAddress: cfg.OptimismPortalAddress,
		nickname:              cfg.Nickname,

		safeAddress:  cfg.SafeAddress,
		onePassVault: cfg.OnePassVault,

		safeNonce: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "safeNonce",
			Help:      "Safe Nonce",
		}, []string{"address", "nickname"}),
		latestPresignedPauseNonce: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "latestPresignedPauseNonce",
			Help:      "Latest pre-signed pause nonce",
		}, []string{"address", "nickname"}),
		pausedState: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "pausedState",
			Help:      "OptimismPortal paused state",
		}, []string{"address", "nickname"}),
		unexpectedRpcErrors: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "unexpectedRpcErrors",
			Help:      "number of unexpcted rpc errors",
		}, []string{"section", "name"}),
	}, nil
}

func (m *Monitor) Run(ctx context.Context) {
	m.checkOptimismPortal(ctx)
	m.checkSafeNonce(ctx)
	m.checkPresignedNonce(ctx)
}

func (m *Monitor) checkOptimismPortal(ctx context.Context) {
	paused, err := m.optimismPortal.Paused(&bind.CallOpts{Context: ctx})
	if err != nil {
		m.log.Error("failed to query OptimismPortal paused status", "err", err)
		m.unexpectedRpcErrors.WithLabelValues("optimismportal", "paused").Inc()
		return
	}

	pausedMetric := 0
	if paused {
		pausedMetric = 1
	}

	m.pausedState.WithLabelValues(m.optimismPortalAddress.String(), m.nickname).Set(float64(pausedMetric))
	m.log.Info("OptimismPortal status", "address", m.optimismPortalAddress.String(), "paused", paused)
}

func (m *Monitor) checkSafeNonce(ctx context.Context) {
	if m.safeAddress == nil {
		m.log.Warn("safe address is not configured, skipping...")
		return
	}

	nonceBytes := hexutil.Bytes{}
	nonceTx := map[string]interface{}{"to": *m.safeAddress, "data": hexutil.Encode(SafeNonceSelector)}
	if err := m.l1Client.Client().CallContext(ctx, &nonceBytes, "eth_call", nonceTx, "latest"); err != nil {
		m.log.Error("failed to query safe nonce", "err", err)
		m.unexpectedRpcErrors.WithLabelValues("safe", "nonce()").Inc()
		return
	}

	nonce := new(big.Int).SetBytes(nonceBytes).Uint64()
	m.safeNonce.WithLabelValues(m.safeAddress.String(), m.nickname).Set(float64(nonce))
	m.log.Info("Safe Nonce", "address", m.safeAddress.String(), "nonce", nonce)
}

func (m *Monitor) checkPresignedNonce(ctx context.Context) {
	if m.onePassVault == nil {
		m.log.Warn("one pass integration is not configured, skipping...")
		return
	}

	cmd := exec.CommandContext(ctx, "op", "item", "list", "--format=json", fmt.Sprintf("--vault=%s", *m.onePassVault))

	output, err := cmd.Output()
	if err != nil {
		m.log.Error("failed to run op cli")
		m.unexpectedRpcErrors.WithLabelValues("1pass", "exec").Inc()
		return
	}

	vaultItems := []struct{ Title string }{}
	if err := json.Unmarshal(output, &vaultItems); err != nil {
		m.log.Error("failed to unmarshal op cli stdout", "err", err)
		m.unexpectedRpcErrors.WithLabelValues("1pass", "stdout").Inc()
		return
	}

	latestPresignedNonce := int64(-1)
	for _, item := range vaultItems {
		if strings.HasPrefix(item.Title, PresignedNonceTitlePrefix) && strings.HasSuffix(item.Title, PresignedNonceTitleSuffix) {
			nonceStr := item.Title[len(PresignedNonceTitlePrefix) : len(item.Title)-len(PresignedNonceTitleSuffix)]
			nonce, err := strconv.ParseInt(nonceStr, 10, 64)
			if err != nil {
				m.log.Error("failed to parse nonce from item title", "title", item.Title)
				m.unexpectedRpcErrors.WithLabelValues("1pass", "title").Inc()
				return
			}
			if nonce > latestPresignedNonce {
				latestPresignedNonce = nonce
			}
		}
	}

	m.latestPresignedPauseNonce.WithLabelValues(m.safeAddress.String(), m.nickname).Set(float64(latestPresignedNonce))
	if latestPresignedNonce == -1 {
		m.log.Error("no presigned nonce found")
		return
	}

	m.log.Info("Latest Presigned Nonce", "nonce", latestPresignedNonce)
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	return nil
}
