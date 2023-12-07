package main

import (
	"database/sql"
	"embed"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func downloadAndInstallSqlite() (err error) {

	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	db, err := createDbIfNotExists("aicommit.db")
	if err != nil {
		return err
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return err
	}
	return nil

}

func createDbIfNotExists(dbFilePath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(dbFilePath); os.IsNotExist(err) {

		sqlSchema := `CREATE TABLE "aicommit" ("id" INTEGER PRIMARY KEY AUTOINCREMENT);`

		if _, err = db.Exec(sqlSchema); err != nil {
			return nil, err
		}
	}
	return db, nil
}
