package migration

import (
	"fmt"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast/class"
	"github.com/rit3sh-x/blaze/core/ast/field"
)

func (me *MigrationEngine) generateTableAlterationSQL(oldClass, newClass *class.Class) ([]MigrationStatement, error) {
	var statements []MigrationStatement
	tableName := applyQuotes(newClass.Name)

	oldFields := make(map[string]*field.Field)
	for _, field := range oldClass.Attributes.Fields {
		oldFields[field.GetName()] = field
	}

	for _, newField := range newClass.Attributes.Fields {
		if _, exists := oldFields[newField.GetName()]; !exists {
			columnDef, possible := me.generateColumnDefinition(newField, newClass)
			if !possible {
				continue
			}
			statements = append(statements, MigrationStatement{
				SQL:      fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", tableName, columnDef),
				Type:     "column_add",
				Priority: 7,
			})
		}
	}

	newFields := make(map[string]*field.Field)
	for _, field := range newClass.Attributes.Fields {
		newFields[field.GetName()] = field
	}

	for _, oldField := range oldClass.Attributes.Fields {
		if _, exists := newFields[oldField.GetName()]; !exists {
			columnName := applyQuotes(oldField.GetName())
			statements = append(statements, MigrationStatement{
				SQL:      fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s", tableName, columnName),
				Type:     "column_drop",
				Priority: 8,
			})
		}
	}

	for _, newField := range newClass.Attributes.Fields {
		if oldField, exists := oldFields[newField.GetName()]; exists {
			alterStatements := me.generateColumnAlterationSQL(tableName, oldField, newField, oldClass, newClass)
			statements = append(statements, alterStatements...)
		}
	}

	return statements, nil
}

func (me *MigrationEngine) generateColumnAlterationSQL(tableName string, oldField, newField *field.Field, oldClass, newClass *class.Class) []MigrationStatement {
	var statements []MigrationStatement
	columnName := applyQuotes(newField.GetName())

	oldType, _ := me.mapToPGType(oldField, oldClass)
	newType, _ := me.mapToPGType(newField, newClass)

	if oldType != newType || oldField.IsArray() != newField.IsArray() {
		targetType := newType
		if newField.IsArray() {
			targetType += "[]"
		}
		statements = append(statements, MigrationStatement{
			SQL:      fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", tableName, columnName, targetType),
			Type:     "column_alter_type",
			Priority: 9,
		})
	}

	if oldField.IsOptional() != newField.IsOptional() {
		if newField.IsOptional() {
			statements = append(statements, MigrationStatement{
				SQL:      fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL", tableName, columnName),
				Type:     "column_alter_null",
				Priority: 10,
			})
		} else {
			statements = append(statements, MigrationStatement{
				SQL:      fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL", tableName, columnName),
				Type:     "column_alter_not_null",
				Priority: 10,
			})
		}
	}

	oldHasDefault := oldField.HasDefault()
	newHasDefault := newField.HasDefault()

	if oldHasDefault && !newHasDefault {
		statements = append(statements, MigrationStatement{
			SQL:      fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT", tableName, columnName),
			Type:     "column_drop_default",
			Priority: 11,
		})
	} else if newHasDefault {
		defaultValue := me.generateDefaultValue(newField)
		statements = append(statements, MigrationStatement{
			SQL:      fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s", tableName, columnName, defaultValue),
			Type:     "column_set_default",
			Priority: 11,
		})
	}

	return statements
}

func (me *MigrationEngine) generateTableMigrations() ([]MigrationStatement, error) {
	var statements []MigrationStatement

	for _, oldClass := range me.fromSchema.Classes {
		if me.toSchema.GetClassByName(oldClass.Name) == nil {
			statements = append(statements, MigrationStatement{
				SQL:      fmt.Sprintf(`DROP TABLE IF EXISTS "%s" CASCADE`, oldClass.Name),
				Type:     "table_drop",
				Priority: 5,
			})
		}
	}

	for _, newClass := range me.toSchema.Classes {
		if me.fromSchema.GetClassByName(newClass.Name) == nil {
			sql, err := me.generateCreateTableSQL(newClass)
			if err != nil {
				return nil, fmt.Errorf("failed to generate CREATE TABLE for %s: %v", newClass.Name, err)
			}
			statements = append(statements, MigrationStatement{
				SQL:      sql,
				Type:     "table_create",
				Priority: 6,
			})
		}
	}

	for _, newClass := range me.toSchema.Classes {
		if oldClass := me.fromSchema.GetClassByName(newClass.Name); oldClass != nil {
			alterStatements, err := me.generateTableAlterationSQL(oldClass, newClass)
			if err != nil {
				return nil, fmt.Errorf("failed to generate table alterations for %s: %v", newClass.Name, err)
			}
			statements = append(statements, alterStatements...)
		}
	}

	return statements, nil
}

func (me *MigrationEngine) generateCreateTableSQL(cls *class.Class) (string, error) {
	tableName := applyQuotes(cls.Name)
	var columns []string
	var constraints []string

	for _, field := range cls.Attributes.Fields {
		columnDef, possible := me.generateColumnDefinition(field, cls)
		if !possible {
			continue
		}
		columns = append(columns, columnDef)
	}

	pkFields := cls.GetPrimaryKeyFields()
	if len(pkFields) > 0 {
        pkColumns := make([]string, len(pkFields))
        for i, field := range pkFields {
            pkColumns[i] = applyQuotes(field)
        }
        constraints = append(constraints, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pkColumns, ", ")))
    }

	if cls.Attributes.HasUnique() {
		if uniqueDirective := cls.Attributes.GetUniqueDirective(); uniqueDirective != nil {
			if fields, err := uniqueDirective.GetFields(); err == nil {
				uniqueColumns := make([]string, len(fields))
				for i, field := range fields {
                    uniqueColumns[i] = applyQuotes(field)
                }
				constraints = append(constraints, fmt.Sprintf("UNIQUE (%s)", strings.Join(uniqueColumns, ", ")))
			}
		}
	}

	if cls.Attributes.HasCheck() {
		if checkDirective := cls.Attributes.GetCheckDirective(); checkDirective != nil {
			if constraint, err := checkDirective.GetConstraint(); err == nil {
				constraints = append(constraints, fmt.Sprintf("CHECK (%s)", constraint))
			}
		}
	}

	var parts []string
	parts = append(parts, columns...)
	parts = append(parts, constraints...)

	return fmt.Sprintf("CREATE TABLE %s (\n  %s\n)", tableName, strings.Join(parts, ",\n  ")), nil
}