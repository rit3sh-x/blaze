package validate

import (
	"fmt"
	"os"

	"github.com/rit3sh-x/blaze/core/ast"
	"github.com/rit3sh-x/blaze/core/constants"
	"github.com/rit3sh-x/blaze/core/utils"
	"github.com/rit3sh-x/blaze/core/validation"
)

func ValidateSchemaFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("schema file not found: %s", filePath)
	}

	enumContent, classContent, err := utils.ReadAndSeparateSchema(filePath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %v", err)
	}

	schemaAST, err := ast.BuildSchemaAST(enumContent, classContent)
	if err != nil {
		return fmt.Errorf("failed to build AST: %v", err)
	}

	if err := validation.ValidateSchema(schemaAST); err != nil {
		return fmt.Errorf("schema validation failed: %v", err)
	}

	return nil
}

func ValidateSchemaFileWithDetails(filePath string) ([]validation.ValidationError, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("schema file not found: %s", filePath)
	}

	enumContent, classContent, err := utils.ReadAndSeparateSchema(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %v", err)
	}

	schemaAST, err := ast.BuildSchemaAST(enumContent, classContent)
	if err != nil {
		return nil, fmt.Errorf("failed to build AST: %v", err)
	}

	validationErrors, err := validation.ValidateSchemaWithDetails(schemaAST)
	return validationErrors, err
}

func ValidateDefaultSchema() error {
	return ValidateSchemaFile(constants.SCHEMA_FILE)
}

func PrintValidationResults(filePath string) {
	validationErrors, err := ValidateSchemaFileWithDetails(filePath)

	if err != nil {
		fmt.Printf("%sValidation failed: %v%s\n", constants.RED, err, constants.RESET)
		return
	}

	if len(validationErrors) == 0 {
		fmt.Printf("%s✔ Schema validation passed! No errors found.%s\n", constants.GREEN, constants.RESET)
		return
	}

	fmt.Printf("%sSchema validation failed with %d error(s):%s\n\n", constants.RED, len(validationErrors), constants.RESET)
	for i, valErr := range validationErrors {
		fmt.Printf("%s%d.%s %s[%s]%s %s\n   %sLocation:%s %s\n\n",
			constants.CYAN, i+1, constants.RESET,
			constants.YELLOW, valErr.Type, constants.RESET, valErr.Message,
			constants.BLUE, constants.RESET, valErr.Location)
	}
}

func PrintValidationSummary(filePath string) {
	validationErrors, err := ValidateSchemaFileWithDetails(filePath)

	if err != nil {
		fmt.Printf("%s[ERROR]%s %v\n", constants.RED, constants.RESET, err)
		return
	}

	if len(validationErrors) == 0 {
		fmt.Printf("%s[PASS]%s Schema validation successful\n", constants.GREEN, constants.RESET)
		return
	}

	fmt.Printf("%s[FAIL]%s %d validation error(s) found\n", constants.RED, constants.RESET, len(validationErrors))
}

func PrintColorfulValidationResults(filePath string) {
	fmt.Printf("%s🔍 Validating schema: %s%s%s\n", constants.BLUE, constants.CYAN, filePath, constants.RESET)

	validationErrors, err := ValidateSchemaFileWithDetails(filePath)

	if err != nil {
		fmt.Printf("\n%s💥 VALIDATION FAILED%s\n", constants.RED, constants.RESET)
		fmt.Printf("%sError: %s%s\n", constants.RED, err, constants.RESET)
		return
	}

	if len(validationErrors) == 0 {
		fmt.Printf("\n%s✔ VALIDATION SUCCESS%s\n", constants.GREEN, constants.RESET)
		fmt.Printf("%sYour schema is perfect! No errors found.%s\n", constants.GREEN, constants.RESET)
		return
	}

	fmt.Printf("\n%sVALIDATION ISSUES DETECTED%s\n", constants.YELLOW, constants.RESET)
	fmt.Printf("%sFound %d validation error(s):%s\n\n", constants.RED, len(validationErrors), constants.RESET)

	for i, valErr := range validationErrors {
		fmt.Printf("%s┌─ Error #%d%s\n", constants.CYAN, i+1, constants.RESET)
		fmt.Printf("%s│%s %sType:%s %s[%s]%s\n",
			constants.CYAN, constants.RESET,
			constants.YELLOW, constants.RESET,
			constants.RED, valErr.Type, constants.RESET)
		fmt.Printf("%s│%s %sMessage:%s %s\n",
			constants.CYAN, constants.RESET,
			constants.YELLOW, constants.RESET, valErr.Message)
		fmt.Printf("%s│%s %sLocation:%s %s\n",
			constants.CYAN, constants.RESET,
			constants.YELLOW, constants.RESET, valErr.Location)
		fmt.Printf("%s└─%s\n\n", constants.CYAN, constants.RESET)
	}
}