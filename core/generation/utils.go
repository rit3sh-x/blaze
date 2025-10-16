package generation

import (
    "fmt"
    "os"
    "sort"
    "strings"

    "github.com/rit3sh-x/blaze/core/ast"
    "github.com/rit3sh-x/blaze/core/constants"
    "github.com/rit3sh-x/blaze/core/generation/types"
)

func GenerateTypes(schemaAST *ast.SchemaAST) error {
    if schemaAST == nil {
        return nil
    }

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

    if err := os.MkdirAll(constants.CLIENT_DIR, 0755); err != nil {
        return fmt.Errorf("failed to create directory %s: %v", constants.CLIENT_DIR, err)
    }

    if err := os.WriteFile(constants.TYPES_FILE, []byte(content.String()), 0644); err != nil {
        return fmt.Errorf("failed to write types file: %v", err)
    }

    return nil
}