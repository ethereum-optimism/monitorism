package global_events

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	// "slices"
)

type EventTopic struct {
	Index  int      `yaml:"index"`
	Values []string `yaml:"values"`
}

type Event struct {
	_4bytes   string // That is the 4 bytes of the event signature, will be generated from the Event.Signature function just below this.
	Signature string `yaml:"signature"` // That is the name of the function like "Transfer(address,address,uint256)"
	// Topics    []EventTopic `yaml:"topics,omitempty"`
}

type Configuration struct {
	Version   string           `yaml:"version"`
	Name      string           `yaml:"name"`
	Priority  string           `yaml:"priority"`
	Addresses []common.Address `yaml:"addresses"` //TODO: add the superchain registry with the format `/l1/l2/optimismPortal`
	Events    []Event          `yaml:"events"`
}

type MonitoringAddress struct {
	Address common.Address `yaml:"addresses"`
	Events  []Event        `yaml:"events"`
}

type TabMonitoringAddress struct {
	MonitoringAddress []MonitoringAddress
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

// fromConfigurationToAddress take the configuration yaml and create the `monitoringAddresses` array
// that will contains all the addresses that will monitore.
func fromConfigurationToAddress(config Configuration) []MonitoringAddress {
	var monitoringAddresses []MonitoringAddress
	if len(config.Addresses) == 0 && len(config.Events) > 0 {
		fmt.Println("No addresses to monitor, but some events are defined (this means we are monitoring all the addresses), probably for debugging purposes.")
		var event_with_4bytes []Event
		for _, event := range config.Events {
			event._4bytes = string(FormatAndHash(event.Signature))
			event_with_4bytes = append(event_with_4bytes, event)

		}
		monitoringAddresses = append(monitoringAddresses, MonitoringAddress{Address: common.Address{}, Events: event_with_4bytes})

		return []MonitoringAddress{MonitoringAddress{Address: common.Address{}, Events: event_with_4bytes}}
	}

	for _, address := range config.Addresses {
		var event_with_4bytes []Event
		for _, event := range config.Events {
			event._4bytes = string(FormatAndHash(event.Signature))
			event_with_4bytes = append(event_with_4bytes, event)

		}
		monitoringAddresses = append(monitoringAddresses, MonitoringAddress{Address: address, Events: event_with_4bytes})
	}

	return monitoringAddresses
}

// Read all the files in the `rules` directory at the given path from the command line `--PathYamlRules` that are YAML files.
func ReadAllYamlRules(PathYamlRules string) TabMonitoringAddress {
	var monitoringAddresses []MonitoringAddress

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
		yamlconfig := ReadYamlFile(path_rule)
		monitoringAddresses = append(monitoringAddresses, fromConfigurationToAddress(yamlconfig)...)

	}

	return TabMonitoringAddress{MonitoringAddress: monitoringAddresses}
}
func DisplayMonitorAddresses(monitoringAddresses []MonitoringAddress) {
	println("============== Monitoring addresses =================")
	for _, address := range monitoringAddresses {
		fmt.Println("Address:", address.Address)
		for _, event := range address.Events {
			fmt.Printf("Event: %s, Topic[0]: %x\n", event.Signature, event._4bytes)
		}
	}
	fmt.Println("===============================")

}