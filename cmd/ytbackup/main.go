package main

import (
	"log"
	"os"

	"github.com/jessevdk/go-flags"
)

type Commander interface {
	Init(opts interface{}) error
	Execute(args []string) error
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	var opts Options
	parser := flags.NewParser(&opts, flags.Default)

	parser.CommandHandler = func(command flags.Commander, args []string) error {
		c := command.(Commander)
		if err := c.Init(opts.Common); err != nil {
			return err
		}
		if err := c.Execute(args); err != nil {
			return err
		}
		return nil
	}

	if _, err := parser.Parse(); err != nil {
		if e, ok := err.(*flags.Error); ok {
			if e.Type == flags.ErrHelp {
				os.Exit(0)
			}
		}
		os.Exit(1)
	}
}
