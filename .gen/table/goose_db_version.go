//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package table

import (
	"github.com/go-jet/jet/v2/sqlite"
)

var GooseDbVersion = newGooseDbVersionTable("", "goose_db_version", "")

type gooseDbVersionTable struct {
	sqlite.Table

	// Columns
	ID        sqlite.ColumnInteger
	VersionID sqlite.ColumnInteger
	IsApplied sqlite.ColumnInteger
	Tstamp    sqlite.ColumnTimestamp

	AllColumns     sqlite.ColumnList
	MutableColumns sqlite.ColumnList
}

type GooseDbVersionTable struct {
	gooseDbVersionTable

	EXCLUDED gooseDbVersionTable
}

// AS creates new GooseDbVersionTable with assigned alias
func (a GooseDbVersionTable) AS(alias string) *GooseDbVersionTable {
	return newGooseDbVersionTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new GooseDbVersionTable with assigned schema name
func (a GooseDbVersionTable) FromSchema(schemaName string) *GooseDbVersionTable {
	return newGooseDbVersionTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new GooseDbVersionTable with assigned table prefix
func (a GooseDbVersionTable) WithPrefix(prefix string) *GooseDbVersionTable {
	return newGooseDbVersionTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new GooseDbVersionTable with assigned table suffix
func (a GooseDbVersionTable) WithSuffix(suffix string) *GooseDbVersionTable {
	return newGooseDbVersionTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newGooseDbVersionTable(schemaName, tableName, alias string) *GooseDbVersionTable {
	return &GooseDbVersionTable{
		gooseDbVersionTable: newGooseDbVersionTableImpl(schemaName, tableName, alias),
		EXCLUDED:            newGooseDbVersionTableImpl("", "excluded", ""),
	}
}

func newGooseDbVersionTableImpl(schemaName, tableName, alias string) gooseDbVersionTable {
	var (
		IDColumn        = sqlite.IntegerColumn("id")
		VersionIDColumn = sqlite.IntegerColumn("version_id")
		IsAppliedColumn = sqlite.IntegerColumn("is_applied")
		TstampColumn    = sqlite.TimestampColumn("tstamp")
		allColumns      = sqlite.ColumnList{IDColumn, VersionIDColumn, IsAppliedColumn, TstampColumn}
		mutableColumns  = sqlite.ColumnList{VersionIDColumn, IsAppliedColumn, TstampColumn}
	)

	return gooseDbVersionTable{
		Table: sqlite.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		ID:        IDColumn,
		VersionID: VersionIDColumn,
		IsApplied: IsAppliedColumn,
		Tstamp:    TstampColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}
