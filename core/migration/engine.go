package migration

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast"
)

type MigrationEngine struct {
	fromSchema *ast.SchemaAST
	toSchema   *ast.SchemaAST
	statements []string
}

type MigrationStatement struct {
	SQL      string
	Type     string
	Priority int
}

func NewMigrationEngine(fromSchema *ast.SchemaAST, toSchema *ast.SchemaAST) *MigrationEngine {
	return &MigrationEngine{
		fromSchema: fromSchema,
		toSchema:   toSchema,
		statements: []string{},
	}
}

func (me *MigrationEngine) GenerateMigration() (string, error) {
	var statements []MigrationStatement

	statements = append(statements, me.generateExtensions()...)

	enumStatements, err := me.generateEnumMigrations()
	if err != nil {
		return "", fmt.Errorf("failed to generate enum migrations: %v", err)
	}
	statements = append(statements, enumStatements...)

	tableStatements, err := me.generateTableMigrations()
	if err != nil {
		return "", fmt.Errorf("failed to generate table migrations: %v", err)
	}
	statements = append(statements, tableStatements...)

	indexStatements, err := me.generateIndexMigrations()
	if err != nil {
		return "", fmt.Errorf("failed to generate index migrations: %v", err)
	}
	statements = append(statements, indexStatements...)

	constraintStatements, err := me.generateConstraintMigrations()
	if err != nil {
		return "", fmt.Errorf("failed to generate constraint migrations: %v", err)
	}
	statements = append(statements, constraintStatements...)

	sort.Slice(statements, func(i, j int) bool {
		return statements[i].Priority < statements[j].Priority
	})

	var sqlStatements []string
	for _, stmt := range statements {
		if stmt.SQL != "" {
			sqlStatements = append(sqlStatements, stmt.SQL)
		}
	}

	return strings.Join(sqlStatements, ";\n\n") + ";", nil
}