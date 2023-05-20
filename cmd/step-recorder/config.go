package main

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Controllers map[string]uint8 `yaml:"controllers"`
	Channels    struct {
		Output uint8 `yaml:"recorder_output"`
		Ping   uint8 `yaml:"ping"`
	} `yaml:"channels"`
}

var config Config

func LoadConfig(filename string) {
	file, err := os.Open(filename)
	he(err)
	data, err := ioutil.ReadAll(file)
	he(err)
	yaml.Unmarshal(data, &config)
}
