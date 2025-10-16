package types

import (
    "fmt"
    "strings"

    "github.com/rit3sh-x/blaze/core/ast"
    "github.com/rit3sh-x/blaze/core/ast/class"
    "github.com/rit3sh-x/blaze/core/ast/field"
    "github.com/rit3sh-x/blaze/core/constants"
)

type ClassGenerator struct {
    class         *class.Class
    ast           *ast.SchemaAST
    main          strings.Builder
}

func NewClassGenerator(cls *class.Class, schemaAST *ast.SchemaAST) *ClassGenerator {
    return &ClassGenerator{
        class:         cls,
        ast:           schemaAST,
    }
}

func (cg *ClassGenerator) Generate() string {
    cg.main.Reset()

    cg.generateMainType()
    cg.main.WriteString("\n")

    cg.generateUniqueType()
    cg.main.WriteString("\n")

    cg.generateGetUniqueFunction()

    return cg.main.String()
}

func (cg *ClassGenerator) generateMainType() {
    cg.main.WriteString(fmt.Sprintf("type %s struct {\n", cg.class.Name))
    
    for _, field := range cg.class.Attributes.Fields {
        fieldType := cg.getGoType(field)
        cg.main.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n",
            cg.toExportedName(field.GetName()),
            fieldType,
            field.GetName()))
    }
    
    cg.main.WriteString("}\n")
}

func (cg *ClassGenerator) generateUniqueType() {
    uniqueFields := cg.getUniqueFields()
    if len(uniqueFields) == 0 {
        return
    }

    cg.main.WriteString(fmt.Sprintf("type %s struct {\n", uniqueAttributeTypeName(cg.class.Name)))
    
    for _, field := range uniqueFields {
        fieldType := cg.getGoType(field)
        fieldType = strings.TrimPrefix(fieldType, "*")
        cg.main.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n",
            cg.toExportedName(field.GetName()),
            fieldType,
            field.GetName()))
    }
    
    cg.main.WriteString("}\n")
}

func (cg *ClassGenerator) getUniqueFields() []*field.Field {
    var uniqueFields []*field.Field

    if cg.class.HasPrimaryKey() {
        pkFields := cg.class.GetPrimaryKeyFields()
        for _, pkFieldName := range pkFields {
            if f := cg.class.Attributes.GetFieldByName(pkFieldName); f != nil {
                uniqueFields = append(uniqueFields, f)
            }
        }
    }

    for _, field := range cg.class.Attributes.Fields {
        if field.IsUnique() && !field.IsPrimaryKey() {
            uniqueFields = append(uniqueFields, field)
        }
    }

    return uniqueFields
}

func (cg *ClassGenerator) getGoType(field *field.Field) string {
    baseType := field.GetBaseType()
    goType := ""

    if scalarType := cg.getScalarType(baseType); scalarType != "" {
        goType = cg.scalarToGoType(scalarType)
    } else if cg.ast.GetEnumByName(baseType) != nil {
        goType = baseType
    } else if cg.ast.GetClassByName(baseType) != nil {
        goType = baseType
    } else {
        goType = "interface{}"
    }

    if field.IsArray() {
        goType = "[]" + goType
    }

    if field.IsOptional() && !field.IsArray() {
        goType = "*" + goType
    }

    return goType
}

func (cg *ClassGenerator) getScalarType(typeName string) string {
    for _, scalarType := range constants.ScalarTypes {
        if string(scalarType) == typeName {
            return typeName
        }
    }
    return ""
}

func (cg *ClassGenerator) scalarToGoType(scalarType string) string {
    switch constants.ScalarType(scalarType) {
    case constants.INT:
        return "int32"
    case constants.BIGINT:
        return "int64"
    case constants.SMALLINT:
        return "int16"
    case constants.FLOAT:
        return "float64"
    case constants.NUMERIC:
        return "float64"
    case constants.STRING:
        return "string"
    case constants.BOOLEAN:
        return "bool"
    case constants.DATE:
        return "time.Time"
    case constants.TIMESTAMP:
        return "time.Time"
    case constants.JSON:
        return "interface{}"
    case constants.BYTES:
        return "[]byte"
    case constants.CHAR:
        return "string"
    default:
        return "interface{}"
    }
}

func (cg *ClassGenerator) toExportedName(name string) string {
    if len(name) == 0 {
        return name
    }
    return strings.ToUpper(name[:1]) + name[1:]
}

func (cg *ClassGenerator) generateGetUniqueFunction() {
    uniqueFields := cg.getUniqueFields()
    if len(uniqueFields) == 0 {
        return
    }

    funcName := fmt.Sprintf("get%s", uniqueAttributeTypeName(cg.class.Name))
    cg.main.WriteString(fmt.Sprintf("func (obj *%s) %s() *%s {\n", 
        cg.class.Name, funcName, uniqueAttributeTypeName(cg.class.Name)))
    cg.main.WriteString(fmt.Sprintf("\treturn &%s{\n", uniqueAttributeTypeName(cg.class.Name)))
    
    for _, field := range uniqueFields {
        exportedName := cg.toExportedName(field.GetName())
        if field.IsOptional() {
            cg.main.WriteString(fmt.Sprintf("\t\t%s: *obj.%s,\n", exportedName, exportedName))
        } else {
            cg.main.WriteString(fmt.Sprintf("\t\t%s: obj.%s,\n", exportedName, exportedName))
        }
    }
    
    cg.main.WriteString("\t}\n")
    cg.main.WriteString("}\n")
}

func uniqueAttributeTypeName(table string) string {
    return fmt.Sprintf("%sUniqueAttribues", table)
}