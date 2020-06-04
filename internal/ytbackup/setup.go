package ytbackup

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
	yt "mkuznets.com/go/ytbackup/internal/youtube"
)

type SetupCommand struct {
	Command
}

func (cmd *SetupCommand) Execute([]string) error {
	ctx := context.Background()

	config := yt.NewConfig()

	tok, err := yt.NewToken(ctx, config)
	if err != nil {
		return err
	}
	if err := printCreds(tok); err != nil {
		return err
	}
	return nil
}

func printCreds(token *oauth2.Token) error {
	cfg := struct {
		Youtube struct {
			OAuth *OAuth
		}
	}{}
	cfg.Youtube.OAuth = NewCredentials(token)

	m, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	fmt.Print("Add the following to the config file:\n\n")
	fmt.Println(string(m))

	return nil
}
