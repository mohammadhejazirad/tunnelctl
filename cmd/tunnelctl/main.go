package main

import (
	"os"
	"tunnelctl/internal/cli"
)

func main() { os.Exit(cli.Execute(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
