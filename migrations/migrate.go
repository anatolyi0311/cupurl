package migrations

import (
	"database/sql"
	"fmt"
)

func Up(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db not init")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(createTable); err != nil {
		tx.Rollback()
		return err
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func Down(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("db not init")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(dropTable); err != nil {
		tx.Rollback()
		return err
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		return err
	}

	return nil
}
