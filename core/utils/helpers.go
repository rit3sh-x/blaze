package utils

import (
	"strings"

	"github.com/rit3sh-x/blaze/core/ast/field"
	"github.com/rit3sh-x/blaze/core/ast"
	"github.com/rit3sh-x/blaze/core/constants"
)

func GetGoType(field *field.Field, ast *ast.SchemaAST) string {
	baseType := field.GetBaseType()
	goType := ""

	if scalarType := GetScalarType(baseType); scalarType != "" {
		goType = ScalarToGoType(scalarType)
	} else if ast.GetEnumByName(baseType) != nil {
		goType = baseType
	} else if ast.GetClassByName(baseType) != nil {
		goType = "*" + baseType
	} else {
		goType = "interface{}"
	}

	if field.IsArray() {
		goType = "[]" + goType
	}

	if field.IsOptional() && !strings.HasPrefix(goType, "*") {
		goType = "*" + goType
	}

	return goType
}

func GetScalarType(typeName string) string {
	for _, scalarType := range constants.ScalarTypes {
		if string(scalarType) == typeName {
			return typeName
		}
	}
	return ""
}

func ScalarToGoType(scalarType string) string {
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

func ToExportedName(name string) string {
	if len(name) == 0 {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}