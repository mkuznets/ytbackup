package check

import (
	"mkuznets.com/go/ytbackup/internal/ytbackup"
)

type IndexCommand struct {
	ytbackup.Command
}

func (cmd *IndexCommand) Execute([]string) error {
	return cmd.Index.Check()
}
