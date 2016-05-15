package main

import (
	"flag"

	"github.com/bscharm/radcliffe/radcliffe"
)

func init() {
	flag.StringVar(&radcliffe.PORT, "p", "3000", "Default port")
	flag.BoolVar(&radcliffe.DEBUG, "debug", false, "Debug mode")
}

func main() {
	flag.Parse()
	radcliffe.Start()
}
