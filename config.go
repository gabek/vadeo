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

	UseArtistImage     bool `yaml:"useArtistImage"`
	UseAudioVisualizer bool `yaml:"useAudioVisualizer"`

	FontFile       string `yaml:"fontFile"`
	ArtistFontSize int    `yaml:"artistFontSize"`
	TrackFontSize  int    `yaml:"trackFontSize"`
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

	if config.FontFile == "" {
		config.FontFile = "FreeSans.ttf"
	}

	if config.ArtistFontSize == 0 {
		config.ArtistFontSize = 40
	}

	if config.TrackFontSize == 0 {
		config.TrackFontSize = 35
	}

	if config.AudioBitrate == 0 {
		config.AudioBitrate = 128
	}

	if config.CPUUsage == "" {
		config.CPUUsage = "faster"
	}

	if config.VideoQualityLevel == 0 {
		config.VideoQualityLevel = 25
	}

	if config.VideoFramerate == 0 {
		config.VideoFramerate = 24
	}

	return config
}
