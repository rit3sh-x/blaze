package migration

import (
	"fmt"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast/enum"
)

func (me *MigrationEngine) generateEnumMigrations() ([]MigrationStatement, error) {
	var statements []MigrationStatement

	for enumName := range me.fromSchema.Enums {
		if _, exists := me.toSchema.Enums[enumName]; !exists {
			statements = append(statements, MigrationStatement{
				SQL:      fmt.Sprintf(`DROP TYPE IF EXISTS "%s" CASCADE`, enumName),
				Type:     "enum_drop",
				Priority: 2,
			})
		}
	}

	for enumName, enumDef := range me.toSchema.Enums {
		if _, exists := me.fromSchema.Enums[enumName]; !exists {
			sql := me.generateCreateEnumSQL(enumDef)
			statements = append(statements, MigrationStatement{
				SQL:      sql,
				Type:     "enum_create",
				Priority: 3,
			})
		}
	}

	for enumName, newEnum := range me.toSchema.Enums {
		if oldEnum, exists := me.fromSchema.Enums[enumName]; exists {
			enumStatements := me.generateEnumAlterationSQL(enumName, oldEnum, newEnum)
			statements = append(statements, enumStatements...)
		}
	}

	return statements, nil
}

func (me *MigrationEngine) generateCreateEnumSQL(enumDef *enum.Enum) string {
	var values []string
	for _, value := range enumDef.Values {
		values = append(values, fmt.Sprintf("'%s'", value.Name))
	}
	return fmt.Sprintf(`CREATE TYPE "%s" AS ENUM (%s)`, enumDef.Name, strings.Join(values, ", "))
}

func (me *MigrationEngine) generateEnumAlterationSQL(enumName string, oldEnum, newEnum *enum.Enum) []MigrationStatement {
	var statements []MigrationStatement

	oldValues := make(map[string]bool)
	for _, value := range oldEnum.Values {
		oldValues[value.Name] = true
	}

	for _, value := range newEnum.Values {
		if !oldValues[value.Name] {
			statements = append(statements, MigrationStatement{
				SQL:      fmt.Sprintf(`ALTER TYPE "%s" ADD VALUE '%s'`, enumName, value.Name),
				Type:     "enum_alter",
				Priority: 4,
			})
		}
	}

	return statements
}