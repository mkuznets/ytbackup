package ytbackup

type StartCommand struct {
	Command
}

func (cmd *StartCommand) Execute(args []string) error {
	return nil
}
