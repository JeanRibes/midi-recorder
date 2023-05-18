package main

import (
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Keymap map[int]struct {
		Channel int
		Value   int
		Hold    bool

		Control bool
		Param   int
	}
}

var config Config

func LoadKeymap(filename string) {

	file, err := os.Open(filename)
	he(err)
	data, err := ioutil.ReadAll(file)
	he(err)
	yaml.Unmarshal(data, &config)
}

func he(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
