package main

import (
	"flag"

	"github.com/bscharm/radcliffe/radcliffe"
)

func init() {
	flag.StringVar(&radcliffe.Port, "p", "3000", "Default port")
}

func main() {
	flag.Parse()
	radcliffe.Start()
}
