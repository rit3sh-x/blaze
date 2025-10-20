package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast"
	"github.com/rit3sh-x/blaze/core/ast/class"
	"github.com/rit3sh-x/blaze/core/ast/field"
	"github.com/rit3sh-x/blaze/core/constants"
	"github.com/rit3sh-x/blaze/core/utils"
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

	relationType := cg.generateRelationsType()
	if relationType != "" {
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

func (cg *ClassGenerator) GenerateCompositeTypes() string {
	var content strings.Builder
	compositeTypes := cg.generateCompositeTypesInternal(&content)

	if len(compositeTypes) > 0 {
		return content.String()
	}
	return ""
}

func (cg *ClassGenerator) GenerateWhereUniqueInput() string {
	var content strings.Builder
	compositeTypes := []string{}

	if cg.class.HasPrimaryKey() {
		pkFields := cg.class.GetPrimaryKeyFields()
		if len(pkFields) > 1 {
			sortedPKFields := make([]string, len(pkFields))
			copy(sortedPKFields, pkFields)
			sort.Strings(sortedPKFields)

			typeNameParts := []string{cg.class.Name}
			for _, pkFieldName := range sortedPKFields {
				typeNameParts = append(typeNameParts, utils.ToExportedName(pkFieldName))
			}
			typeName := strings.Join(typeNameParts, "") + "Composite"
			compositeTypes = append(compositeTypes, typeName)
		}
	}

	for _, directive := range cg.class.Attributes.Directives {
		if directive.Name == constants.CLASS_ATTR_UNIQUE {
			fields, err := directive.GetFields()
			if err != nil || len(fields) < 2 {
				continue
			}

			sortedFields := make([]string, len(fields))
			copy(sortedFields, fields)
			sort.Strings(sortedFields)

			typeNameParts := []string{cg.class.Name}
			for _, fieldName := range sortedFields {
				typeNameParts = append(typeNameParts, utils.ToExportedName(fieldName))
			}
			typeName := strings.Join(typeNameParts, "") + "Composite"
			compositeTypes = append(compositeTypes, typeName)
		}
	}

	uniqueFields := cg.getUniqueFields()
	if len(uniqueFields) == 0 && len(compositeTypes) == 0 {
		return ""
	}

	typeName := fmt.Sprintf("%sWhereUniqueInput", cg.class.Name)
	content.WriteString(fmt.Sprintf("type %s struct {\n", typeName))

	for _, field := range uniqueFields {
		fieldType := utils.GetGoType(field, cg.ast)
		fieldName := field.GetName()
		content.WriteString(fmt.Sprintf("\t%s *%s\n",
			utils.ToExportedName(fieldName),
			strings.TrimPrefix(fieldType, "*")))
	}

	for _, compositeTypeName := range compositeTypes {
		fieldName := strings.TrimPrefix(compositeTypeName, cg.class.Name)
		fieldName = strings.TrimSuffix(fieldName, "Composite")

		content.WriteString(fmt.Sprintf("\t%s *%s\n",
			fieldName,
			compositeTypeName))
	}

	content.WriteString("}\n")
	return content.String()
}

func (cg *ClassGenerator) generateMainType() string {
	cg.main.WriteString(fmt.Sprintf("type %s struct {\n", cg.class.Name))
	relationFields := cg.getRelationFields()

	for _, field := range cg.class.Attributes.Fields {
		skip := false
		for _, rf := range relationFields {
			if rf.GetName() == field.GetName() {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		fieldType := utils.GetGoType(field, cg.ast)
		cg.main.WriteString(fmt.Sprintf("\t%s %s\n",
			utils.ToExportedName(field.GetName()),
			fieldType))
	}

	if len(relationFields) != 0 {
		cg.main.WriteString(fmt.Sprintf("\tRelations %sRelations\n", cg.class.Name))
	}
	cg.main.WriteString("}\n")
	return cg.class.Name
}

func (cg *ClassGenerator) generateCompositeTypesInternal(content *strings.Builder) []string {
	var compositeTypes []string

	if cg.class.HasPrimaryKey() {
		pkFields := cg.class.GetPrimaryKeyFields()
		if len(pkFields) > 1 {
			sortedPKFields := make([]string, len(pkFields))
			copy(sortedPKFields, pkFields)
			sort.Strings(sortedPKFields)

			typeNameParts := []string{cg.class.Name}
			for _, pkFieldName := range sortedPKFields {
				typeNameParts = append(typeNameParts, utils.ToExportedName(pkFieldName))
			}
			typeName := strings.Join(typeNameParts, "") + "Composite"

			content.WriteString(fmt.Sprintf("type %s struct {\n", typeName))
			for _, pkFieldName := range sortedPKFields {
				if f := cg.class.Attributes.GetFieldByName(pkFieldName); f != nil {
					fieldType := utils.GetGoType(f, cg.ast)
					fieldType = strings.TrimPrefix(fieldType, "*")
					content.WriteString(fmt.Sprintf("\t%s %s\n",
						utils.ToExportedName(pkFieldName),
						fieldType))
				}
			}
			content.WriteString("}\n\n")

			compositeTypes = append(compositeTypes, typeName)
		}
	}

	for _, directive := range cg.class.Attributes.Directives {
		if directive.Name == constants.CLASS_ATTR_UNIQUE {
			fields, err := directive.GetFields()
			if err != nil || len(fields) < 2 {
				continue
			}

			sortedFields := make([]string, len(fields))
			copy(sortedFields, fields)
			sort.Strings(sortedFields)

			typeNameParts := []string{cg.class.Name}
			for _, fieldName := range sortedFields {
				typeNameParts = append(typeNameParts, utils.ToExportedName(fieldName))
			}
			typeName := strings.Join(typeNameParts, "") + "Composite"

			content.WriteString(fmt.Sprintf("type %s struct {\n", typeName))
			for _, fieldName := range sortedFields {
				field := cg.class.Attributes.GetFieldByName(fieldName)
				if field != nil {
					fieldType := utils.GetGoType(field, cg.ast)
					fieldType = strings.TrimPrefix(fieldType, "*")
					content.WriteString(fmt.Sprintf("\t%s %s\n",
						utils.ToExportedName(fieldName),
						fieldType))
				}
			}
			content.WriteString("}\n\n")

			compositeTypes = append(compositeTypes, typeName)
		}
	}

	relationComposites := cg.generateRelationCompositeType(content)
	compositeTypes = append(compositeTypes, relationComposites...)

	return compositeTypes
}

func (cg *ClassGenerator) generateRelationCompositeType(content *strings.Builder) []string {
	var compositeTypes []string
	relationFields := cg.getRelationFields()

	for _, relField := range relationFields {
		baseType := relField.GetBaseType()
		relatedClass := cg.ast.GetClassByName(baseType)

		if relatedClass == nil {
			continue
		}

		if relField.HasRelation() {
			relation := relField.AttributeDefinition.Relation
			if relation != nil && len(relation.From) > 1 {
				sortedFields := make([]string, len(relation.From))
				copy(sortedFields, relation.From)
				sort.Strings(sortedFields)

				typeNameParts := []string{cg.class.Name}
				for _, fieldName := range sortedFields {
					typeNameParts = append(typeNameParts, utils.ToExportedName(fieldName))
				}
				typeName := strings.Join(typeNameParts, "") + "Composite"

				alreadyExists := false
				for _, existing := range compositeTypes {
					if existing == typeName {
						alreadyExists = true
						break
					}
				}

				if !alreadyExists {
					content.WriteString(fmt.Sprintf("type %s struct {\n", typeName))
					for _, fieldName := range sortedFields {
						field := cg.class.Attributes.GetFieldByName(fieldName)
						if field != nil {
							fieldType := utils.GetGoType(field, cg.ast)
							fieldType = strings.TrimPrefix(fieldType, "*")
							content.WriteString(fmt.Sprintf("\t%s %s\n",
								utils.ToExportedName(fieldName),
								fieldType))
						}
					}
					content.WriteString("}\n\n")
					compositeTypes = append(compositeTypes, typeName)
				}
			}
		} else {
			for _, relatedField := range relatedClass.Attributes.Fields {
				if relatedField.GetBaseType() == cg.class.Name && relatedField.HasRelation() {
					relation := relatedField.AttributeDefinition.Relation
					if relation != nil && len(relation.To) > 1 {
						sortedFields := make([]string, len(relation.To))
						copy(sortedFields, relation.To)
						sort.Strings(sortedFields)

						typeNameParts := []string{cg.class.Name}
						for _, fieldName := range sortedFields {
							typeNameParts = append(typeNameParts, utils.ToExportedName(fieldName))
						}
						typeName := strings.Join(typeNameParts, "") + "Composite"

						alreadyExists := false
						for _, existing := range compositeTypes {
							if existing == typeName {
								alreadyExists = true
								break
							}
						}

						if !alreadyExists {
							content.WriteString(fmt.Sprintf("type %s struct {\n", typeName))
							for _, fieldName := range sortedFields {
								field := cg.class.Attributes.GetFieldByName(fieldName)
								if field != nil {
									fieldType := utils.GetGoType(field, cg.ast)
									fieldType = strings.TrimPrefix(fieldType, "*")
									content.WriteString(fmt.Sprintf("\t%s %s\n",
										utils.ToExportedName(fieldName),
										fieldType))
								}
							}
							content.WriteString("}\n\n")
							compositeTypes = append(compositeTypes, typeName)
						}
					}
				}
			}
		}
	}

	return compositeTypes
}

func (cg *ClassGenerator) generateRelationsType() string {
	relationFields := cg.getRelationFields()
	if len(relationFields) == 0 {
		return ""
	}

	typeName := fmt.Sprintf("%sRelations", cg.class.Name)
	cg.main.WriteString(fmt.Sprintf("type %s struct {\n", typeName))

	for _, field := range relationFields {
		fieldType := utils.GetGoType(field, cg.ast)
		fieldName := field.GetName()
		cg.main.WriteString(fmt.Sprintf("\t%s %s\n",
			utils.ToExportedName(fieldName),
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
		relationNames[i] = utils.ToExportedName(rel.GetName())
	}
	sort.Strings(relationNames)

	typeName := mainType + "With" + strings.Join(relationNames, "And")

	cg.main.WriteString(fmt.Sprintf("type %s struct {\n", typeName))
	cg.main.WriteString(fmt.Sprintf("\t%s\n", mainType))

	for _, rel := range relations {
		relationType := utils.GetGoType(rel, cg.ast)
		fieldName := rel.GetName()
		cg.main.WriteString(fmt.Sprintf("\t%s %s\n",
			utils.ToExportedName(fieldName),
			relationType))
	}

	cg.main.WriteString("}\n")
}

func (cg *ClassGenerator) getRelationFields() []*field.Field {
	var relations []*field.Field
	for _, f := range cg.class.Attributes.Fields {
		baseType := f.GetBaseType()
		if utils.GetScalarType(baseType) == "" && cg.ast.GetEnumByName(baseType) == nil {
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

		sort.Slice(combo, func(a, b int) bool {
			return utils.ToExportedName(combo[a].GetName()) < utils.ToExportedName(combo[b].GetName())
		})

		result = append(result, combo)
	}

	return result
}

func (cg *ClassGenerator) getUniqueFields() []*field.Field {
	var uniqueFields []*field.Field
	seen := make(map[string]bool)

	if cg.class.HasPrimaryKey() {
		pkFields := cg.class.GetPrimaryKeyFields()
		if len(pkFields) == 1 {
			if f := cg.class.Attributes.GetFieldByName(pkFields[0]); f != nil {
				uniqueFields = append(uniqueFields, f)
				seen[pkFields[0]] = true
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

	for _, directive := range cg.class.Attributes.Directives {
		if directive.Name == constants.CLASS_ATTR_UNIQUE {
			fields, err := directive.GetFields()
			if err != nil {
				continue
			}
			if len(fields) == 1 {
				if f := cg.class.Attributes.GetFieldByName(fields[0]); f != nil && !seen[fields[0]] {
					uniqueFields = append(uniqueFields, f)
					seen[fields[0]] = true
				}
			}
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
