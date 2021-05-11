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

	AudioBitrate      int    `yaml:"audioBitrate"`
	VideoQualityLevel int    `yaml:"videoQualityLevel"`
	CPUUsage          string `yaml:"cpuUsage"`

	VideoFramerate int `yaml:"framerate"`

	StreamingURL string `yaml:"streamingURL"`
	StreamingKey string `yaml:"streamingKey"`

	OwncastServerURL   string `yaml:"owncastServerURL"`
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
