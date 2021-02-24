package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

const (
	configFile = "config.yaml"
)

type Config struct {
	AudioURL string `yaml:"audioUrl"`

	StreamingURL string `yaml:"streamingURL"`
	StreamingKey string `yaml:"streamingKey"`

	OwncastAccessToken string `yaml:"owncastAccessToken"`
}

func loadConfig() Config {
	var config Config

	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println("config file err", err)
		return config
	}

	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		fmt.Println("Error reading the config file.", err)
		return config
	}

	return config
}
