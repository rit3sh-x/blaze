package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast"
	"github.com/rit3sh-x/blaze/core/ast/class"
	"github.com/rit3sh-x/blaze/core/ast/field"
	"github.com/rit3sh-x/blaze/core/constants"
)

type ClassInfo struct {
	Name    string
	Type    string
	Uniques []string
}

type ClassGenerator struct {
	class   *class.Class
	ast     *ast.SchemaAST
	main    strings.Builder
	classes []ClassInfo
}

func NewClassGenerator(cls *class.Class, schemaAST *ast.SchemaAST) *ClassGenerator {
	return &ClassGenerator{
		class:   cls,
		ast:     schemaAST,
		classes: make([]ClassInfo, 0),
	}
}

func (cg *ClassGenerator) Generate() string {
	cg.main.Reset()

	mainType := cg.generateMainType()
	cg.main.WriteString("\n")

	uniqueType := cg.generateUniqueType()
	if uniqueType != "" {
		cg.main.WriteString("\n")
	}

	cg.generateRelationPermutations(mainType)

	classInfo := ClassInfo{
		Name:    cg.class.Name,
		Type:    mainType,
		Uniques: cg.getUniqueFieldNames(),
	}
	cg.classes = append(cg.classes, classInfo)

	return cg.main.String()
}

func (cg *ClassGenerator) generateMainType() string {
	cg.main.WriteString(fmt.Sprintf("type %s struct {\n", cg.class.Name))

	for _, field := range cg.class.Attributes.Fields {
		fieldType := cg.getGoType(field)
		cg.main.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n",
			cg.toExportedName(field.GetName()),
			fieldType,
			field.GetName()))
	}

	cg.main.WriteString("}\n")
	return cg.class.Name
}

func (cg *ClassGenerator) generateUniqueType() string {
	uniqueFields := cg.getUniqueFields()
	if len(uniqueFields) == 0 {
		return ""
	}

	typeName := fmt.Sprintf("%sUnique", cg.class.Name)
	cg.main.WriteString(fmt.Sprintf("type %s struct {\n", typeName))

	for _, field := range uniqueFields {
		if !cg.isBaseTypeOrEnum(field) {
			continue
		}

		fieldType := cg.getGoType(field)
		fieldType = strings.TrimPrefix(fieldType, "*")
		fieldType = strings.TrimPrefix(fieldType, "[]")
		fieldName := field.GetName()
		cg.main.WriteString(fmt.Sprintf("\t%s %s\n",
			cg.toExportedName(fieldName),
			fieldType))
	}

	cg.main.WriteString("}\n")
	return typeName
}

func (cg *ClassGenerator) generateRelationPermutations(mainType string) {
	relationFields := cg.getRelationFields()
	if len(relationFields) == 0 {
		return
	}

	combinations := cg.getCombinations(relationFields)

	for _, combo := range combinations {
		cg.generatePermutationType(mainType, combo)
		cg.main.WriteString("\n")
	}
}

func (cg *ClassGenerator) generatePermutationType(mainType string, relations []*field.Field) {
	relationNames := make([]string, len(relations))
	for i, rel := range relations {
		relationNames[i] = cg.toExportedName(rel.GetName())
	}
	sort.Strings(relationNames)

	typeName := mainType + "With" + strings.Join(relationNames, "And")

	cg.main.WriteString(fmt.Sprintf("type %s struct {\n", typeName))

	cg.main.WriteString(fmt.Sprintf("\t%s\n", mainType))

	sortedRelations := make([]*field.Field, len(relations))
	copy(sortedRelations, relations)
	sort.Slice(sortedRelations, func(i, j int) bool {
		return cg.toExportedName(sortedRelations[i].GetName()) < cg.toExportedName(sortedRelations[j].GetName())
	})

	for _, rel := range sortedRelations {
		relationType := cg.getGoType(rel)
		fieldName := rel.GetName()
		cg.main.WriteString(fmt.Sprintf("\t%s %s\n",
			fieldName,
			relationType))
	}

	cg.main.WriteString("}\n")
}

func (cg *ClassGenerator) getRelationFields() []*field.Field {
	var relations []*field.Field
	for _, f := range cg.class.Attributes.Fields {
		baseType := f.GetBaseType()
		if cg.getScalarType(baseType) == "" && cg.ast.GetEnumByName(baseType) == nil {
			relations = append(relations, f)
		}
	}
	return relations
}


func (cg *ClassGenerator) getCombinations(relations []*field.Field) [][]*field.Field {
	var result [][]*field.Field

	n := len(relations)
	for i := 1; i < (1 << n); i++ {
		var combo []*field.Field
		for j := 0; j < n; j++ {
			if i&(1<<j) != 0 {
				combo = append(combo, relations[j])
			}
		}
		result = append(result, combo)
	}

	return result
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

func (cg *ClassGenerator) isBaseTypeOrEnum(field *field.Field) bool {
	baseType := field.GetBaseType()

	if cg.getScalarType(baseType) != "" {
		return true
	}

	if cg.ast.GetEnumByName(baseType) != nil {
		return true
	}

	return false
}

func (cg *ClassGenerator) getUniqueFields() []*field.Field {
	var uniqueFields []*field.Field
	seen := make(map[string]bool)

	if cg.class.HasPrimaryKey() {
		pkFields := cg.class.GetPrimaryKeyFields()
		for _, pkFieldName := range pkFields {
			if f := cg.class.Attributes.GetFieldByName(pkFieldName); f != nil {
				uniqueFields = append(uniqueFields, f)
				seen[pkFieldName] = true
			}
		}
	}

	for _, field := range cg.class.Attributes.Fields {
		fieldName := field.GetName()
		if field.IsUnique() && !seen[fieldName] {
			uniqueFields = append(uniqueFields, field)
			seen[fieldName] = true
		}
	}

	return uniqueFields
}

func (cg *ClassGenerator) getUniqueFieldNames() []string {
	uniqueFields := cg.getUniqueFields()
	fieldNames := make([]string, 0, len(uniqueFields))

	for _, field := range uniqueFields {
		fieldNames = append(fieldNames, field.GetName())
	}

	return fieldNames
}

func (cg *ClassGenerator) GetClasses() []ClassInfo {
	return cg.classes
}