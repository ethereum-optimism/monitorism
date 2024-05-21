package global_events

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
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
	Version   string   `yaml:"version"`
	Name      string   `yaml:"name"`
	Priority  string   `yaml:"priority"`
	Addresses []string `yaml:"addresses"`
	Events    []Event  `yaml:"events"`
}

type MonitoringAddress struct {
	Address string  `yaml:"addresses"`
	Events  []Event `yaml:"events"`
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
func ReadAllYamlRules(PathYamlRules string) []MonitoringAddress {
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

	return monitoringAddresses
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

// if _, err := toml.DecodeFile("config.toml", &config); err != nil {
//         fmt.Println("Error loading TOML data:", err)
//         os.Exit(1)
//     }
