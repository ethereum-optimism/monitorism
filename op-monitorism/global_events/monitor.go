package global_events

import (
	"context"
	// "os"
	// "os"
	// "encoding/json"
	"fmt"
	// "math/big"
	// "os"
	// "os/exec"
	// "strconv"
	// "strings"
	// "github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"regexp"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum"

	// "github.com/ethereum/go-ethereum/accounts/abi/bind"
	// "github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricsNamespace = "global_events_mon"

	// OPTokenEnvName = "OP_SERVICE_ACCOUNT_TOKEN"

	// Item names follow a `ready-<nonce>.json` format
	// PresignedNonceTitlePrefix = "ready-"
	// PresignedNonceTitleSuffix = ".json"
)

type Monitor struct {
	log log.Logger

	l1Client               *ethclient.Client
	TabMonitoringAddresses TabMonitoringAddress

	nickname string

	filename   string //filename of the yaml rules
	yamlconfig Configuration

	// metrics
	safeNonce                 *prometheus.GaugeVec
	latestPresignedPauseNonce *prometheus.GaugeVec
	pausedState               *prometheus.GaugeVec
	unexpectedRpcErrors       *prometheus.CounterVec
}

func ChainIDToName(chainID int64) string {
	switch chainID {
	case 1:
		return "Ethereum [Mainnet]"
	case 11155111:
		return "Sepolia [Testnet]"
	}
	return "The `ChainID` is Not defined into the `chaindIDToName` function, this is probably a custom chain otherwise something is going wrong!"
}

func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1 rpc: %w", err)
	}
	fmt.Printf("--------------------------------------- Global_events_mon (Infos) -----------------------------\n")
	// fmt.Printf("chainID:", ChainIDToName(l1Client.ChainID())
	ChainID, err := l1Client.ChainID(context.Background())
	if err != nil {
		log.Crit("Failed to retrieve chain ID: %v", err)
	}
	header, err := l1Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Crit("Failed to fetch the latest block header: %v", err)
	}
	// display the infos at the start to ensure everything is correct.
	fmt.Printf("latestBlockNumber: %s\n", header.Number)
	fmt.Printf("chainId: %+v\n", ChainIDToName(ChainID.Int64()))
	fmt.Printf("PathYaml: %v\n", cfg.PathYamlRules)
	fmt.Printf("Nickname: %v\n", cfg.Nickname)
	fmt.Printf("L1NodeURL: %v\n", cfg.L1NodeURL)
	TabMonitoringAddresses := ReadAllYamlRules(cfg.PathYamlRules)
	// yamlconfig := ReadYamlFile(cfg.PathYamlRules)
	fmt.Printf("Number of Addresses monitored (for now don't take in consideration the duplicates): %v\n", len(TabMonitoringAddresses.GetUniqueMonitoredAddresses()))
	fmt.Printf("Number of Events monitored (for now don't take in consideration the duplicates): %v\n", len(TabMonitoringAddresses.GetMonitoredEvents()))

	// MonitoringAddresses := fromConfigurationToAddress(yamlconfig)
	// DisplayMonitorAddresses(MonitoringAddresses)
	// Should I make a sleep of 10 seconds to ensure we can read this information before the prod?
	fmt.Printf("--------------------------------------- End of Infos -----------------------------\n")
	// fmt.Printf("YAML Config: %v\n", yamlconfig)
	return &Monitor{
		log:                    log,
		l1Client:               l1Client,
		TabMonitoringAddresses: TabMonitoringAddresses,

		nickname: cfg.Nickname,
		// yamlconfig: yamlconfig,
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
func formatSignature(signature string) string {
	// Regex to extract function name and parameters
	r := regexp.MustCompile(`(\w+)\s*\(([^)]*)\)`)
	matches := r.FindStringSubmatch(signature)

	if len(matches) != 3 {
		return ""
	}

	// Function name
	funcName := matches[1]
	// Parameters, split by commas
	params := matches[2]
	// Clean parameters to keep only types
	cleanParams := make([]string, 0)
	for _, param := range strings.Split(params, ",") {
		parts := strings.Fields(param)
		if len(parts) > 0 {
			cleanParams = append(cleanParams, parts[0])
		}
	}

	// Return formatted function signature
	return fmt.Sprintf("%s(%s)", funcName, strings.Join(cleanParams, ","))
}

// Format And Hash the signature to create the topic.
// Formatting allows use to use "transfer(address owner, uint256 amount)" instead of "transfer(address,uint256
// So with the name parameters

func FormatAndHash(signature string) common.Hash { // this will return the topic not the 4bytes so longer.
	formattedSignature := formatSignature(signature)
	if formattedSignature == "" {
		panic("Invalid signature")
	}
	hash := crypto.Keccak256([]byte(formattedSignature))
	return common.BytesToHash(hash)

}

func (m *Monitor) Run(ctx context.Context) {
	m.checkEvents(ctx)
	// input := "balanceOf(address owner)"
	// formattedSignature := formatSignature(input)
	// if formattedSignature == "" {
	// 	panic("Invalid signature")
	// }
	// hash := crypto.Keccak256([]byte(formattedSignature))
	// fmt.Printf("Function Selector: 0x%x\n", hash[:4])
	// // m.checkSafeNonce(ctx)
	// // m.checkPresignedNonce(ctx)
}
func (m *Monitor) checkEvents(ctx context.Context) {
	header, err := m.l1Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Crit("Failed to retrieve latest block header: %v", err)
	}
	latestBlockNumber := header.Number
	// fmt.Printf("Get the list of the addresses we are going to monitore\n", m.TabMonitoringAdHexToHashHashetUniqueMonitoredAddresses())
	query := ethereum.FilterQuery{
		FromBlock: latestBlockNumber,
		ToBlock:   latestBlockNumber,
		Addresses: m.TabMonitoringAddresses.GetUniqueMonitoredAddresses(), //if empty means that all addresses are monitored!
	}
	// os.Exit(0)
	logs, err := m.l1Client.FilterLogs(context.Background(), query)
	if err != nil {
		m.log.Crit("Failed to retrieve logs: %v", err)
	}

	fmt.Printf("-------------------------- START OF BLOCK (%s)--------------------------------------", latestBlockNumber) // Prints the log data; consider using `vLog.Topics` or `vLog.Data`
	fmt.Println("Block Number: ", latestBlockNumber)
	fmt.Println("Number of logs: ", len(logs))
	fmt.Println("BlockHash:", header.Hash().Hex())

	for _, vLog := range logs {
		// if vlog.Topics == topics_toml {
		// 	// alerting + 1
		// }
		if IsTopicInMonitoredEvents(vLog.Topics, m.TabMonitoringAddresses.GetMonitoredEvents()) {
			fmt.Printf("----------------------------------------------------------------\n")           // Prints the log data; consider using `vLog.Topics` or `vLog.Data`
			fmt.Printf("TxHash: %s\nAddress:%s\nTopics: %s\n", vLog.TxHash, vLog.Address, vLog.Topics) // Prints the log data; consider using `vLog.Topics` or `vLog.Data`
			fmt.Printf("----------------------------------------------------------------\n")           // Prints the log data; consider using `vLog.Topics` or `vLog.Data`
		}

	}

	fmt.Printf("-------------------------- END OF BLOCK (%s)--------------------------------------", latestBlockNumber) // Prints the log data; consider using `vLog.Topics` or `vLog.Data`
	// paused, err := m.optimismPortal.Paused(&bind.CallOpts{Context: ctx})
	// if err != nil {
	// 	m.log.Error("failed to query OptimismPortal paused status", "err", err)
	// 	m.unexpectedRpcErrors.WithLabelValues("optimismportal", "paused").Inc()
	// 	return
	// }

	// pausedMetric := 0
	// if paused {
	// 	pausedMetric = 1
	// }
	//
	// m.pausedState.WithLabelValues(m.optimismPortalAddress.String(), m.nickname).Set(float64(pausedMetric))
	// m.log.Info("OptimismPortal status", "address", m.optimismPortalAddress.String(), "paused", paused)
	m.log.Info("Checking events")
}
func IsTopicInMonitoredEvents(topics []common.Hash, monitoredEvents []Event) bool {
	for _, monitoredEvent := range monitoredEvents {
		fmt.Printf("Monitored Event: %v\n", monitoredEvent._4bytes)
		fmt.Printf("Topics: %v\n", topics[0])
		if monitoredEvent._4bytes == topics[0] {
			return true
		}
	}
	return false
}

// func (m *Monitor) checkSafeNonce(ctx context.Context) {
// 	if m.safeAddress == nil {
// 		m.log.Warn("safe address is not configured, skipping...")
// 		return
// 	}
//
// 	nonceBytes := hexutil.Bytes{}
// 	nonceTx := map[string]interface{}{"to": *m.safeAddress, "data": hexutil.Encode(SafeNonceSelector)}
// 	if err := m.l1Client.Client().CallContext(ctx, &nonceBytes, "eth_call", nonceTx, "latest"); err != nil {
// 		m.log.Error("failed to query safe nonce", "err", err)
// 		m.unexpectedRpcErrors.WithLabelValues("safe", "nonce()").Inc()
// 		return
// 	}
//
// 	nonce := new(big.Int).SetBytes(nonceBytes).Uint64()
// 	m.safeNonce.WithLabelValues(m.safeAddress.String(), m.nickname).Set(float64(nonce))
// 	m.log.Info("Safe Nonce", "address", m.safeAddress.String(), "nonce", nonce)
// }

//	func (m *Monitor) checkPresignedNonce(ctx context.Context) {
//		if m.onePassVault == nil {
//			m.log.Warn("one pass integration is not configured, skipping...")
//			return
//		}
//
//		cmd := exec.CommandContext(ctx, "op", "item", "list", "--format=json", fmt.Sprintf("--vault=%s", *m.onePassVault))
//
//		output, err := cmd.Output()
//		if err != nil {
//			m.log.Error("failed to run op cli")
//			m.unexpectedRpcErrors.WithLabelValues("1pass", "exec").Inc()
//			return
//		}
//
//		vaultItems := []struct{ Title string }{}
//		if err := json.Unmarshal(output, &vaultItems); err != nil {
//			m.log.Error("failed to unmarshal op cli stdout", "err", err)
//			m.unexpectedRpcErrors.WithLabelValues("1pass", "stdout").Inc()
//			return
//		}
//
//		latestPresignedNonce := int64(-1)
//		for _, item := range vaultItems {
//			if strings.HasPrefix(item.Title, PresignedNonceTitlePrefix) && strings.HasSuffix(item.Title, PresignedNonceTitleSuffix) {
//				nonceStr := item.Title[len(PresignedNonceTitlePrefix) : len(item.Title)-len(PresignedNonceTitleSuffix)]
//				nonce, err := strconv.ParseInt(nonceStr, 10, 64)
//				if err != nil {
//					m.log.Error("failed to parse nonce from item title", "title", item.Title)
//					m.unexpectedRpcErrors.WithLabelValues("1pass", "title").Inc()
//					return
//				}
//				if nonce > latestPresignedNonce {
//					latestPresignedNonce = nonce
//				}
//			}
//		}
//
//		m.latestPresignedPauseNonce.WithLabelValues(m.safeAddress.String(), m.nickname).Set(float64(latestPresignedNonce))
//		if latestPresignedNonce == -1 {
//			m.log.Error("no presigned nonce found")
//			return
//		}
//
//		m.log.Info("Latest Presigned Nonce", "nonce", latestPresignedNonce)
//	}
func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	return nil
}
