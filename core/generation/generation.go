package generation

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast"
	"github.com/rit3sh-x/blaze/core/constants"
	"github.com/rit3sh-x/blaze/core/generation/db"
	"github.com/rit3sh-x/blaze/core/generation/hooks"
	"github.com/rit3sh-x/blaze/core/generation/types"
)

func Generate(schemaAST *ast.SchemaAST) error {
	if schemaAST == nil {
		return fmt.Errorf("schema AST is nil")
	}

	if err := GenerateTypes(schemaAST); err != nil {
		return fmt.Errorf("failed to generate types: %w", err)
	}

	if err := GenerateHooks(schemaAST); err != nil {
		return fmt.Errorf("failed to generate hooks: %w", err)
	}

	if err := GenerateDBUtils(schemaAST); err != nil {
		return fmt.Errorf("failed to generate DB utilities: %w", err)
	}

	return nil
}

func GenerateTypes(schemaAST *ast.SchemaAST) error {
	var content strings.Builder
	globalImports := make(map[string]bool)

	content.WriteString("package client\n\n")

	for _, cls := range schemaAST.Classes {
		for _, field := range cls.Attributes.Fields {
			baseType := field.GetBaseType()
			for _, scalarType := range constants.ScalarTypes {
				if string(scalarType) == baseType {
					if scalarType == constants.DATE || scalarType == constants.TIMESTAMP {
						globalImports["time"] = true
					}
					break
				}
			}
		}
	}

	if len(globalImports) > 0 {
		content.WriteString("import (\n")
		importList := make([]string, 0, len(globalImports))
		for imp := range globalImports {
			importList = append(importList, imp)
		}
		sort.Strings(importList)
		for _, imp := range importList {
			content.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
		}
		content.WriteString(")\n\n")
	}

	if len(schemaAST.Enums) > 0 {
		content.WriteString("// ==================== ENUMS ====================\n\n")
		enumNames := make([]string, 0, len(schemaAST.Enums))
		for name := range schemaAST.Enums {
			enumNames = append(enumNames, name)
		}
		sort.Strings(enumNames)
		for _, enumName := range enumNames {
			enum := schemaAST.Enums[enumName]
			enumGen := types.NewEnumGenerator(enum)
			content.WriteString(enumGen.Generate())
		}
	}

	if len(schemaAST.Classes) > 0 {
		content.WriteString("// ==================== TYPES ====================\n\n")
		for _, cls := range schemaAST.Classes {
			classGen := types.NewClassGenerator(cls, schemaAST)
			classCode := classGen.Generate()
			content.WriteString(classCode)
			content.WriteString("\n")
		}
	}

	content.WriteString("// ==================== COMPOSITE UNIQUE CONSTRAINT TYPES ====================\n\n")
	for _, cls := range schemaAST.Classes {
		classGen := types.NewClassGenerator(cls, schemaAST)
		compositeCode := classGen.GenerateCompositeTypes()
		if compositeCode != "" {
			content.WriteString(compositeCode)
		}
	}

	content.WriteString("// ==================== UNIQUE INPUT TYPES ====================\n\n")
	for _, cls := range schemaAST.Classes {
		classGen := types.NewClassGenerator(cls, schemaAST)
		whereUniqueCode := classGen.GenerateWhereUniqueInput()
		if whereUniqueCode != "" {
			content.WriteString(whereUniqueCode)
			content.WriteString("\n")
		}
	}

	content.WriteString("// ==================== UNIQUE INPUT CONSTRUCTORS ====================\n\n")
	for _, cls := range schemaAST.Classes {
		uniqueConstructors := hooks.GenerateUniqueConstructors(cls, schemaAST)
		content.WriteString(uniqueConstructors)
	}

	if err := os.MkdirAll(constants.CLIENT_DIR, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", constants.CLIENT_DIR, err)
	}

	if err := os.WriteFile(constants.TYPES_FILE, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write types file: %v", err)
	}

	return nil
}

func GenerateHooks(schemaAST *ast.SchemaAST) error {
	var content strings.Builder

	content.WriteString("package client\n\n")
	content.WriteString("import (\n")
	content.WriteString("\t\"time\"\n")
	content.WriteString("\t\"fmt\"\n")
	content.WriteString(")\n\n")

	content.WriteString(hooks.GenerateCoreTypes())
	content.WriteString("\n\n")

	classNames := make([]string, len(schemaAST.Classes))
	for i, cls := range schemaAST.Classes {
		classNames[i] = cls.Name
	}

	content.WriteString("// ==================== PREDICATES ====================\n\n")
	content.WriteString(hooks.GeneratePredicateVars(classNames))
	content.WriteString("\n")

	for _, cls := range schemaAST.Classes {
		content.WriteString(fmt.Sprintf("// ==================== %s PREDICATES ====================\n\n", strings.ToUpper(cls.Name)))
		predicates := hooks.GenerateClassPredicates(cls, schemaAST)
		content.WriteString(predicates)
		content.WriteString("\n")

		compositePredicates := hooks.GenerateCompositePredicates(cls, schemaAST)
		if compositePredicates != "" {
			content.WriteString("// Composite predicates for WHERE clauses\n")
			content.WriteString(compositePredicates)
			content.WriteString("\n")
		}
	}

	content.WriteString("// ==================== LOGICAL OPERATORS ====================\n\n")
	content.WriteString(hooks.GenerateLogicalOperators())
	content.WriteString("\n\n")

	content.WriteString("// ==================== ORDER FUNCTIONS ====================\n\n")
	content.WriteString(hooks.GenerateOrderFunctions())
	content.WriteString("\n\n")

	content.WriteString("// ==================== FIELD CONSTANTS ====================\n\n")
	content.WriteString(hooks.GenerateFieldConstants(schemaAST))
	content.WriteString("\n\n")

	for _, cls := range schemaAST.Classes {
		content.WriteString(fmt.Sprintf("// ==================== %s QUERY ====================\n\n", strings.ToUpper(cls.Name)))
		content.WriteString(hooks.GenerateQueryBuilder(cls, schemaAST))
		content.WriteString("\n")

		content.WriteString(fmt.Sprintf("// ==================== %s CREATE ====================\n\n", strings.ToUpper(cls.Name)))
		content.WriteString(hooks.GenerateCreateBuilder(cls, schemaAST))
		content.WriteString("\n")

		content.WriteString(fmt.Sprintf("// ==================== %s UPDATE ====================\n\n", strings.ToUpper(cls.Name)))
		content.WriteString(hooks.GenerateUpdateBuilder(cls, schemaAST))
		content.WriteString("\n")

		content.WriteString(fmt.Sprintf("// ==================== %s UPDATE ONE ====================\n\n", strings.ToUpper(cls.Name)))
		content.WriteString(hooks.GenerateUpdateOneBuilder(cls, schemaAST))
		content.WriteString("\n")

		content.WriteString(fmt.Sprintf("// ==================== %s DELETE ====================\n\n", strings.ToUpper(cls.Name)))
		content.WriteString(hooks.GenerateDeleteBuilder(cls, schemaAST))
		content.WriteString("\n")

		content.WriteString(fmt.Sprintf("// ==================== %s CLIENT ====================\n\n", strings.ToUpper(cls.Name)))
		content.WriteString(hooks.GenerateClassClient(cls, schemaAST))
		content.WriteString("\n")
	}

	if err := os.MkdirAll(constants.CLIENT_DIR, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", constants.CLIENT_DIR, err)
	}

	if err := os.WriteFile(constants.HOOKS_FILE, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to write hooks file: %v", err)
	}

	return nil
}

func GenerateDBUtils(schemaAST *ast.SchemaAST) error {
	content := db.GenerateDBUtils()

	classNames := []string{}
	for _, cls := range schemaAST.Classes {
		classNames = append(classNames, cls.Name)
	}
	content += db.GenerateClientAccessors(classNames)

	if err := os.MkdirAll(constants.CLIENT_DIR, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", constants.CLIENT_DIR, err)
	}

	if err := os.WriteFile(constants.UTIL_FILE, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write utility file: %v", err)
	}
	return nil
}
