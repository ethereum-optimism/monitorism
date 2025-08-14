package multisig

import (
	"context"
	"fmt"
	"strings"

	gsbindings "github.com/ethereum-optimism/monitorism/op-monitorism/liveness_expiration/bindings"
	"github.com/ethereum-optimism/optimism/op-service/metrics"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "multisig_registry"
)

type Monitor struct {
	log log.Logger

	l1Client *ethclient.Client
	nickname string

	// Notion configuration
	notionDatabaseID string
	notionToken      string
	notionProps      NotionProps

	// Metrics
	thresholdMismatch   *prometheus.GaugeVec
	onchainThreshold    *prometheus.GaugeVec
	notionThreshold     *prometheus.GaugeVec
	signerCountMismatch *prometheus.GaugeVec
	onchainSignerCount  *prometheus.GaugeVec
	notionSignerCount   *prometheus.GaugeVec
	safeAccessibleGauge *prometheus.GaugeVec
	unexpectedErrors    *prometheus.CounterVec
	totalSafesMonitored *prometheus.GaugeVec
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1 rpc: %w", err)
	}

	return &Monitor{
		log:              log,
		l1Client:         l1Client,
		nickname:         cfg.Nickname,
		notionDatabaseID: cfg.NotionDatabaseID,
		notionToken:      cfg.NotionToken,
		notionProps:      DefaultNotionProps(),

		thresholdMismatch: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "threshold_mismatch",
			Help:      "1 if Notion threshold differs from onchain, 0 if matches",
		}, []string{"safe_address", "safe_name", "nickname"}),

		onchainThreshold: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "onchain_threshold",
			Help:      "Current onchain Safe threshold",
		}, []string{"safe_address", "safe_name", "nickname"}),

		notionThreshold: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "notion_threshold",
			Help:      "Threshold recorded in Notion database",
		}, []string{"safe_address", "safe_name", "nickname"}),

		signerCountMismatch: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "signer_count_mismatch",
			Help:      "1 if Notion signer count differs from onchain, 0 if matches",
		}, []string{"safe_address", "safe_name", "nickname"}),

		onchainSignerCount: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "onchain_signer_count",
			Help:      "Current onchain Safe owner count",
		}, []string{"safe_address", "safe_name", "nickname"}),

		notionSignerCount: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "notion_signer_count",
			Help:      "Signer count recorded in Notion database",
		}, []string{"safe_address", "safe_name", "nickname"}),

		safeAccessibleGauge: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "safe_accessible",
			Help:      "1 if Safe contract is accessible, 0 if not",
		}, []string{"safe_address", "safe_name", "nickname"}),

		unexpectedErrors: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "unexpected_errors",
			Help:      "Number of unexpected errors during monitoring",
		}, []string{"error_type", "safe_address"}),

		totalSafesMonitored: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "total_safes_monitored",
			Help:      "Total number of Safes being monitored from Notion",
		}, []string{"nickname"}),
	}, nil
}

func (m *Monitor) Run(ctx context.Context) {
	m.log.Info("Starting Notion-based Safe monitoring cycle")

	// Step 1: Call QueryNotionSafes function from query.go
	safeRows, err := QueryNotionSafes(ctx, m.notionToken, m.notionDatabaseID, m.notionProps)
	if err != nil {
		m.log.Error("Failed to query Notion database", "err", err)
		m.unexpectedErrors.WithLabelValues("notion_query", "").Inc()
		return
	}

	m.log.Info("Retrieved Safe records from Notion", "count", len(safeRows))
	m.totalSafesMonitored.WithLabelValues(m.nickname).Set(float64(len(safeRows)))

	// Step 2: Iterate over each Safe and check that values are correct
	for _, safeRow := range safeRows {
		m.monitorSafe(ctx, safeRow)
	}

	m.log.Info("Completed monitoring cycle", "safes_checked", len(safeRows))
}

func (m *Monitor) monitorSafe(ctx context.Context, safeRow NotionSafeRow) {
	addr := strings.TrimSpace(safeRow.Address)
	safeName := safeRow.Name
	if safeName == "" {
		safeName = "unnamed"
		m.log.Info("The safe name is empty for the safe address:", "address", addr)
	}

	// Validate address format
	if !common.IsHexAddress(addr) {
		m.log.Error("Invalid Safe address in Notion", "address", addr, "name", safeName)
		m.unexpectedErrors.WithLabelValues("invalid_address", addr).Inc()
		return
	}

	safeAddress := common.HexToAddress(addr)
	addrStr := safeAddress.Hex()

	// Create Safe contract binding
	safeCaller, err := gsbindings.NewGnosisSafeCaller(safeAddress, m.l1Client)
	if err != nil {
		m.log.Error("Failed to create Safe contract binding", "address", addrStr, "name", safeName, "err", err)
		m.unexpectedErrors.WithLabelValues("binding_error", addrStr).Inc()
		m.safeAccessibleGauge.WithLabelValues(addrStr, safeName, m.nickname).Set(0)
		return
	}

	// Check threshold
	m.checkThreshold(ctx, safeCaller, safeRow, addrStr, safeName)

	// Check signer count
	m.checkSignerCount(ctx, safeCaller, safeRow, addrStr, safeName)
	// TODO: Check the amount of held tokens + native tokens of the multisig
	//m.checkTokenBalance(ctx, safeCaller, safeRow, addrStr, safeName)

	m.safeAccessibleGauge.WithLabelValues(addrStr, safeName, m.nickname).Set(1)
}

func (m *Monitor) checkThreshold(ctx context.Context, safeCaller *gsbindings.GnosisSafeCaller, safeRow NotionSafeRow, addrStr, safeName string) {
	onchainThreshold, err := safeCaller.GetThreshold(&bind.CallOpts{Context: ctx})
	if err != nil {
		m.log.Error("Failed to get Safe threshold", "address", addrStr, "name", safeName, "err", err)
		m.unexpectedErrors.WithLabelValues("threshold_query", addrStr).Inc()
		return
	}

	onchainThr := onchainThreshold.Uint64()
	notionThr := uint64(safeRow.Threshold)

	// Update metrics
	m.onchainThreshold.WithLabelValues(addrStr, safeName, m.nickname).Set(float64(onchainThr))
	m.notionThreshold.WithLabelValues(addrStr, safeName, m.nickname).Set(float64(notionThr))

	if onchainThr != notionThr {
		m.thresholdMismatch.WithLabelValues(addrStr, safeName, m.nickname).Set(1)
		m.log.Error("Threshold mismatch detected",
			"name", safeName,
			"address", addrStr,
			"onchain_threshold", onchainThr,
			"notion_threshold", notionThr,
			"networks", safeRow.Networks,
			"risk", safeRow.Risk,
			"multisig_lead", safeRow.MultisigLead)
	} else {
		m.thresholdMismatch.WithLabelValues(addrStr, safeName, m.nickname).Set(0)
		m.log.Info("Threshold matches",
			"name", safeName,
			"address", addrStr,
			"threshold", onchainThr)
	}
}

func (m *Monitor) checkSignerCount(ctx context.Context, safeCaller *gsbindings.GnosisSafeCaller, safeRow NotionSafeRow, addrStr, safeName string) {
	owners, err := safeCaller.GetOwners(&bind.CallOpts{Context: ctx})
	if err != nil {
		m.log.Error("Failed to get Safe owners", "address", addrStr, "name", safeName, "err", err)
		m.unexpectedErrors.WithLabelValues("owners_query", addrStr).Inc()
		return
	}

	onchainSignerCount := uint64(len(owners))
	notionSignerCount := uint64(safeRow.SignerCount)

	// Update metrics
	m.onchainSignerCount.WithLabelValues(addrStr, safeName, m.nickname).Set(float64(onchainSignerCount))
	m.notionSignerCount.WithLabelValues(addrStr, safeName, m.nickname).Set(float64(notionSignerCount))

	if notionSignerCount > 0 && onchainSignerCount != notionSignerCount {
		m.signerCountMismatch.WithLabelValues(addrStr, safeName, m.nickname).Set(1)
		m.log.Warn("Signer count mismatch detected",
			"name", safeName,
			"address", addrStr,
			"onchain_signers", onchainSignerCount,
			"notion_signers", notionSignerCount,
			"multisig_lead", safeRow.MultisigLead)
	} else {
		m.signerCountMismatch.WithLabelValues(addrStr, safeName, m.nickname).Set(0)
		if notionSignerCount > 0 {
			m.log.Info("Signer count matches",
				"name", safeName,
				"address", addrStr,
				"signer_count", onchainSignerCount)
		}
	}
}

func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	return nil
}
