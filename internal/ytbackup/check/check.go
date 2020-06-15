package check

type Command struct {
	Files *FilesCommand `command:"files" description:""`
	Index *IndexCommand `command:"index" description:""`
}
