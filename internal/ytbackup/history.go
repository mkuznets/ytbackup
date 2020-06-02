package ytbackup

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	ytbrowser "mkuznets.com/go/ytbackup/internal/browser"
	"mkuznets.com/go/ytbackup/internal/database"
	"mkuznets.com/go/ytbackup/internal/history"
)

type HistoryCommand struct {
	Command
}

func (cmd *HistoryCommand) Execute(args []string) error {
	bcfg := cmd.Config.Browser

	bro, err := ytbrowser.New(bcfg.Executable, bcfg.DataDir, bcfg.DebugPort, bcfg.ExtraArgs)
	if err != nil {
		return err
	}

	ctx := context.Background()

	var videos []*history.Video
	err = bro.Do(ctx, func(ctx context.Context, url string) error {
		var e error
		videos, e = history.Videos(ctx, url)
		if e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := insertMany(ctx, cmd.DB, videos[:5]); err != nil {
		return err
	}

	return nil
}

func insertMany(ctx context.Context, db *sql.DB, values []*history.Video) error {
	var q strings.Builder
	vals := make([]interface{}, len(values))

	q.WriteString(`INSERT INTO videos (video_id) VALUES `)

	last := len(values) - 1
	for i, v := range values {
		q.WriteString(`(?)`)
		if i < last {
			q.WriteString(",")
		}
		vals[i] = v.ID
	}
	q.WriteString(" ON CONFLICT (video_id) DO NOTHING;")

	err := database.InTx(ctx, db, func(tx *sql.Tx) error {
		_, err := tx.Exec(q.String(), vals...)
		return err
	})

	if err != nil {
		return fmt.Errorf("transaction error: %v", err)
	}
	return nil
}
