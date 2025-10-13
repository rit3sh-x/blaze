package migration

import (
	"fmt"
	"strings"
)

func (me *MigrationEngine) generateConstraintMigrations() ([]MigrationStatement, error) {
	var statements []MigrationStatement

	for _, newClass := range me.toSchema.Classes {
		for _, field := range newClass.Attributes.Fields {
			if field.HasRelation() {
				relation := field.AttributeDefinition.Relation
				if relation != nil {
					fkName := fmt.Sprintf("fk_%s_%s", strings.ToLower(newClass.Name), strings.ToLower(field.GetName()))
					referencedTable := applyQuotes(relation.ToClass)

					sourceColumns := make([]string, len(relation.From))
					for i, col := range relation.From {
                        sourceColumns[i] = applyQuotes(col)
                    }

					targetColumns := make([]string, len(relation.To))
					for i, col := range relation.To {
                        targetColumns[i] = applyQuotes(col)
                    }

					var constraintParts []string
					constraintParts = append(constraintParts, fmt.Sprintf("FOREIGN KEY (%s)", strings.Join(sourceColumns, ", ")))
					constraintParts = append(constraintParts, fmt.Sprintf("REFERENCES %s (%s)", referencedTable, strings.Join(targetColumns, ", ")))

					if relation.HasOnDelete() {
						constraintParts = append(constraintParts, fmt.Sprintf("ON DELETE %s", me.mapConstraintAction(relation.OnDelete)))
					}

					if relation.HasOnUpdate() {
						constraintParts = append(constraintParts, fmt.Sprintf("ON UPDATE %s", me.mapConstraintAction(relation.OnUpdate)))
					}

					statements = append(statements, MigrationStatement{
						SQL:      fmt.Sprintf(`ALTER TABLE "%s" ADD CONSTRAINT %s %s`, newClass.Name, fkName, strings.Join(constraintParts, " ")),
						Type:     "constraint_add",
						Priority: 14,
					})

					shouldCreateIndex := true

					if len(sourceColumns) == 1 && field.IsUnique() {
						shouldCreateIndex = false
					}

					if shouldCreateIndex && newClass.Attributes.HasUnique() {
						if uniqueDirective := newClass.Attributes.GetUniqueDirective(); uniqueDirective != nil {
							if uniqueFields, err := uniqueDirective.GetFields(); err == nil {
								sourceSet := make(map[string]bool)
								for _, col := range sourceColumns {
									sourceSet[col] = true
								}

								uniqueSet := make(map[string]bool)
								for _, col := range uniqueFields {
									uniqueSet[col] = true
								}

								allSourceInUnique := true
								for _, sourceCol := range sourceColumns {
									if !uniqueSet[sourceCol] {
										allSourceInUnique = false
										break
									}
								}

								if allSourceInUnique {
									shouldCreateIndex = false
								}
							}
						}
					}

					if shouldCreateIndex {
						pkFields := newClass.GetPrimaryKeyFields()
						pkSet := make(map[string]bool)
						for _, pkField := range pkFields {
							pkSet[pkField] = true
						}

						allInPK := len(sourceColumns) > 0
						for _, sourceCol := range sourceColumns {
							if !pkSet[sourceCol] {
								allInPK = false
								break
							}
						}
						if allInPK {
							shouldCreateIndex = false
						}
					}

					if shouldCreateIndex {
                        indexColumns := make([]string, len(relation.From))
                        for i, col := range relation.From {
                            indexColumns[i] = applyQuotes(col)
                        }
                        indexName := fmt.Sprintf("idx_%s_%s", strings.ToLower(newClass.Name), strings.ToLower(strings.Join(relation.From, "_")))
                        statements = append(statements, MigrationStatement{
                            SQL:      fmt.Sprintf(`CREATE INDEX %s ON "%s" (%s)`, indexName, newClass.Name, strings.Join(indexColumns, ", ")),
                            Type:     "fk_index_create",
                            Priority: 14,
                        })
                    }
				}
			}
		}
	}

	return statements, nil
}