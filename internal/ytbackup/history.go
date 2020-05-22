package ytbackup

import (
	"context"
	"fmt"

	ytbrowser "mkuznets.com/go/ytbackup/internal/browser"
	"mkuznets.com/go/ytbackup/internal/history"
)

type HistoryCommand struct {
	Command
}

func (cmd *HistoryCommand) Execute(args []string) error {
	if err := cmd.Config.ValidateForHistory(); err != nil {
		return err
	}

	bcfg := cmd.Config.Browser
	bro, err := ytbrowser.New(bcfg.Executable, bcfg.DataDir, bcfg.DebugPort, bcfg.ExtraArgs)
	if err != nil {
		return err
	}

	ctx := context.Background()

	var videos []history.Video
	err = bro.Do(ctx, func(ctx context.Context, url string) error {
		vs, err := history.Videos(ctx, url)
		if err != nil {
			return err
		}
		videos = vs
		return nil
	})
	if err != nil {
		return err
	}

	for _, video := range videos {
		fmt.Println(video.URL)
	}

	return nil
}
