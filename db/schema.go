// schema.go — Full PascalCase schema creation and views.
package db

import (
	"github.com/alimtvnetwork/movie-cli-v4/apperror"

	"github.com/alimtvnetwork/movie-cli-v4/version"
)

// migrateSchema creates all tables, indexes, views, and seed data.
func (d *DB) migrateSchema() error {
	if err := d.createTables(); err != nil {
		return apperror.Wrap("create tables", err)
	}
	if err := d.seedFileActions(); err != nil {
		return apperror.Wrap("seed FileAction", err)
	}
	if err := d.seedDefaultConfig(); err != nil {
		return apperror.Wrap("seed Config", err)
	}
	if err := d.createViews(); err != nil {
		return apperror.Wrap("create views", err)
	}
	if err := d.SetConfig("AppVersion", version.Short()); err != nil {
		return apperror.Wrap("stamp app version", err)
	}
	return nil
}

// createTables creates all PascalCase tables.
func (d *DB) createTables() error {
	if err := d.createLookupTables(); err != nil {
		return err
	}
	if err := d.createCoreTables(); err != nil {
		return err
	}
	if err := d.createJoinTables(); err != nil {
		return err
	}
	if err := d.createHistoryTables(); err != nil {
		return err
	}
	if err := d.createSystemTables(); err != nil {
		return err
	}
	return d.createIndexes()
}
