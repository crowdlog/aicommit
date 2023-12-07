package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	dbmodel "aicommit/.gen/model"
	"aicommit/.gen/table"

	. "github.com/go-jet/jet/v2/sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

func initSqlite() (err error) {
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	db, err := initDB("aicommit.db")
	if err != nil {
		return err
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return err
	}

	test(db)

	return nil
}

func initDB(dbFilePath string) (*sql.DB, error) {
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

func test(db *sql.DB) {
	stmt := SELECT(table.Aicommit.AllColumns).FROM(table.Aicommit).WHERE(table.Aicommit.ID.EQ(Int(1)))
	var dest []struct {
		dbmodel.Aicommit
	}
	stmt.Query(db, &dest)
	println(stmt.DebugSql())
	jsonText, _ := json.MarshalIndent(dest, "", "\t")
	println(string(jsonText))

}
