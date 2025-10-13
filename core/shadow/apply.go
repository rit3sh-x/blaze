package shadow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/rit3sh-x/blaze/core/constants"
)

type MigrationFile struct {
	Name      string
	Path      string
	Timestamp int64
	SQL       string
}

type ApplyEngine struct {
	migrationDir string
	parser       *SQLParser
}

func NewApplyEngine() *ApplyEngine {
	return &ApplyEngine{
		migrationDir: constants.MIGRATION_DIR,
		parser:       NewSQLParser(),
	}
}

func (ae *ApplyEngine) BuildProgressiveSchema() (string, string, error) {
	migrationFiles, err := ae.readMigrationFiles()
	if err != nil {
		return "", "", fmt.Errorf("failed to read migration files: %v", err)
	}

	if len(migrationFiles) == 0 {
		return "", "", nil
	}

	sort.Slice(migrationFiles, func(i, j int) bool {
		return migrationFiles[i].Timestamp < migrationFiles[j].Timestamp
	})

	currentEnumsStr := ""
	currentClassesStr := ""

	for _, migrationFile := range migrationFiles {
		newSchema, err := ae.parser.ApplyMigrationToSchema(currentEnumsStr, currentClassesStr, migrationFile.SQL)
		if err != nil {
			return "", "", fmt.Errorf("failed to apply migration %s: %v", migrationFile.Name, err)
		}

		currentEnumsStr = newSchema.enumsStr
		currentClassesStr = newSchema.classesStr
	}

	return currentEnumsStr, currentClassesStr, nil
}

func (ae *ApplyEngine) readMigrationFiles() ([]*MigrationFile, error) {
	if _, err := os.Stat(ae.migrationDir); os.IsNotExist(err) {
		return []*MigrationFile{}, nil
	}

	files, err := os.ReadDir(ae.migrationDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %v", err)
	}

	var migrationFiles []*MigrationFile

	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		queryFile := filepath.Join(file.Name(), constants.QUERY_FILE_NAME)

		timestamp, err := ae.extractTimestamp(file.Name())
		if err != nil {
			continue
		}

		filePath := filepath.Join(ae.migrationDir, queryFile)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %v", file.Name(), err)
		}

		migrationFile := &MigrationFile{
			Name:      file.Name(),
			Path:      filePath,
			Timestamp: timestamp,
			SQL:       string(content),
		}

		migrationFiles = append(migrationFiles, migrationFile)
	}

	return migrationFiles, nil
}

func (ae *ApplyEngine) extractTimestamp(filename string) (int64, error) {
	parts := strings.Split(filename, "_")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	timestampStr := parts[0]
	if len(timestampStr) != 14 {
		return 0, fmt.Errorf("invalid timestamp format in filename: %s", filename)
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse timestamp from filename %s: %v", filename, err)
	}

	return timestamp, nil
}

func BuildSchemaFromMigrations() (string, string, error) {
	engine := NewApplyEngine()
	return engine.BuildProgressiveSchema()
}