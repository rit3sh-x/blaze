package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rit3sh-x/blaze/core/ast"
	"github.com/rit3sh-x/blaze/core/constants"
	"github.com/rit3sh-x/blaze/core/migration"
)

type MigrateCommand struct {
	migrationName string
	fromSchema    *ast.SchemaAST
	toSchema      *ast.SchemaAST
}

func NewMigrateCommand(migrationName string, fromSchema *ast.SchemaAST, toSchema *ast.SchemaAST) *MigrateCommand {
	return &MigrateCommand{
		migrationName: migrationName,
		fromSchema:    fromSchema,
		toSchema:      toSchema,
	}
}

func (mc *MigrateCommand) Execute() error {
	timestamp := time.Now().Format("20060102150405")

	migrationFolderName := fmt.Sprintf("%s_%s", timestamp, mc.migrationName)

	migrationPath := filepath.Join(constants.MIGRATION_DIR, migrationFolderName)

	err := os.MkdirAll(migrationPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create migration directory: %v", err)
	}

	engine := migration.NewMigrationEngine(mc.fromSchema, mc.toSchema)
	migrationSQL, err := engine.GenerateMigration()
	if err != nil {
		return fmt.Errorf("failed to generate migration: %v", err)
	}
	
	queryFilePath := filepath.Join(migrationPath, "query.sql")

	err = os.WriteFile(queryFilePath, []byte(migrationSQL), 0644)
	if err != nil {
		return fmt.Errorf("failed to write query.sql file: %v", err)
	}

	fmt.Printf("%sMigration created successfully:%s\n", constants.GREEN, constants.RESET)
	fmt.Printf("  Directory: %s\n", migrationPath)
	fmt.Printf("  SQL File: %s\n", queryFilePath)

	return nil
}

func GenerateMigration(migrationName string, fromSchema *ast.SchemaAST, toSchema *ast.SchemaAST) error {
	cmd := NewMigrateCommand(migrationName, fromSchema, toSchema)
	return cmd.Execute()
}