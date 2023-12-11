package main

import (
	"database/sql"
	"embed"
	"io"
	nativeLog "log"
	"os"
	"time"

	jet "github.com/go-jet/jet/v2/sqlite"
	_ "github.com/mattn/go-sqlite3"
	"github.com/phuslu/log"
	"github.com/pressly/goose/v3"

	dbmodel "aicommit/.gen/model"
	"aicommit/.gen/table"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS
var dbFilePathConst = "aicommit.db"

func getCommitDBFactory(initialize bool) (cDB *CommitDB, error error) {
	if initialize {
		goose.SetBaseFS(embedMigrations)
		if err := goose.SetDialect("sqlite3"); err != nil {
			return nil, err
		}
	}

	db, err := getDB(dbFilePathConst)
	if err != nil {
		return nil, err
	}

	cDB = &CommitDB{
		db:         db,
		dbFilePath: dbFilePathConst,
	}
	if initialize {
		if err := cDB.InitDB(); err != nil {
			return nil, err
		}
	}

	return cDB, nil
}

func getDB(dbFilePath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return nil, err
	}
	return db, nil
}

type CommitDB struct {
	db         *sql.DB
	dbFilePath string
}

func (cDB *CommitDB) _DoesDBExist() bool {
	if _, err := os.Stat(cDB.dbFilePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func (cDB *CommitDB) _InitializeBlankTable() (sql.Result, error) {
	log.Debug().Msg("Initializing blank table")
	sqlSchema := `CREATE TABLE "aicommit" ("id" INTEGER PRIMARY KEY AUTOINCREMENT);`
	return cDB.db.Exec(sqlSchema)
}

func (cDB *CommitDB) _RunGooseMigration() error {
	originalOutput := nativeLog.Writer()
	if log.DefaultLogger.Level != log.DebugLevel {
		nativeLog.SetOutput(io.Discard)
	}
	log.Debug().Msg("Running goose migration")
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	err := goose.Up(cDB.db, "migrations")
	if err != nil {
		return err
	}
	nativeLog.SetOutput(originalOutput)
	return nil

}

func (cDB *CommitDB) InitDB() error {
	var DBExists bool = cDB._DoesDBExist()
	if !DBExists {
		log.Debug().Msg("Initializing new database")
		_, err := cDB._InitializeBlankTable()
		if err != nil {
			return err
		}
	}
	if err := cDB._RunGooseMigration(); err != nil {
		return err
	}

	if !DBExists {
		log.Debug().Msg("Initializing user settings")
		_, err := cDB.InitializeUserSettings()
		if err != nil {
			return err
		}
	}

	return nil
}

func (cDB *CommitDB) InitializeUserSettings() (sql.Result, error) {
	id := "user_settings"
	modelSelection := ""
	excludeFiles := ""
	useConventionalCommits := false
	dateCreated := time.Now()
	initialSettingsStruct := dbmodel.UserSettings{
		ID:                     &id,
		ModelSelection:         &modelSelection,
		ExcludeFiles:           &excludeFiles,
		UseConventionalCommits: &useConventionalCommits,
		DateCreated:            &dateCreated,
	}
	stmt := table.UserSettings.INSERT(
		table.UserSettings.ID,
		table.UserSettings.ModelSelection,
		table.UserSettings.ExcludeFiles,
		table.UserSettings.UseConventionalCommits,
		table.UserSettings.DateCreated,
	).MODEL(initialSettingsStruct)
	return stmt.Exec(cDB.db)
}

func (cDB *CommitDB) GetUserSettings() (dbmodel.UserSettings, error) {
	var userSettings dbmodel.UserSettings
	stmt := table.UserSettings.SELECT(
		table.UserSettings.ID,
		table.UserSettings.ModelSelection,
		table.UserSettings.ExcludeFiles,
		table.UserSettings.UseConventionalCommits,
		table.UserSettings.DateCreated,
	).FROM(table.UserSettings).WHERE(table.UserSettings.ID.EQ(jet.String("user_settings")))
	err := stmt.Query(cDB.db, &userSettings)
	if err != nil {
		return userSettings, err
	}
	return userSettings, nil
}

func (cDB *CommitDB) InsertDiff(diff dbmodel.Diff) (sql.Result, error) {
	diffId := "diff"
	diffStruct := dbmodel.Diff{
		ID:                 &diffId,
		Diff:               diff.Diff,
		DateCreated:        diff.DateCreated,
		DiffStructuredJSON: diff.DiffStructuredJSON,
		Model:              diff.Model,
		AiProvider:         diff.AiProvider,
		Prompts:            diff.Prompts,
	}
	deleteStmt := table.Diff.DELETE().WHERE(table.Diff.ID.EQ(jet.String("diff")))
	_, err := deleteStmt.Exec(cDB.db)
	if err != nil {
		return nil, err
	}
	stmt := table.Diff.INSERT(
		table.Diff.ID,
		table.Diff.Diff,
		table.Diff.DateCreated,
		table.Diff.DiffStructuredJSON,
		table.Diff.Model,
		table.Diff.AiProvider,
		table.Diff.Prompts,
	).MODEL(diffStruct)
	return stmt.Exec(cDB.db)

}

// func test(db *sql.DB) {
// 	CommitMessage := "Initial commit"
// 	GitDiffCommand := "git diff HEAD"
// 	GitDiffCommandOutput := "diff output"
// 	ExcludeFiles := "*.log"
// 	DateCreated := time.Now()
// 	something := dbmodel.Commits{

// 		CommitMessage:        &CommitMessage,
// 		GitDiffCommand:       &GitDiffCommand,
// 		GitDiffCommandOutput: &GitDiffCommandOutput,
// 		ExcludeFiles:         &ExcludeFiles,
// 		DateCreated:          &DateCreated,
// 	}
// 	stmt := table.Commits.INSERT(
// 		table.Commits.CommitMessage,
// 		table.Commits.GitDiffCommand,
// 		table.Commits.GitDiffCommandOutput,
// 		table.Commits.ExcludeFiles,
// 		table.Commits.DateCreated,
// 	).MODEL(something)

// 	resp, err := stmt.Exec(db)
// 	if err != nil {
// 		println(err.Error())
// 		println("Error inserting into table")
// 		return
// 	}
// 	fmt.Printf("%+v\n angelo", resp)
// 	// jsonText, _ := json.MarshalIndent(dest, "", "\t")
// 	// println(string(jsonText))

// }
