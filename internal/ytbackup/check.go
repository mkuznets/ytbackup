package ytbackup

type CheckCommand struct {
	Command
}

func (cmd *CheckCommand) Execute([]string) error {
	return cmd.Index.Check()
}
