package main

import (
	"io/ioutil"

	butt "github.com/jahkeup/buttonoff"
	"github.com/pelletier/go-toml"
)

func LoadConfig(file string) (*butt.Config, error) {
	var config butt.Config
	tree, loadErr := toml.LoadFile(file)
	if loadErr != nil {
		return nil, loadErr
	}

	unmarshalErr := tree.Unmarshal(&config)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return &config, nil
}

func defaultConfig() []byte {
	tomlStr := `
[general]
dropunconfigured = false

[listener]
interface = "eth0"

[mqtt]
brokeraddr = "tcp://127.0.0.1:1883"

[[buttons]]
buttonid = "my-button"
hwaddr = "fc:a6:67:b1:24:41"
`
	return []byte(tomlStr)
}

func writeDefaultConfig(file string) error {
	return ioutil.WriteFile(file, defaultConfig(), 0660)
}
