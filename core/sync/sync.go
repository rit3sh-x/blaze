package sync

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rit3sh-x/blaze/core/constants"
	"github.com/rit3sh-x/blaze/core/sync/class"
	"github.com/rit3sh-x/blaze/core/sync/enum"
)

func GetEnums(client *pgxpool.Pool, ctx context.Context) (string, []string, error) {
	var enumData []enum.EnumData
	var enumNames []string

	rows, err := client.Query(ctx, constants.ALL_ENUMS_QUERY)
	if err != nil {
		return "", nil, fmt.Errorf("failed to fetch available enums: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var enumName, enumValue string
		var sortOrder int8

		if err := rows.Scan(&enumName, &enumValue, &sortOrder); err != nil {
			return "", nil, fmt.Errorf("scan enum row error: %v", err)
		}

		enumData = append(enumData, enum.EnumData{
			Name:      enumName,
			Value:     enumValue,
			SortOrder: sortOrder,
		})

		if !class.Contains(enumNames, enumName) {
			enumNames = append(enumNames, enumName)
		}
	}

	if rows.Err() != nil {
		return "", nil, fmt.Errorf("row iteration error: %v", rows.Err())
	}

	if len(enumData) == 0 {
		return "", nil, nil
	}

	enumSchema := enum.GenerateEnumSchema(enumData)
	return enumSchema, enumNames, nil
}

func GetClasses(client *pgxpool.Pool, ctx context.Context, enums []string) (string, error) {
	var classData []class.ClassData
	rows, err := client.Query(ctx, constants.FETCH_AVAILABLE_TABLES)
	if err != nil {
		return "", fmt.Errorf("failed to fetch available tables: %v", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return "", fmt.Errorf("scan table name error: %v", err)
		}
		if tableName == constants.MIGRATION_TABLE_NAME {
			continue
		}
		tableNames = append(tableNames, tableName)
	}

	if len(tableNames) == 0 {
		return "", nil
	}

	for _, tableName := range tableNames {
		var tableData class.ClassData
		tableData.Name = tableName

		columnRows, err := client.Query(ctx, constants.TableTypes(tableName))
		if err != nil {
			return "", fmt.Errorf("failed to fetch columns for table %s: %v", tableName, err)
		}
		for columnRows.Next() {
			var columnName, baseType, isNullable string
			var columnDefault *string
			var ordinalPosition int8
			if err := columnRows.Scan(&columnName, &baseType, &isNullable, &columnDefault, &ordinalPosition); err != nil {
				columnRows.Close()
				return "", fmt.Errorf("scan column row error for table %s: %v", tableName, err)
			}

			isArray := false

			if baseType != "" && baseType[0] == '_' {
				baseType = strings.ReplaceAll(baseType, "_", "")
				isArray = true
			}

			nullable := isNullable == "YES"
			defaultVal := "null"
			if columnDefault != nil {
				defaultVal = *columnDefault
			}
			defaultVal = strings.TrimSpace(strings.ReplaceAll(defaultVal, " ", "_"))
			cleanDataType := strings.ReplaceAll(baseType, " ", "_")

			enumCastRegex := regexp.MustCompile(`^'([^']*)'::.*`)
			if matches := enumCastRegex.FindStringSubmatch(defaultVal); len(matches) > 1 {
				defaultVal = matches[1]
			}

			column := class.Column{
				Name:            columnName,
				DataType:        cleanDataType,
				IsNullable:      nullable,
				ColumnDefault:   defaultVal,
				OrdinalPosition: ordinalPosition,
				IsArray:         isArray,
			}
			tableData.Columns = append(tableData.Columns, column)
		}
		if err := columnRows.Err(); err != nil {
			columnRows.Close()
			return "", fmt.Errorf("row iteration error for table %s: %v", tableName, err)
		}
		columnRows.Close()

		constraintRows, err := client.Query(ctx, constants.TableConstraints(tableName))
		if err != nil {
			return "", fmt.Errorf("failed to fetch constraints for table %s: %v", tableName, err)
		}
		constraintMap := make(map[string]*class.Constraint)
		for constraintRows.Next() {
			var constraintType, columnName, constraintName string
			if err := constraintRows.Scan(&constraintType, &columnName, &constraintName); err != nil {
				constraintRows.Close()
				return "", fmt.Errorf("scan constraint row error for table %s: %v", tableName, err)
			}

			cleanConstraintType := strings.ReplaceAll(constraintType, " ", "_")

			if constraint, exists := constraintMap[constraintName]; exists {
				constraint.Columns = append(constraint.Columns, columnName)
			} else {
				constraintMap[constraintName] = &class.Constraint{
					Name:    constraintName,
					Type:    cleanConstraintType,
					Columns: []string{columnName},
				}
			}
		}
		if err := constraintRows.Err(); err != nil {
			constraintRows.Close()
			return "", fmt.Errorf("row iteration error for table %s: %v", tableName, err)
		}
		constraintRows.Close()
		for _, constraint := range constraintMap {
			tableData.Constraints = append(tableData.Constraints, *constraint)
		}

		indexRows, err := client.Query(ctx, constants.TableIndexes(tableName))
		if err != nil {
			return "", fmt.Errorf("failed to fetch indexes for table %s: %v", tableName, err)
		}
		for indexRows.Next() {
			var indexName, columns string
			var isUnique, isPrimary bool
			if err := indexRows.Scan(&indexName, &isUnique, &isPrimary, &columns); err != nil {
				indexRows.Close()
				return "", fmt.Errorf("scan index row error for table %s: %v", tableName, err)
			}

			columnRegex := regexp.MustCompile(`\s*,\s*`)
			columnFields := columnRegex.Split(strings.TrimSpace(columns), -1)
			var cleanFields []string
			for _, field := range columnFields {
				if field = strings.TrimSpace(field); field != "" {
					cleanFields = append(cleanFields, field)
				}
			}

			index := class.Index{
				Name:      indexName,
				IsUnique:  isUnique,
				IsPrimary: isPrimary,
				Fields:    cleanFields,
			}
			tableData.Indexes = append(tableData.Indexes, index)
		}
		if err := indexRows.Err(); err != nil {
			indexRows.Close()
			return "", fmt.Errorf("row iteration error for table %s: %v", tableName, err)
		}
		indexRows.Close()

		relationRows, err := client.Query(ctx, constants.TableRelations(tableName))
		if err != nil {
			return "", fmt.Errorf("failed to fetch relations for table %s: %v", tableName, err)
		}
		relationMap := make(map[string]*class.Relation)
		for relationRows.Next() {
			var fkColumn, referencedTable, referencedColumn, updateRule, deleteRule, constraintName string
			if err := relationRows.Scan(&fkColumn, &referencedTable, &referencedColumn, &updateRule, &deleteRule, &constraintName); err != nil {
				relationRows.Close()
				return "", fmt.Errorf("scan relation row error for table %s: %v", tableName, err)
			}

			if relation, exists := relationMap[constraintName]; exists {
				relation.FkColumns = append(relation.FkColumns, fkColumn)
				relation.ReferencedColumns = append(relation.ReferencedColumns, referencedColumn)
			} else {
				relationMap[constraintName] = &class.Relation{
					Name:              constraintName,
					FkColumns:         []string{fkColumn},
					ReferencedTable:   referencedTable,
					ReferencedColumns: []string{referencedColumn},
					UpdateRule:        updateRule,
					DeleteRule:        deleteRule,
				}
			}
		}
		if err := relationRows.Err(); err != nil {
			relationRows.Close()
			return "", fmt.Errorf("row iteration error for table %s: %v", tableName, err)
		}
		relationRows.Close()
		for _, relation := range relationMap {
			tableData.Relations = append(tableData.Relations, *relation)
		}

		classData = append(classData, tableData)
	}

	classSchema := class.GenerateClassSchema(classData, enums)
	return classSchema, nil
}