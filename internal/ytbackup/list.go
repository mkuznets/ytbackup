package ytbackup

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"mkuznets.com/go/ytbackup/internal/index"
)

type ListCommand struct {
	Status string `short:"s" long:"status" choice:"DONE" choice:"NEW" choice:"FAILED" choice:"SKIPPED" choice:"INPROGRESS" description:""` // nolint
	JSON   bool   `long:"json" description:""`
	Command
}

func (cmd *ListCommand) Execute([]string) error {
	status := index.Status(cmd.Status)

	if !cmd.JSON {
		tw := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)

		err := cmd.Index.Map(status, func(video *index.Video) error {
			if _, err := fmt.Fprintln(tw, video.Row()); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
		if err := tw.Flush(); err != nil {
			return err
		}

		return nil
	}

	items := make([]*index.Video, 0)
	err := cmd.Index.Map(status, func(video *index.Video) error {
		items = append(items, video)
		return nil
	})
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(items); err != nil {
		return err
	}

	return nil
}
