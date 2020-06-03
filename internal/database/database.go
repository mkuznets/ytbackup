package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3" // sqlite driver for `migrate`
	"github.com/golang-migrate/migrate/v4/source/httpfs"
	_ "github.com/mattn/go-sqlite3" // sqlite driver for `database/sql`
	"github.com/rakyll/statik/fs"
	"mkuznets.com/go/ytbackup/internal/sqlfs"
)

const currentVersion = 1

func New() (*sql.DB, error) {
	dsn := "sqlite3://ytbackup.db?_journal_mode=WAL&_synchronous=normal"

	sqlFS, err := fs.NewWithNamespace(sqlfs.Sql)
	if err != nil {
		return nil, fmt.Errorf("could not open sqlfs: %v", err)
	}
	source, err := httpfs.New(sqlFS, "/")
	if err != nil {
		return nil, fmt.Errorf("could not create migration source: %v", err)
	}

	migrator, err := migrate.NewWithSourceInstance("httpfs", source, dsn)
	if err != nil {
		return nil, fmt.Errorf("could not create migrator: %v", err)
	}
	if err := migrator.Migrate(currentVersion); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("could not apply migrations: %v", err)
	}

	db, err := sql.Open("sqlite3", "file:ytbackup.db")
	if err != nil {
		return nil, fmt.Errorf("could not open database: %v", err)
	}
	db.SetMaxOpenConns(1)

	return db, nil
}
