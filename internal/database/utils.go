package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
