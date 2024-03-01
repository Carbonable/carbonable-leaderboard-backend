package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

var (
	ErrParseConfig     = errors.New("unable to parse config")
	ErrUnmarshalFailed = errors.New("unable to unmarshal config")
)

type Contract struct {
	Events  map[string]string `yaml:"events"`
	Address string            `yaml:"address"`
	Name    string            `yaml:"name"`
}

type Config struct {
	Contracts  []Contract `yaml:"contracts"`
	StartBlock uint64     `yaml:"start_block"`
}

func (c *Config) GetContract(address string) *Contract {
	for _, contract := range c.Contracts {
		if contract.Address == address {
			return &contract
		}
	}
	return nil
}

func FromYamlFile(file string) (*Config, error) {
	cfg := Config{}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, ErrParseConfig
	}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, ErrUnmarshalFailed
	}

	return &cfg, nil
}
