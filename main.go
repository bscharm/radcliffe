package main

import (
	"flag"

	"github.com/bscharm/radcliffe/parser"
)

func main() {
	filename := flag.String("file", "", "Filename to parse")
	flag.Parse()

	parser.Parse(*filename)
}
