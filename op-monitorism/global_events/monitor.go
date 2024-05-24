package global_events

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	git "github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

const (
	MetricsNamespace = "global_events_mon"
)

// Monitor is the main struct of the monitor.
type Monitor struct {
	log log.Logger

	l1Client     *ethclient.Client
	globalconfig GlobalConfiguration
	// nickname is the nickname of the monitor (we need to change the name this is not an ideal one here).
	nickname    string
	safeAddress *bindings.OptimismPortalCaller

	LiveAddress *common.Address

	filename   string //filename of the yaml rules
	yamlconfig Configuration

	// Prometheus metrics
	eventEmitted        *prometheus.GaugeVec
	unexpectedRpcErrors *prometheus.CounterVec
}

// ChainIDToName() allows to convert the chainID to a human readable name.
// For now only ethereum + Sepolia are supported.
func ChainIDToName(chainID int64) string {
	switch chainID {
	case 1:
		return "Ethereum [Mainnet]"
	case 11155111:
		return "Sepolia [Testnet]"
	}
	return "The `ChainID` is Not defined into the `chaindIDToName` function, this is probably a custom chain otherwise something is going wrong!"
}

func cloneRepo(repoURL string, path string) error { //for debug purpose will be store to /tmp/Monitorism/rules_l1
	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: plumbing.NewBranchReferenceName("main"), // Replace 'main' with your branch if different
		SingleBranch:  true,
		Depth:         1, // Use a shallow clone to speed up the process
	})
	if err != nil {

		fmt.Printf("Error while cloning the repo: %v", err)
	}
	return err
}

// NewMonitor creates a new Monitor instance.
func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	cloneRepo("https://github.com/ethereum-optimism/monitorism.git", "/tmp/Monitorism/") //rules are located
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
	globalConfig := ReadAllYamlRules(cfg.PathYamlRules)
	fmt.Printf("GlobalConfig: %#v\n", globalConfig.Configuration)
	globalConfig.DisplayMonitorAddresses()
	fmt.Printf("--------------------------------------- End of Infos -----------------------------\n")
	time.Sleep(10 * time.Second) // sleep for 10 seconds usefull to read the information before the prod.
	return &Monitor{
		log:          log,
		l1Client:     l1Client,
		globalconfig: globalConfig,

		nickname: cfg.Nickname,
		eventEmitted: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "eventEmitted",
			Help:      "Event monitored emitted an log",
		}, []string{"nickname", "rulename", "priority", "functionName", "address"}),
		unexpectedRpcErrors: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "unexpectedRpcErrors",
			Help:      "number of unexpcted rpc errors",
		}, []string{"section", "name"}),
	}, nil
}

// formatSignature allows to format the signature of a function to be able to hash it.
// e.g: "transfer(address owner, uint256 amount)" -> "transfer(address,uint256)"
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

// FormatAndHash allow to Format the signature (e.g: "transfer(address,uint256)") to create the keccak256 hash associated with it.
// Formatting allows use to use "transfer(address owner, uint256 amount)" instead of "transfer(address,uint256)"
func FormatAndHash(signature string) common.Hash {
	formattedSignature := formatSignature(signature)
	if formattedSignature == "" {
		panic("Invalid signature")
	}
	hash := crypto.Keccak256([]byte(formattedSignature))
	return common.BytesToHash(hash)

}

func (m *Monitor) Run(ctx context.Context) {
	m.checkEvents(ctx)
	//  m.SignerCanBeRemove
}

func (m *Monitor) SignerCanBeRemove(ctx context.Context) { //TODO: Ensure the logs crit are not causing panic in runtime!
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
	//
	// 	m.pausedState.WithLabelValues(m.optimismPortalAddress.String(), m.nickname).Set(float64(pausedMetric))
	// 	m.log.Info("OptimismPortal status", "address", m.optimismPortalAddress.String(), "paused", paused)
	// 	m.log.Info("Checking if the signer can be SignerCanBeRemove..")

}

// checkEvents function to check the events. If an events is emitted onchain and match the rules defined in the yaml file, then we will display the event.
func (m *Monitor) checkEvents(ctx context.Context) { //TODO: Ensure the logs crit are not causing panic in runtime!
	header, err := m.l1Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		m.log.Warn("Failed to retrieve latest block header: %v", err) //TODO:need to wait 12 and retry here!
	}

	latestBlockNumber := header.Number
	query := ethereum.FilterQuery{
		FromBlock: latestBlockNumber,
		ToBlock:   latestBlockNumber,
		// Addresses: []common.Address{}, //if empty means that all addresses are monitored should be this value for optimisation and avoiding to take every logs every time -> m.globalconfig.GetUniqueMonitoredAddresses
	}

	logs, err := m.l1Client.FilterLogs(context.Background(), query)
	if err != nil { //TODO:need to wait 12 and retry here!
		m.log.Warn("Failed to retrieve logs: %v", err)
	}

	for _, vLog := range logs {
		if len(vLog.Topics) > 0 { //Ensure no anonymous event is here.
			if len(m.globalconfig.SearchIfATopicIsInsideAnAlert(vLog.Topics[0]).Events) > 0 { // We matched an alert!
				config := m.globalconfig.SearchIfATopicIsInsideAnAlert(vLog.Topics[0])
				fmt.Printf("-------------------------- Event Detected ------------------------\n")
				fmt.Printf("TxHash: h%s\nAddress:%s\nTopics: %s\n", vLog.TxHash, vLog.Address, vLog.Topics)
				fmt.Printf("The current config that matched this function: %v\n", config)
				fmt.Printf("----------------------------------------------------------------\n")
				m.eventEmitted.WithLabelValues(m.nickname, config.Name, config.Priority, config.Events[0].Signature, vLog.Address.String()).Set(float64(1))

			}
		}

	}
	m.log.Info("Checking events...", "CurrentBlock", latestBlockNumber)

}

func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	return nil
}
