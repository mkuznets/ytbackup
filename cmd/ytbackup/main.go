package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Commander interface {
	Init(opts interface{}) error
	Execute(args []string) error
	Close()
}

func main() {
	_ = godotenv.Load()

	var opts Options
	parser := flags.NewParser(&opts, flags.Default)

	parser.CommandHandler = func(command flags.Commander, args []string) error {
		c := command.(Commander)

		if err := c.Init(opts.Common); err != nil {
			log.Fatal().Msg(err.Error())
			return nil
		}
		defer c.Close()

		if err := c.Execute(args); err != nil {
			log.Fatal().Msg(err.Error())
			return nil
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
