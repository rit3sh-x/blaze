package class

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rit3sh-x/blaze/core/constants"
)

type Column struct {
	Name            string
	DataType        string
	IsNullable      bool
	IsArray         bool
	ColumnDefault   string
	OrdinalPosition int8
}

type Constraint struct {
	Name    string
	Type    string
	Columns []string
}

type Index struct {
	Name      string
	IsUnique  bool
	IsPrimary bool
	Fields    []string
}

type Relation struct {
	Name              string
	FkColumns         []string
	ReferencedTable   string
	ReferencedColumns []string
	UpdateRule        string
	DeleteRule        string
}

type ClassData struct {
	Name        string
	Columns     []Column
	Constraints []Constraint
	Indexes     []Index
	Relations   []Relation
}

func mapPostgreSQLType(pgType string, enums []string) (string, error) {
	if Contains(enums, pgType) {
		return pgType, nil
	}

	pgType = strings.TrimSpace(strings.ToLower(pgType))

	if blazeType, exists := constants.PGTypeMapping[pgType]; exists {
		return blazeType, nil
	}
	return "", fmt.Errorf("unsupported PostgreSQL type: %s", pgType)
}

func mapConstraintAction(pgAction string) string {
	if blazeAction, exists := constants.PGConstraintActionMapping[strings.ToUpper(pgAction)]; exists {
		return blazeAction
	}
	return constants.ON_DELETE_NO_ACTION
}

func isDefaultCallback(defaultVal string) (string, bool) {
	if defaultVal == "" || defaultVal == "null" {
		return "", false
	}

	defaultVal = strings.TrimSpace(strings.ToLower(defaultVal))

	if strings.Contains(defaultVal, "gen_random_uuid") {
		return constants.DEFAULT_UUID_CALLBACK, true
	}
	if strings.Contains(defaultVal, "current_timestamp") {
		return constants.DEFAULT_NOW_CALLBACK, true
	}
	if strings.Contains(defaultVal, "nextval") {
		return constants.DEFAULT_AUTOINCREMENT_CALLBACK, true
	}

	return "", false
}

func GenerateClassSchema(classData []ClassData, enums []string) string {
	if len(classData) == 0 {
		return ""
	}

	var schema strings.Builder

	for i, class := range classData {
		if i > 0 {
			schema.WriteString("\n\n")
		}

		schema.WriteString(fmt.Sprintf(constants.KEYWORD_CLASS+" %s {\n", class.Name))

		sort.Slice(class.Columns, func(i, j int) bool {
			return class.Columns[i].OrdinalPosition < class.Columns[j].OrdinalPosition
		})

		for _, column := range class.Columns {
			fieldType, err := mapPostgreSQLType(column.DataType, enums)
			if err != nil {
				return err.Error()
			}

			if column.IsArray {
				fieldType += "[]"
			}

			isPrimaryKey := isColumnPrimaryKey(column.Name, class.Constraints)
			if column.IsNullable && !isPrimaryKey {
				fieldType += "?"
			}

			schema.WriteString(fmt.Sprintf("  %-10s %s", column.Name, fieldType))

			var attributes []string

			if isPrimaryKey {
				attributes = append(attributes, "@primaryKey")
			}

			if isColumnUnique(column.Name, class.Constraints) && !isPrimaryKey {
				attributes = append(attributes, "@unique")
			}

			defualtValue := column.ColumnDefault

			if defualtValue != "null" && defualtValue != "" {
				if callback, isCallback := isDefaultCallback(defualtValue); isCallback {
					attributes = append(attributes, fmt.Sprintf("@default(%s)", callback))
				} else {
					if strings.Contains(strings.ToLower(fieldType), "string") || strings.Contains(strings.ToLower(fieldType), "char") {
						if !strings.Contains(defualtValue, "(") {
							attributes = append(attributes, fmt.Sprintf("@default(\"%s\")", defualtValue))
						}
					} else if strings.Contains(strings.ToLower(fieldType), "int") || strings.Contains(strings.ToLower(fieldType), "float") || strings.Contains(strings.ToLower(fieldType), "numeric") || strings.Contains(strings.ToLower(fieldType), "boolean") {
						attributes = append(attributes, fmt.Sprintf("@default(%s)", defualtValue))
					} else {
						attributes = append(attributes, fmt.Sprintf("@default(%s)", defualtValue))
					}
				}
			}

			if len(attributes) > 0 {
				schema.WriteString(" " + strings.Join(attributes, " "))
			}

			schema.WriteString("\n")
		}

		for _, relation := range class.Relations {
			relationFieldType := relation.ReferencedTable
			if len(relation.FkColumns) == 1 && isColumnNullable(relation.FkColumns[0], class.Columns) {
				relationFieldType += "?"
			}

			relationFieldName := relation.ReferencedTable

			schema.WriteString(fmt.Sprintf("\n  %-10s %s", relationFieldName, relationFieldType))

			fkCols := fmt.Sprintf("[%s]", strings.Join(relation.FkColumns, ", "))
			refCols := fmt.Sprintf("[%s]", strings.Join(relation.ReferencedColumns, ", "))

			relationAttr := fmt.Sprintf("@relation(%s, %s", fkCols, refCols)

			if relation.DeleteRule != "" && strings.ToUpper(relation.DeleteRule) != "NO ACTION" {
				relationAttr += fmt.Sprintf(", onDelete: %s", mapConstraintAction(relation.DeleteRule))
			}

			if relation.UpdateRule != "" && strings.ToUpper(relation.UpdateRule) != "NO ACTION" {
				relationAttr += fmt.Sprintf(", onUpdate: %s", mapConstraintAction(relation.UpdateRule))
			}

			if relation.Name != "" {
				relationAttr += fmt.Sprintf(", name: %s", relation.Name)
			}

			relationAttr += ")"
			schema.WriteString(" " + relationAttr)
			schema.WriteString("\n")
		}

		var classAttributes []string

		compositePK := getCompositePrimaryKey(class.Constraints)
		if len(compositePK) > 1 {
			pkCols := fmt.Sprintf("[%s]", strings.Join(compositePK, ", "))
			classAttributes = append(classAttributes, fmt.Sprintf("@@primaryKey(%s)", pkCols))
		}

		for _, constraint := range class.Constraints {
			if constraint.Type == "UNIQUE" && len(constraint.Columns) > 1 {
				uniqueCols := fmt.Sprintf("[%s]", strings.Join(constraint.Columns, ", "))
				classAttributes = append(classAttributes, fmt.Sprintf("@@unique(%s)", uniqueCols))
			}
		}

		for _, index := range class.Indexes {
			if !index.IsPrimary && len(index.Fields) > 1 {
				indexCols := fmt.Sprintf("[%s]", strings.Join(index.Fields, ", "))
				if index.IsUnique {
					classAttributes = append(classAttributes, fmt.Sprintf("@@unique(%s)", indexCols))
				} else {
					classAttributes = append(classAttributes, fmt.Sprintf("@@index(%s)", indexCols))
				}
			}
		}

		if len(classAttributes) > 0 {
			schema.WriteString("\n")
			for _, attr := range classAttributes {
				schema.WriteString(fmt.Sprintf("  %s\n", attr))
			}
		}

		schema.WriteString("}")
	}

	return schema.String()
}

func isColumnPrimaryKey(columnName string, constraints []Constraint) bool {
	for _, constraint := range constraints {
		if strings.EqualFold(constraint.Type, "PRIMARY_KEY") && len(constraint.Columns) == 1 {
			if constraint.Columns[0] == columnName {
				return true
			}
		}
	}
	return false
}

func isColumnUnique(columnName string, constraints []Constraint) bool {
	for _, constraint := range constraints {
		if strings.EqualFold(constraint.Type, "UNIQUE") && len(constraint.Columns) == 1 {
			if constraint.Columns[0] == columnName {
				return true
			}
		}
	}
	return false
}

func isColumnNullable(columnName string, columns []Column) bool {
	for _, column := range columns {
		if column.Name == columnName {
			return column.IsNullable
		}
	}
	return false
}

func getCompositePrimaryKey(constraints []Constraint) []string {
	for _, constraint := range constraints {
		if strings.Contains(strings.ToUpper(constraint.Type), "PRIMARY_KEY") {
			return constraint.Columns
		}
	}
	return []string{}
}

func Contains(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}