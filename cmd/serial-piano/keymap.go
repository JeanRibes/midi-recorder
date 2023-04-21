package main

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
)

func LoadKeymap(filename string) map[int]int {
	file, err := os.Open(filename)
	he(err)
	r := bufio.NewReader(file)
	keymap := map[int]int{}
	for {
		line, err := r.ReadString('\n')
		if err != nil || line == "" {
			println(err)
			break
		}
		line = line[0 : len(line)-1]
		s := strings.Split(line, ":")
		if len(s) == 2 {
			key, err := strconv.Atoi(s[0])
			he(err)
			val, err := strconv.Atoi(s[1])
			he(err)
			keymap[key] = val
		}
	}
	return keymap
}

func he(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
