package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type TxBase interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func InTx(ctx context.Context, base TxBase, op func(*sql.Tx) error) (err error) {
	tx, err := base.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("could not start transaction: %v", err)
	}

	defer func() {
		if e := tx.Rollback(); e != nil && !errors.Is(e, sql.ErrTxDone) {
			err = fmt.Errorf("rollback error: %v", e)
			return
		}
	}()

	if err := op(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %v", err)
	}

	return
}

func IterRows(rows *sql.Rows, op func(rows *sql.Rows) error) (err error) {
	defer func() {
		if e := rows.Close(); e != nil {
			err = fmt.Errorf("error on closing rows: %v", e)
			return
		}
	}()
	for rows.Next() {
		if err := op(rows); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error on reading rows: %v", err)
	}
	return
}

func InsertMany(ctx context.Context, db *sql.DB, values []string) (int, error) {
	var q strings.Builder
	vals := make([]interface{}, len(values))

	q.WriteString(`INSERT INTO videos (video_id) VALUES `)

	last := len(values) - 1
	for i, v := range values {
		q.WriteString(`(?)`)
		if i < last {
			q.WriteString(",")
		}
		vals[i] = v
	}
	q.WriteString(" ON CONFLICT (video_id) DO NOTHING;")

	n := 0
	err := InTx(ctx, db, func(tx *sql.Tx) error {
		r, err := tx.Exec(q.String(), vals...)
		if err != nil {
			return err
		}
		n = must(r.RowsAffected())
		return err
	})

	if err != nil {
		return 0, fmt.Errorf("transaction error: %v", err)
	}
	return n, nil
}

func must(v int64, err error) int {
	if err != nil {
		panic(err)
	}
	return int(v)
}
