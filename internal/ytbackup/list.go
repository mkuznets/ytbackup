package ytbackup

import (
	"os"

	"mkuznets.com/go/ytbackup/internal/format"
	"mkuznets.com/go/ytbackup/internal/index"
)

type ListCommand struct {
	Status string `short:"s" long:"status" description:"Filter videos by status. Valid options: NEW, ENQUEUED, DONE, INPROGRESS, FAILED, SKIPPED."`
	JSON   bool   `long:"json" description:"JSON output"`
	Command
}

func (cmd *ListCommand) Execute([]string) error {
	status := index.Status(cmd.Status)

	var f format.Formatter
	if cmd.JSON {
		f = format.NewJSON(os.Stdout)
	} else {
		f = format.NewTable(os.Stdout)
	}

	if err := cmd.Index.Iter(status, f.Put); err != nil {
		return err
	}

	if err := f.Flush(); err != nil {
		return err
	}

	return nil
}
