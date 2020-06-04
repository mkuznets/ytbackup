package ytbackup

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"

	"github.com/mitchellh/go-homedir"
	"mkuznets.com/go/ytbackup/internal/database"
	"mkuznets.com/go/ytbackup/internal/utils"
)

type ImportCommand struct {
	Args struct {
		File string `positional-arg-name:"FILE"`
	} `positional-args:"1" required:"1"`
	Command
}

func (cmd *ImportCommand) Execute([]string) error {
	path, err := homedir.Expand(cmd.Args.File)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file: %v", err)
	}

	ids := make([]string, 0, 1024)

	// Direct IDs from likes.json and other playlist takeouts
	err = utils.ExtractByKey(data, "videoId", func(id string) {
		ids = append(ids, id)
	})
	if err != nil {
		return fmt.Errorf("import error: %v", err)
	}

	// URLs from watch-history.json
	err = utils.ExtractByKey(data, "titleUrl", func(u string) {
		up, err := url.Parse(u)
		if err != nil {
			return
		}
		id := up.Query().Get("v")
		if id != "" {
			ids = append(ids, id)
		}
	})
	if err != nil {
		return fmt.Errorf("import error: %v", err)
	}

	n, err := database.InsertMany(context.Background(), cmd.DB, ids)
	if err != nil {
		return err
	}

	log.Printf("Imported: %d", len(ids))
	log.Printf("Added with status NEW: %d", n)

	return nil
}
