package global_events

import (
	// "bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
	// "slices"
)

type EventTopic struct {
	Index  int      `yaml:"index"`
	Values []string `yaml:"values"`
}

type Event struct {
	Keccak256_Signature common.Hash  // `Topic[0]` That is the 4 bytes of the event signature, will be generated from the Event.Signature function just below this.
	Signature           string       `yaml:"signature"` // That is the name of the function like "Transfer(address,address,uint256)"
	Topics              []EventTopic `yaml:"topics,omitempty"`
}

type Configuration struct {
	Version   string           `yaml:"version"`
	Name      string           `yaml:"name"`
	Priority  string           `yaml:"priority"`
	Addresses []common.Address `yaml:"addresses"` //TODO: add the superchain registry with the format `/l1/l2/optimismPortal`
	Events    []Event          `yaml:"events"`
}

type GlobalConfiguration struct {
	Configuration []Configuration `yaml:"configuration"`
}

// monitore one address for multiples events
type MonitoringAddress struct {
	Address common.Address `yaml:"addresses"`
	Events  []Event        `yaml:"events"`
}

// tab of monitoring addresses allows us to have multiples addresses with multiples events cf above.
type TabMonitoringAddress struct {
	MonitoringAddress []MonitoringAddress
}

// This will return at the FIRST occurence of the address in the configuration.
// This can be an issue if there is multiples times the same alerts in multiples yaml rules.
// TODO: mark this one into the docs.
func (G GlobalConfiguration) ReturnEventsMonitoredForAnAddress(target_address common.Address) []Event {
	for _, config := range G.Configuration {
		for _, address := range config.Addresses {
			if address == target_address {
				return config.Events
			}
		}
	}
	return []Event{} // no events monitored for this address

}
func (G GlobalConfiguration) SearchIfATopicIsInsideAnAlert(topic common.Hash) Configuration {
	for _, config := range G.Configuration {
		for _, event := range config.Events {
			// fmt.Printf("Comparing %x with %x\n", topic, event.Keccak256_Signature)
			if topic == event.Keccak256_Signature {
				return config
			}

		}
	}
	return Configuration{}

}

// Return all the addresses currently monitored
func (T TabMonitoringAddress) GetMonitoredAddresses() []common.Address {
	var addresses []common.Address
	for _, T := range T.MonitoringAddress {
		addresses = append(addresses, T.Address)
	}

	return addresses
}

// Return all the events currently monitored
func (T TabMonitoringAddress) GetMonitoredEvents() []Event {
	var Events []Event
	for _, T := range T.MonitoringAddress {
		for _, event := range T.Events {
			Events = append(Events, event)
		}
	}

	return Events
}

// return all the UNIQUE addresses currently GetMonitoredEvents
func (T TabMonitoringAddress) GetUniqueMonitoredAddresses() []common.Address {
	hashmap := make(map[common.Address]bool)
	for _, address := range T.GetMonitoredAddresses() {
		if address != common.HexToAddress("0x0") { // If the address is set to 0x0, it means we are monitoring all the addresses, so we need to remove it from the tab here.
			hashmap[address] = true
		}

	}
	return maps.Keys(hashmap)
}

func ReadYamlFile(filename string) Configuration {
	var config Configuration
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading YAML file:", err)
		panic("Error reading YAML")
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("Error reading YAML file:", err)
		panic("Error reading YAML")

	}
	return config
}

// fromConfigurationToAddress take the configuration yaml and resolve the signature stringed like Transfer(address  // that will contains all the addresses that will monitore.
func StringFunctionToHex(config Configuration) Configuration {
	var FinalConfig Configuration

	if len(config.Addresses) == 0 && len(config.Events) > 0 {
		fmt.Println("No addresses to monitor, but some events are defined (this means we are monitoring all the addresses), probably for debugging purposes.")
		keccak256_topic_0 := config.Events
		for i, event := range config.Events {
			keccak256_topic_0[i].Keccak256_Signature = FormatAndHash(event.Signature)
			fmt.Printf("Keccak256_Signature: %x\n", keccak256_topic_0[i].Keccak256_Signature)
		}
		FinalConfig = Configuration{Version: config.Version, Name: config.Name, Priority: config.Priority, Addresses: []common.Address{}, Events: keccak256_topic_0}

		return FinalConfig
	}
	// If there is addresses to monitor, we will resolve the signature of the events.
	for _, address := range config.Addresses { //resolve the hex signature from a topic
		keccak256_topic_0 := config.Events
		for i, event := range config.Events {
			keccak256_topic_0[i].Keccak256_Signature = FormatAndHash(event.Signature)

		}
		FinalConfig = Configuration{Version: config.Version, Name: config.Name, Priority: config.Priority, Addresses: []common.Address{address}, Events: keccak256_topic_0}
	}

	return FinalConfig
}

// Read all the files in the `rules` directory at the given path from the command line `--PathYamlRules` that are YAML files.
func ReadAllYamlRules(PathYamlRules string) GlobalConfiguration {
	var GlobalConfig GlobalConfiguration

	entries, err := os.ReadDir(PathYamlRules) //Only read yaml files
	if err != nil {
		fmt.Println("Error reading directory:", err)
		panic("Error reading directory")
	}
	var yamlFiles []os.DirEntry
	// Filter entries for files ending with ".yaml" or ".yml"
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		// Check if the file ends with ".yaml" or ".yml"
		if filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml" {
			yamlFiles = append(yamlFiles, entry)
		}
	}

	for _, file := range yamlFiles {
		path_rule := PathYamlRules + "/" + file.Name()
		fmt.Printf("Reading a rule named: %s\n", path_rule)
		yamlconfig := ReadYamlFile(path_rule)        // Read the yaml file
		yamlconfig = StringFunctionToHex(yamlconfig) // Modify the yaml config to have the common.hash of the event signature.
		GlobalConfig.Configuration = append(GlobalConfig.Configuration, yamlconfig)
		// monitoringAddresses = append(monitoringAddresses, fromConfigurationToAddress(yamlconfig)...)

	}

	yaml_marshalled, err := yaml.Marshal(GlobalConfig)
	err = os.WriteFile("globalconfig.yaml", yaml_marshalled, 0644)
	if err != nil {
		fmt.Println("Error writing the globalconfig YAML file on the disk:", err)
		panic("Error writing the globalconfig YAML file on the disk")
	}
	return GlobalConfig
}

func (G GlobalConfiguration) DisplayMonitorAddresses() {
	println("============== Monitoring addresses =================")
	for _, config := range G.Configuration {
		fmt.Printf("Name: %s\n", config.Name)
		if len(config.Addresses) == 0 && len(config.Events) > 0 {
			fmt.Println("No addresses to monitor, but some events are defined (this means we are monitoring all the addresses), probably for debugging purposes.")
			for _, events := range config.Events {
				fmt.Printf("Events: %v\n", events)
			}
		} else {
			for _, address := range config.Addresses {
				fmt.Println("Address:", address)
				fmt.Printf("Events: %v\n", G.ReturnEventsMonitoredForAnAddress(address))
			}
		}
	}
}
