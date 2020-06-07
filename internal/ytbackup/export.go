package ytbackup

import (
	"encoding/json"
	"os"

	"mkuznets.com/go/ytbackup/internal/index"
)

type ExportCommand struct {
	Command
}

func (cmd *ExportCommand) Execute([]string) error {
	items := make([]*index.Video, 0)

	err := cmd.Index.ListDone(func(video *index.Video) error {
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
