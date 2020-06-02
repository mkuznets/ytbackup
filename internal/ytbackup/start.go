package ytbackup

import (
	"context"

	"mkuznets.com/go/ytbackup/internal/downloader"
	"mkuznets.com/go/ytbackup/internal/volumes"
)

type StartCommand struct {
	Command
}

func (cmd *StartCommand) Execute([]string) error {
	ctx := context.Background()

	vs, err := volumes.New(cmd.Config.Volumes)
	if err != nil {
		return err
	}

	dl, err := downloader.New(vs)
	if err != nil {
		return err
	}

	if err := dl.Serve(ctx, cmd.DB); err != nil {
		return err
	}

	return nil
}
