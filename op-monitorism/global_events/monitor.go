package global_events

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	"regexp"
	"strings"
	"time"
)

const (
	MetricsNamespace = "global_events_mon"
)

var counter int = 0

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
	CurrentBlock        *prometheus.GaugeVec
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

// NewMonitor creates a new Monitor instance.
func NewMonitor(ctx context.Context, log log.Logger, m metrics.Factory, cfg CLIConfig) (*Monitor, error) {
	l1Client, err := ethclient.Dial(cfg.L1NodeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial l1 rpc: %w", err)
	}
	log.Info("--------------------------------------- Global_events_mon (Infos) -----------------------------\n")
	ChainID, err := l1Client.ChainID(context.Background())
	if err != nil {
		log.Crit("Failed to retrieve chain ID: %v", err)
	}
	header, err := l1Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Crit("Failed to fetch the latest block header", "error", err)
	}
	// display the infos at the start to ensure everything is correct.
	log.Info("", "latestBlockNumber", header.Number)
	log.Info("", "chainId", ChainIDToName(ChainID.Int64()))
	log.Info("", "PathYaml", cfg.PathYamlRules)
	log.Info("", "Nickname", cfg.Nickname)
	log.Info("", "L1NodeURL", cfg.L1NodeURL)
	globalConfig, err := ReadAllYamlRules(cfg.PathYamlRules, log)
	if err != nil {
		log.Crit("Failed to read the yaml rules", "error", err.Error())
	}

	globalConfig.DisplayMonitorAddresses(log) //Display all the addresses that are monitored.
	log.Info("--------------------------------------- End of Infos -----------------------------\n")
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
		}, []string{"nickname", "rulename", "priority", "functionName", "topics", "address", "blockNumber", "txHash"}),
		unexpectedRpcErrors: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "unexpectedRpcErrors",
			Help:      "number of unexpcted rpc errors",
		}, []string{"section", "name"}),
		CurrentBlock: m.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: MetricsNamespace,
			Name:      "CurrentBlock",
			Help:      "This metric return the current blockNumber Monitored.",
		}, []string{"nickname"}),
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

// Run the monitor functions declared as a monitor method.
func (m *Monitor) Run(ctx context.Context) {
	m.checkEvents(ctx)
}

// metricsAllEventsRegistered allows to emit all the events at the start of the program with the values set to `0`.
func metricsAllEventsRegistered(globalconfig GlobalConfiguration, eventEmitted *prometheus.GaugeVec, nickname string) {
	for _, config := range globalconfig.Configuration {
		if len(config.Addresses) == 0 {
			for _, event := range config.Events {
				eventEmitted.WithLabelValues(nickname, config.Name, config.Priority, event.Signature, event.Keccak256_Signature.Hex(), "ANY_ADDRESSES_[]", "0", "N/A").Set(float64(0))
			}
			continue //pass to the next config so the [] any are not displayed as metrics here.
		}
		for _, address := range config.Addresses {
			for _, event := range globalconfig.ReturnEventsMonitoredForAnAddressFromAConfig(address, config) {
				eventEmitted.WithLabelValues(nickname, config.Name, config.Priority, event.Signature, event.Keccak256_Signature.Hex(), address.String(), "0", "N/A").Set(float64(0))
			}
		}
	}

}

// checkEvents function to check the events. If an events is emitted onchain and match the rules defined in the yaml file, then we will display the event.
func (m *Monitor) checkEvents(ctx context.Context) { //TODO: Ensure the logs crit are not causing panic in runtime!

	if counter == 0 { //meaning we are at the start of the program.
		metricsAllEventsRegistered(m.globalconfig, m.eventEmitted, m.nickname) // Emit all the events
	}

	counter++
	header, err := m.l1Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		m.unexpectedRpcErrors.WithLabelValues("L1", "HeaderByNumber").Inc()
		m.log.Warn("Failed to retrieve latest block header", "error", err.Error()) //TODO:need to wait 12 and retry here!
		return
	}
	latestBlockNumber := header.Number
	blocknumber, _ := latestBlockNumber.Float64()

	m.CurrentBlock.WithLabelValues(m.nickname).Set(float64(blocknumber)) //metrics for the current block monitored.
	query := ethereum.FilterQuery{
		FromBlock: latestBlockNumber,
		ToBlock:   latestBlockNumber,
		// Addresses: []common.Address{}, //if empty means that all addresses are monitored should be this value for optimisation and avoiding to take every logs every time -> m.globalconfig.GetUniqueMonitoredAddresses
	}

	logs, err := m.l1Client.FilterLogs(context.Background(), query)
	if err != nil { //TODO:need to wait 12 and retry here!
		m.unexpectedRpcErrors.WithLabelValues("L1", "FilterLogs").Inc()
		m.log.Warn("Failed to retrieve logs:", "error", err.Error())
		return
	}

	for _, vLog := range logs {
		if len(vLog.Topics) > 0 { // Ensure no anonymous event is here.
			configs := m.globalconfig.ReturnConfigsFromTopic(vLog.Topics[0])
			if len(configs) > 0 {
				config := ReturnConfigFromConfigsAndAddress(vLog.Address, configs)
				if len(config.Events) == 0 {
					continue
				}
				// We matched an alert!
				event_config := ReturnAndEventForAnTopic(vLog.Topics[0], config)
				m.log.Info("Event Detected", "TxHash", vLog.TxHash.String(), "Address", vLog.Address, "Topics", vLog.Topics, "Config", config, "event_config.Signature", event_config.Signature, "event_config.Keccak256_Signature", event_config.Keccak256_Signature.Hex())
				m.eventEmitted.WithLabelValues(m.nickname, config.Name, config.Priority, event_config.Signature, event_config.Keccak256_Signature.Hex(), vLog.Address.String(), latestBlockNumber.String(), vLog.TxHash.String()).Set(float64(1))
			}
		}
	}
	m.log.Info("Checking events..", "CurrentBlock", latestBlockNumber)
}

// ReturnConfigFromConfigsAndAddress allows to return the config from the configs and the address.
func ReturnConfigFromConfigsAndAddress(address common.Address, configs []Configuration) Configuration {
	configDefault := Configuration{}
	for _, config := range configs {
		if len(config.Addresses) == 0 { //return true to listen to every addresses.
			configDefault = config
			continue
		}
		for _, addr := range config.Addresses { // iterate over all the addresses in the config.
			if addr == address {
				return config
			}
		}
	}
	return configDefault
}

// ReturnEventsMonitoredForAnAddressFromAConfig return a full `Event` struct for a topic and a config.
func ReturnAndEventForAnTopic(topic common.Hash, config Configuration) Event {
	for _, event := range config.Events {
		if topic == event.Keccak256_Signature {
			return event
		}
	}
	return Event{}
}

// Close closes the monitor.
func (m *Monitor) Close(_ context.Context) error {
	m.l1Client.Close()
	return nil
}
