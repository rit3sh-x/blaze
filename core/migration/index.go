package migration

import (
	"fmt"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast/class"
)

func (me *MigrationEngine) generateIndexMigrations() ([]MigrationStatement, error) {
	var statements []MigrationStatement

	for _, oldClass := range me.fromSchema.Classes {
		if newClass := me.toSchema.GetClassByName(oldClass.Name); newClass != nil {
			statements = append(statements, me.generateIndexDropStatements(oldClass, newClass)...)
		}
	}

	for _, newClass := range me.toSchema.Classes {
		if oldClass := me.fromSchema.GetClassByName(newClass.Name); oldClass != nil {
			statements = append(statements, me.generateIndexCreateStatements(oldClass, newClass)...)
		} else {
			statements = append(statements, me.generateIndexCreateStatements(nil, newClass)...)
		}
	}

	return statements, nil
}

func (me *MigrationEngine) generateIndexDropStatements(oldClass, newClass *class.Class) []MigrationStatement {
	var statements []MigrationStatement

	if oldClass.Attributes.HasIndex() && (newClass == nil || !newClass.Attributes.HasIndex()) {
		if indexDirective := oldClass.Attributes.GetIndexDirective(); indexDirective != nil {
			if fields, err := indexDirective.GetFields(); err == nil {
				indexColumns := make([]string, len(fields))
				copy(indexColumns, fields)
				indexName := fmt.Sprintf("idx_%s_%s_index",
					strings.ToLower(oldClass.Name),
					strings.ToLower(strings.Join(indexColumns, "_")),
				)
				statements = append(statements, MigrationStatement{
					SQL:      fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName),
					Type:     "index_drop",
					Priority: 12,
				})
			}
		}
	}

	if oldClass.Attributes.HasTextIndex() && (newClass == nil || !newClass.Attributes.HasTextIndex()) {
		if textIndexDirective := oldClass.Attributes.GetTextIndexDirective(); textIndexDirective != nil {
			if fields, err := textIndexDirective.GetFields(); err == nil {
				indexColumns := make([]string, len(fields))
				copy(indexColumns, fields)
				indexName := fmt.Sprintf("idx_%s_%s_text_index",
					strings.ToLower(oldClass.Name),
					strings.ToLower(strings.Join(indexColumns, "_")),
				)
				statements = append(statements, MigrationStatement{
					SQL:      fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName),
					Type:     "index_drop",
					Priority: 12,
				})
			}
		}
	}

	return statements
}

func (me *MigrationEngine) generateIndexCreateStatements(oldClass, newClass *class.Class) []MigrationStatement {
	var statements []MigrationStatement

	needsIndex := newClass.Attributes.HasIndex() && (oldClass == nil || !oldClass.Attributes.HasIndex())
	needsTextIndex := newClass.Attributes.HasTextIndex() && (oldClass == nil || !oldClass.Attributes.HasTextIndex())

	if needsIndex {
		if indexDirective := newClass.Attributes.GetIndexDirective(); indexDirective != nil {
			if fields, err := indexDirective.GetFields(); err == nil {
				indexColumns := make([]string, len(fields))
				for i, field := range fields {
                    indexColumns[i] = applyQuotes(field)
                }
				indexName := fmt.Sprintf("idx_%s_%s_index",
					strings.ToLower(newClass.Name),
					strings.ToLower(strings.Join(indexColumns, "_")),
				)
				statements = append(statements, MigrationStatement{
					SQL:      fmt.Sprintf(`CREATE INDEX %s ON "%s" (%s)`, indexName, newClass.Name, strings.Join(indexColumns, ", ")),
					Type:     "index_create",
					Priority: 13,
				})
			}
		}
	}

	if needsTextIndex {
		if textIndexDirective := newClass.Attributes.GetTextIndexDirective(); textIndexDirective != nil {
			if fields, err := textIndexDirective.GetFields(); err == nil {
				indexColumns := make([]string, len(fields))
				for i, field := range fields {
                    indexColumns[i] = applyQuotes(field)
                }
				indexName := fmt.Sprintf("idx_%s_%s_text_index",
					strings.ToLower(newClass.Name),
					strings.ToLower(strings.Join(indexColumns, "_")),
				)
				statements = append(statements, MigrationStatement{
					SQL: fmt.Sprintf(
						`CREATE INDEX %s ON "%s" USING gin ((%s) gin_trgm_ops)`,
						indexName,
						newClass.Name,
						strings.Join(indexColumns, " || ' ' || "),
					),
					Type:     "text_index_create",
					Priority: 13,
				})
			}
		}
	}

	return statements
}