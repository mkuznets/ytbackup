package ytbackup

import (
	"fmt"

	"github.com/rs/zerolog/log"
)

type AddCommand struct {
	Command
	Force bool `short:"f" long:"force" description:"Force adding existing videos (e.g. to retry failed downloads)"`
	Args  struct {
		IDs []string `positional-arg-name:"ID"`
	} `positional-args:"1"`
}

const videoIDLength = 11

func (cmd *AddCommand) Execute([]string) error {
	if len(cmd.Args.IDs) == 0 {
		return nil
	}

	invalid := make([]string, 0)
	for _, id := range cmd.Args.IDs {
		if len(id) != videoIDLength {
			invalid = append(invalid, id)
		}
	}
	if len(invalid) > 0 {
		return fmt.Errorf("invalid video IDs: %v", invalid)
	}

	err := cmd.Index.PutByID(cmd.Force, cmd.Args.IDs...)
	if err != nil {
		return err
	}
	log.Info().Int("count", len(cmd.Args.IDs)).Msg("Videos added")

	return nil
}
