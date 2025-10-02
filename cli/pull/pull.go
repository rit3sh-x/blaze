package pull

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rit3sh-x/blaze/core/ast"
	"github.com/rit3sh-x/blaze/core/constants"
	"github.com/rit3sh-x/blaze/core/sync"
	"github.com/rit3sh-x/blaze/core/validation"
)

func PullSchema(client *pgxpool.Pool, ctx context.Context) error {
	enumSchema, arr, err := sync.GetEnums(client, ctx)
	if err != nil {
		log.Fatalf("failed to generate enum schema: %v", err)
	}

	classSchema, err := sync.GetClasses(client, ctx, arr)
	if err != nil {
		log.Fatalf("failed to generate class schema: %v", err)
	}

	schemaAST, err := ast.BuildSchemaAST(enumSchema, classSchema)
	if err != nil {
		return fmt.Errorf("failed to build AST: %v", err)
	}

	if err := validation.ValidateSchema(schemaAST); err != nil {
		return fmt.Errorf("schema validation failed: %v", err)
	}
	var fullSchema strings.Builder

	if enumSchema != "" {
		fullSchema.WriteString(enumSchema)
		if classSchema != "" {
			fullSchema.WriteString("\n\n")
		}
	}

	if classSchema != "" {
		fullSchema.WriteString(classSchema)
	}

	if fullSchema.Len() != 0 {
		WriteToFile(fullSchema.String(), constants.SCHEMA_FILE)
	}

	return nil
}

func WriteToFile(content string, filePath string) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("failed to open or create file: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		log.Fatalf("failed to write to file: %v", err)
	}
}