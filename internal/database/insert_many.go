package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

const MaxVarNumber = 256

func InsertMany(ctx context.Context, db *sql.DB, values []string) (int, error) {
	vals := make([]interface{}, len(values))
	for i, v := range values {
		vals[i] = v
	}

	total := 0

	err := InTx(ctx, db, func(tx *sql.Tx) error {
		n := len(vals)

		for i := 0; i < n; i += MaxVarNumber {
			varN := min(MaxVarNumber, n-i)
			q := makeInsertQuery(varN)

			r, err := tx.Exec(q, vals[i:i+varN]...)
			if err != nil {
				return err
			}
			total += must(r.RowsAffected())
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("transaction error: %v", err)
	}
	return total, nil
}

func must(v int64, err error) int {
	if err != nil {
		panic(err)
	}
	return int(v)
}

func makeInsertQuery(n int) string {
	var q strings.Builder

	q.WriteString(`INSERT INTO videos (video_id) VALUES `)
	for i := 0; i < n; i++ {
		q.WriteString(`(?)`)
		if i < n-1 {
			q.WriteString(",")
		}
	}
	q.WriteString(" ON CONFLICT (video_id) DO NOTHING;")

	return q.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
