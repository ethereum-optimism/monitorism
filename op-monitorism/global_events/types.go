package global_events

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

// EventTopic is the struct that will contain the index of the topic and the values that will be monitored (not used currently).
type EventTopic struct {
	Index  int      `yaml:"index"`
	Values []string `yaml:"values"`
}

// Event is the struct that will contain the signature of the event and the topics that will be monitored.
type Event struct {
	Keccak256_Signature common.Hash  // the value is the `Topic[0]`. This is generated from the `Event.Signature` field (eg. 0x23428b18acfb3ea64b08dc0c1d296ea9c09702c09083ca5272e64d115b687d23 --> ExecutionFailure(bytes32,uint256)
	Signature           string       `yaml:"signature"`        // That is the name of the function like "Transfer(address,address,uint256)"
	Topics              []EventTopic `yaml:"topics,omitempty"` // The topics that will be monitored not used yet.
}

// Configuration is the struct that will contain the configuration coming from the yaml files under the `rules` directory.
type Configuration struct {
	Version   string           `yaml:"version"`
	Name      string           `yaml:"name"`
	Priority  string           `yaml:"priority"`
	Addresses []common.Address `yaml:"addresses"` //TODO: add the superchain registry with the format `/l1/l2/optimismPortal`
	Events    []Event          `yaml:"events"`
}

// GlobalConfiguration is the struct that will contain all the configuration of the monitoring.
type GlobalConfiguration struct {
	Configuration []Configuration `yaml:"configuration"`
}

// ReturnEventsMonitoredForAnAddress will return the list of events monitored for a given address /!\ This will return the first occurence of the address in the configuration.
// We assume currently there is no duplicates into the rules.
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

// ReturnEventsMonitoredForAnAddressFromAConfig will return the list of events monitored for a given address from a given configuration. /!\ This will return the first occurence of the address in the configuration. As we assume currently there is no duplicates into the rules.
func (G GlobalConfiguration) ReturnEventsMonitoredForAnAddressFromAConfig(target_address common.Address, config Configuration) []Event {
	for _, address := range config.Addresses {
		if address == target_address {
			return config.Events
		}
	}
	return []Event{} // no events monitored for this address

}
func (G GlobalConfiguration) ReturnConfigsFromTopic(topic common.Hash) []Configuration {
	configs := []Configuration{}
	for _, config := range G.Configuration {
		for _, event := range config.Events {
			if topic == event.Keccak256_Signature {
				configs = append(configs, config)
			}
		}
	}
	return configs
}

// ReadYamlFile read a yaml file and return a Configuration struct.
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

// StringFunctionToHex take the configuration yaml and resolve a solidity event like "Transfer(address)" to the keccak256 hash of the event signature and UPDATE the configuration with the keccak256 hash.
func StringFunctionToHex(config Configuration, log log.Logger) Configuration {
	var FinalConfig Configuration
	if len(config.Addresses) == 0 && len(config.Events) > 0 {
		log.Warn("No addresses to monitor, but some events are defined (this means we are monitoring all the addresses), probably for debugging purposes.")
		keccak256_topic_0 := config.Events
		for i, event := range config.Events {
			keccak256_topic_0[i].Keccak256_Signature = FormatAndHash(event.Signature)
			log.Info("", "Keccak256", keccak256_topic_0[i].Keccak256_Signature)
		}
		FinalConfig = Configuration{Version: config.Version, Name: config.Name, Priority: config.Priority, Addresses: []common.Address{}, Events: keccak256_topic_0}
		return FinalConfig
	}
	// If there is addresses to monitor, we will resolve the signature of the events.
	for _ = range config.Addresses { //resolve the hex signature from a topic
		keccak256_topic_0 := config.Events
		for i, event := range config.Events {
			keccak256_topic_0[i].Keccak256_Signature = FormatAndHash(event.Signature)

		}
		FinalConfig = Configuration{Version: config.Version, Name: config.Name, Priority: config.Priority, Addresses: config.Addresses, Events: keccak256_topic_0}
	}

	return FinalConfig
}

// ReadAllYamlRules Read all the files in the `rules` directory at the given path from the command line `--PathYamlRules` that are YAML files.
func ReadAllYamlRules(PathYamlRules string, log log.Logger) (GlobalConfiguration, error) {
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
	if len(yamlFiles) == 0 {
		return GlobalConfiguration{}, errors.New("No YAML files found in the directory")
	}
	for _, file := range yamlFiles {
		path_rule := PathYamlRules + "/" + file.Name()
		log.Info("Reading a new rule", "Rule", path_rule)
		yamlconfig := ReadYamlFile(path_rule)             // Read the yaml file
		yamlconfig = StringFunctionToHex(yamlconfig, log) // Modify the yaml config to have the common.hash of the event signature.
		GlobalConfig.Configuration = append(GlobalConfig.Configuration, yamlconfig)
		// monitoringAddresses = append(monitoringAddresses, fromConfigurationToAddress(yamlconfig)...)

	}

	yaml_marshalled, err := yaml.Marshal(GlobalConfig)
	err = os.WriteFile("/tmp/globalconfig.yaml", yaml_marshalled, 0644) // Storing the configuration if we need to debug and knows what is monitored in the future.
	if err != nil {
		log.Warn("Error writing the globalconfig YAML file on the disk:", "ERROR", err)
		panic("Error writing the globalconfig YAML file on the disk")
	}
	return GlobalConfig, nil
}

// DisplayMonitorAddresses will display the addresses that are monitored and the events that are monitored for each address.
func (G GlobalConfiguration) DisplayMonitorAddresses(log log.Logger) {
	log.Info("============== Monitoring addresses =================")

	for _, config := range G.Configuration {
		log.Info("", "Name:", config.Name)
		if len(config.Addresses) == 0 && len(config.Events) > 0 {
			log.Warn("Address:[], No address are defined but some events are defined (this means we are monitoring all the addresses), probably for debugging purposes.")
			for _, events := range config.Events {
				log.Info("", "Events", events)
			}
		} else {
			for _, address := range config.Addresses {
				log.Info("   ", " Address", address)
				for _, events := range G.ReturnEventsMonitoredForAnAddressFromAConfig(address, config) {
					log.Info("", "    Events", events)
				}
			}
		}
	}
}
