package check

type Command struct {
	Files *FilesCommand `command:"files" description:"Check integrity of downloaded files"`
	Index *IndexCommand `command:"index" description:"Check consistency of index database"`
}
