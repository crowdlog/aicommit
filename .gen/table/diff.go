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

var Diff = newDiffTable("", "diff", "")

type diffTable struct {
	sqlite.Table

	// Columns
	ID                 sqlite.ColumnString
	Diff               sqlite.ColumnString
	DateCreated        sqlite.ColumnTimestamp
	DiffStructuredJSON sqlite.ColumnString
	Model              sqlite.ColumnString
	AiProvider         sqlite.ColumnString
	Prompts            sqlite.ColumnString

	AllColumns     sqlite.ColumnList
	MutableColumns sqlite.ColumnList
}

type DiffTable struct {
	diffTable

	EXCLUDED diffTable
}

// AS creates new DiffTable with assigned alias
func (a DiffTable) AS(alias string) *DiffTable {
	return newDiffTable(a.SchemaName(), a.TableName(), alias)
}

// Schema creates new DiffTable with assigned schema name
func (a DiffTable) FromSchema(schemaName string) *DiffTable {
	return newDiffTable(schemaName, a.TableName(), a.Alias())
}

// WithPrefix creates new DiffTable with assigned table prefix
func (a DiffTable) WithPrefix(prefix string) *DiffTable {
	return newDiffTable(a.SchemaName(), prefix+a.TableName(), a.TableName())
}

// WithSuffix creates new DiffTable with assigned table suffix
func (a DiffTable) WithSuffix(suffix string) *DiffTable {
	return newDiffTable(a.SchemaName(), a.TableName()+suffix, a.TableName())
}

func newDiffTable(schemaName, tableName, alias string) *DiffTable {
	return &DiffTable{
		diffTable: newDiffTableImpl(schemaName, tableName, alias),
		EXCLUDED:  newDiffTableImpl("", "excluded", ""),
	}
}

func newDiffTableImpl(schemaName, tableName, alias string) diffTable {
	var (
		IDColumn                 = sqlite.StringColumn("id")
		DiffColumn               = sqlite.StringColumn("diff")
		DateCreatedColumn        = sqlite.TimestampColumn("date_created")
		DiffStructuredJSONColumn = sqlite.StringColumn("diff_structured_json")
		ModelColumn              = sqlite.StringColumn("model")
		AiProviderColumn         = sqlite.StringColumn("ai_provider")
		PromptsColumn            = sqlite.StringColumn("prompts")
		allColumns               = sqlite.ColumnList{IDColumn, DiffColumn, DateCreatedColumn, DiffStructuredJSONColumn, ModelColumn, AiProviderColumn, PromptsColumn}
		mutableColumns           = sqlite.ColumnList{DiffColumn, DateCreatedColumn, DiffStructuredJSONColumn, ModelColumn, AiProviderColumn, PromptsColumn}
	)

	return diffTable{
		Table: sqlite.NewTable(schemaName, tableName, alias, allColumns...),

		//Columns
		ID:                 IDColumn,
		Diff:               DiffColumn,
		DateCreated:        DateCreatedColumn,
		DiffStructuredJSON: DiffStructuredJSONColumn,
		Model:              ModelColumn,
		AiProvider:         AiProviderColumn,
		Prompts:            PromptsColumn,

		AllColumns:     allColumns,
		MutableColumns: mutableColumns,
	}
}
