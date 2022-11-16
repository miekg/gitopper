package main

import (
	"fmt"
	"log"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	Global   Machine   // although not _all_ fields
	Machines []Machine `toml:"machines"`
}

func main() {
	doc, err := os.ReadFile("config")
	if err != nil {
		log.Fatal(err)
	}

	var cfg Config
	if err := toml.Unmarshal([]byte(doc), &cfg); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", cfg)
}
