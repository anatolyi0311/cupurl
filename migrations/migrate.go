package migrations

import (
	"database/sql"
	"fmt"

	"go.uber.org/zap"
)

func Up(db *sql.DB, sugar zap.SugaredLogger) error {
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
	sugar.Info("DB.Up.3 ", db == nil)

	return nil
}

func Down(db *sql.DB, sugar zap.SugaredLogger) error {
	if db == nil {
		return fmt.Errorf("db not init")
	}

	tx, err := db.Begin()
	if err != nil {
		sugar.Info("DB.Down.0 ", db == nil)
		return err
	}

	if _, err := tx.Exec(dropTable); err != nil {
		tx.Rollback()
		sugar.Info("DB.Down.1 ", db == nil)
		return err
	}

	if err = tx.Commit(); err != nil {
		sugar.Info("DB.Down.2 ", db == nil)
		tx.Rollback()
		return err
	}
	sugar.Info("DB.Down.3 ", db == nil)

	return nil
}
