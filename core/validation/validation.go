package validation

import (
	"fmt"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast"
	"github.com/rit3sh-x/blaze/core/ast/class"
	"github.com/rit3sh-x/blaze/core/ast/field"
	"github.com/rit3sh-x/blaze/core/constants"
)

type ValidationError struct {
	Type     string
	Message  string
	Location string
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s at %s", ve.Type, ve.Message, ve.Location)
}

type SchemaValidator struct {
	ast    *ast.SchemaAST
	errors []ValidationError
}

func NewSchemaValidator(ast *ast.SchemaAST) *SchemaValidator {
	return &SchemaValidator{
		ast:    ast,
		errors: []ValidationError{},
	}
}

func (sv *SchemaValidator) addError(errorType, message, location string) {
	sv.errors = append(sv.errors, ValidationError{
		Type:     errorType,
		Message:  message,
		Location: location,
	})
}

func (sv *SchemaValidator) ValidateSchema() error {
	sv.errors = []ValidationError{}

	sv.validateUniqueNames()
	sv.validateClassEnumNameConflicts()
	sv.validateFieldTypes()
	sv.validatePrimaryKeys()
	sv.validateAndNameRelationsAndIndexes()
	sv.validateForeignKeyUniqueness()
	sv.validateRelationConsistency()
	sv.validateBackReferences()
	sv.validateCircularDependencies()

	if len(sv.errors) > 0 {
		return sv.formatErrors()
	}

	return nil
}

func (sv *SchemaValidator) validateUniqueNames() {
	classNames := make(map[string]*class.Class)
	for _, cls := range sv.ast.Classes {
		if existing, exists := classNames[cls.Name]; exists {
			sv.addError("DUPLICATE_CLASS",
				fmt.Sprintf("Duplicate class name '%s'", cls.Name),
				fmt.Sprintf("class '%s' (conflicts with class at position %d)", cls.Name, existing.Position))
		} else {
			classNames[cls.Name] = cls
		}
	}

	enumNames := make(map[string]string)
	for enumName := range sv.ast.Enums {
		if _, exists := enumNames[enumName]; exists {
			sv.addError("DUPLICATE_ENUM",
				fmt.Sprintf("Duplicate enum name '%s'", enumName),
				fmt.Sprintf("enum '%s'", enumName))
		} else {
			enumNames[enumName] = enumName
		}
	}

	for _, cls := range sv.ast.Classes {
		fieldNames := make(map[string]*field.Field)
		for _, fld := range cls.Attributes.Fields {
			fieldName := fld.GetName()
			if existing, exists := fieldNames[fieldName]; exists {
				sv.addError("DUPLICATE_FIELD",
					fmt.Sprintf("Duplicate field name '%s' in class '%s'", fieldName, cls.Name),
					fmt.Sprintf("class '%s', field '%s' (conflicts with field at position %d)", cls.Name, fieldName, existing.Position))
			} else {
				fieldNames[fieldName] = fld
			}
		}
	}
}

func (sv *SchemaValidator) validateClassEnumNameConflicts() {
	for _, cls := range sv.ast.Classes {
		if _, exists := sv.ast.Enums[cls.Name]; exists {
			sv.addError("NAME_CONFLICT",
				fmt.Sprintf("Class name '%s' conflicts with enum name", cls.Name),
				fmt.Sprintf("class '%s'", cls.Name))
		}
	}
}

func (sv *SchemaValidator) validateFieldTypes() {
	for _, cls := range sv.ast.Classes {
		for _, fld := range cls.Attributes.Fields {
			baseType := fld.GetBaseType()

			if constants.IsScalarType(baseType) {
				continue
			}

			if _, exists := sv.ast.Enums[baseType]; exists {
				continue
			}

			if sv.ast.GetClassByName(baseType) != nil {
				continue
			}

			sv.addError("INVALID_TYPE",
				fmt.Sprintf("Unknown type '%s' for field '%s'", baseType, fld.GetName()),
				fmt.Sprintf("class '%s', field '%s'", cls.Name, fld.GetName()))
		}
	}
}

func (sv *SchemaValidator) validatePrimaryKeys() {
	for _, cls := range sv.ast.Classes {
		fieldPKCount := 0
		classPKExists := cls.Attributes.HasPrimaryKey()

		for _, fld := range cls.Attributes.Fields {
			if fld.IsPrimaryKey() {
				fieldPKCount++
			}
		}

		if fieldPKCount > 1 {
			sv.addError("MULTIPLE_FIELD_PK",
				fmt.Sprintf("Class '%s' has multiple field-level primary keys", cls.Name),
				fmt.Sprintf("class '%s'", cls.Name))
		}

		if fieldPKCount > 0 && classPKExists {
			sv.addError("CONFLICTING_PK",
				fmt.Sprintf("Class '%s' has both field-level and class-level primary keys", cls.Name),
				fmt.Sprintf("class '%s'", cls.Name))
		}

		if classPKExists {
			pkFields := cls.Attributes.GetPrimaryKeyFields()
			for _, fieldName := range pkFields {
				fld := cls.Attributes.GetFieldByName(fieldName)
				if fld == nil {
					sv.addError("INVALID_PK_FIELD",
						fmt.Sprintf("Primary key references non-existent field '%s'", fieldName),
						fmt.Sprintf("class '%s'", cls.Name))
				} else {
					if fld.IsOptional() {
						sv.addError("OPTIONAL_PK_FIELD",
							fmt.Sprintf("Primary key field '%s' cannot be optional", fieldName),
							fmt.Sprintf("class '%s', field '%s'", cls.Name, fieldName))
					}
					if fld.IsArray() {
						sv.addError("ARRAY_PK_FIELD",
							fmt.Sprintf("Primary key field '%s' cannot be an array", fieldName),
							fmt.Sprintf("class '%s', field '%s'", cls.Name, fieldName))
					}
				}
			}
		}
	}
}

func (sv *SchemaValidator) validateAndNameRelationsAndIndexes() {
	sv.nameRelations()
	sv.nameIndexes()
}

func (sv *SchemaValidator) nameRelations() {
	for _, cls := range sv.ast.Classes {
		for _, fld := range cls.Attributes.Fields {
			if !fld.HasRelation() {
				continue
			}

			relation := fld.AttributeDefinition.GetRelation()
			if relation == nil {
				continue
			}

			if relation.Name == "" {
				relationName := sv.generateRelationName(relation.FromClass, relation.ToClass, relation.From, relation.To)
				relation.Name = relationName
			}
		}
	}
}

func (sv *SchemaValidator) nameIndexes() {
	for _, cls := range sv.ast.Classes {
		for _, directive := range cls.Attributes.Directives {
			switch directive.Name {
			case constants.CLASS_ATTR_INDEX, constants.CLASS_ATTR_TEXT_INDEX:
				if directive.PseudoName == "" {
					values := directive.Value.([]string)
					indexName := sv.generateIndexName(cls.Name, values)
					directive.PseudoName = indexName
				}
			}
		}
	}
}

func (sv *SchemaValidator) generateRelationName(fromClass string, toClass string, from []string, to []string) string {
	var nameParts []string

	nameParts = append(nameParts, fromClass)
	nameParts = append(nameParts, from...)

	nameParts = append(nameParts, toClass)
	nameParts = append(nameParts, to...)

	return "_relation_" + strings.Join(nameParts, "_")
}

func (sv *SchemaValidator) generateIndexName(className string, values []string) string {
	var nameParts []string

	nameParts = append(nameParts, className)
	nameParts = append(nameParts, values...)

	return "_idx_" + strings.Join(nameParts, "_")
}

func (sv *SchemaValidator) validateForeignKeyUniqueness() {
	for _, cls := range sv.ast.Classes {
		for _, fld := range cls.Attributes.Fields {
			if !fld.HasRelation() {
				continue
			}

			relation := fld.AttributeDefinition.GetRelation()
			if relation == nil {
				continue
			}

			referencedFields := relation.To
			referencedClass := relation.ToClass
			if referencedClass == "" {
				continue
			}

			relationFieldMap := make(map[string]bool)

			for _, field := range referencedFields {
				relationFieldMap[field] = false
			}

			targetClass := sv.ast.GetClassByName(referencedClass)
			if targetClass == nil {
				sv.addError("INVALID_RELATION_TARGET",
					fmt.Sprintf("Field '%s' references non-existent class '%s'", fld.GetName(), referencedClass),
					fmt.Sprintf("class '%s', field '%s'", cls.Name, fld.GetName()))
				continue
			}

			for _, directive := range targetClass.Attributes.Directives {
				if directive.Name == constants.CLASS_ATTR_UNIQUE {
					if uniqueFields, ok := directive.Value.([]string); ok {
						allFieldsPresent := true
						for _, uniqueField := range uniqueFields {
							if _, exists := relationFieldMap[uniqueField]; !exists {
								allFieldsPresent = false
								break
							}
						}

						if allFieldsPresent && len(uniqueFields) == len(relationFieldMap) {
							for _, field := range uniqueFields {
								relationFieldMap[field] = true
							}
						}
					}
				}
			}

			for _, directive := range targetClass.Attributes.Directives {
				if directive.Name == constants.CLASS_ATTR_PRIMARY_KEY {
					if pkFields, ok := directive.Value.([]string); ok {
						allFieldsPresent := true
						for _, pkField := range pkFields {
							if _, exists := relationFieldMap[pkField]; !exists {
								allFieldsPresent = false
								break
							}
						}

						if allFieldsPresent && len(pkFields) == len(relationFieldMap) {
							for _, field := range pkFields {
								relationFieldMap[field] = true
							}
						}
					}
				}
			}

			for _, targetField := range targetClass.Attributes.Fields {
				fieldName := targetField.GetName()
				if _, exists := relationFieldMap[fieldName]; exists {
					if len(relationFieldMap) == 1 && (targetField.IsUnique() || targetField.IsPrimaryKey()) {
						relationFieldMap[fieldName] = true
					}
				}
			}

			for _, directive := range targetClass.Attributes.Directives {
				if directive.Name == constants.CLASS_ATTR_INDEX {
					if indexFields, ok := directive.Value.([]string); ok {
						if sv.sliceEqual(referencedFields, indexFields) {
							hasCorrespondingUnique := false

							for _, uniqueDirective := range targetClass.Attributes.Directives {
								if uniqueDirective.Name == constants.CLASS_ATTR_UNIQUE {
									if uniqueFields, ok := uniqueDirective.Value.([]string); ok {
										if sv.sliceEqual(indexFields, uniqueFields) {
											hasCorrespondingUnique = true
											break
										}
									}
								}
							}

							if !hasCorrespondingUnique {
								for _, pkDirective := range targetClass.Attributes.Directives {
									if pkDirective.Name == constants.CLASS_ATTR_PRIMARY_KEY {
										if pkFields, ok := pkDirective.Value.([]string); ok {
											if sv.sliceEqual(indexFields, pkFields) {
												hasCorrespondingUnique = true
												break
											}
										}
									}
								}
							}

							if hasCorrespondingUnique {
								for _, field := range indexFields {
									if _, exists := relationFieldMap[field]; exists {
										relationFieldMap[field] = true
									}
								}
							}
						}
					}
				}
			}

			for _, targetField := range targetClass.Attributes.Fields {
				fieldName := targetField.GetName()
				if _, exists := relationFieldMap[fieldName]; exists {
					if targetField.IsUnique() || targetField.IsPrimaryKey() {
						hasIndex := false
						for _, directive := range targetClass.Attributes.Directives {
							if directive.Name == constants.CLASS_ATTR_INDEX {
								if indexFields, ok := directive.Value.([]string); ok {
									if len(indexFields) == 1 && indexFields[0] == fieldName {
										hasIndex = true
										break
									}
								}
							}
						}

						if hasIndex {
							relationFieldMap[fieldName] = true
						}
					}
				}
			}

			nonUniqueFields := []string{}
			for field, isUnique := range relationFieldMap {
				if !isUnique {
					nonUniqueFields = append(nonUniqueFields, field)
				}
			}

			if len(nonUniqueFields) > 0 {
				sv.addError("NON_UNIQUE_REFERENCE",
					fmt.Sprintf("Foreign key references non-unique fields %v in class '%s' (no corresponding unique constraint found)", nonUniqueFields, referencedClass),
					fmt.Sprintf("class '%s', field '%s'", cls.Name, fld.GetName()))
			}
		}
	}
}

func (sv *SchemaValidator) validateRelationConsistency() {
	relationMap := make(map[string][]string)

	for _, cls := range sv.ast.Classes {
		for _, fld := range cls.Attributes.Fields {
			if fld.HasRelation() {
				targetType := fld.GetBaseType()
				if targetClass := sv.ast.GetClassByName(targetType); targetClass != nil {
					relationMap[cls.Name] = append(relationMap[cls.Name], targetType)
				}
			}
		}
	}

	for className, relatedClasses := range relationMap {
		for _, relatedClass := range relatedClasses {
			if !sv.validateRelationExists(className, relatedClass) {
				sv.addError("INVALID_RELATION",
					fmt.Sprintf("Relation from '%s' to '%s' is not properly defined", className, relatedClass),
					fmt.Sprintf("class '%s'", className))
			}
		}
	}
}

func (sv *SchemaValidator) validateBackReferences() {
	for _, cls := range sv.ast.Classes {
		for _, fld := range cls.Attributes.Fields {
			if !fld.IsBackReference() {
				continue
			}

			targetType := fld.GetBaseType()
			targetClass := sv.ast.GetClassByName(targetType)
			if targetClass == nil {
				continue
			}

			hasForeignKey := false
			for _, targetField := range targetClass.Attributes.Fields {
				if targetField.IsForeignKey() && targetField.GetBaseType() == cls.Name {
					hasForeignKey = true
					break
				}
			}

			if !hasForeignKey {
				sv.addError("MISSING_FOREIGN_KEY",
					fmt.Sprintf("Back reference field '%s' has no corresponding foreign key in class '%s'", fld.GetName(), targetType),
					fmt.Sprintf("class '%s', field '%s'", cls.Name, fld.GetName()))
			}
		}
	}
}

func (sv *SchemaValidator) validateCircularDependencies() {
	for _, cls := range sv.ast.Classes {
		for _, fld := range cls.Attributes.Fields {
			targetType := fld.GetBaseType()
			targetClass := sv.ast.GetClassByName(targetType)
			if targetClass == nil || fld.AttributeDefinition.Relation == nil {
				continue
			}

			hasForeignKey := false
			for _, targetField := range targetClass.Attributes.Fields {
				if targetField.IsForeignKey() && targetField.GetBaseType() == cls.Name && !targetField.IsOptional() && !fld.IsOptional() {
					hasForeignKey = true
					break
				}
			}

			if hasForeignKey {
				sv.addError("CIRCULAR_DEPENDENCY",
					fmt.Sprintf("Circular dependency found between '%s' and '%s'", targetType, cls.Name),
					fmt.Sprintf("class '%s', field '%s'", cls.Name, fld.GetName()))
			}
		}
	}
}

func (sv *SchemaValidator) validateRelationExists(sourceClass, targetClass string) bool {
	source := sv.ast.GetClassByName(sourceClass)
	target := sv.ast.GetClassByName(targetClass)
	return source != nil && target != nil
}

func (sv *SchemaValidator) sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func (sv *SchemaValidator) formatErrors() error {
	var errorMessages []string
	for _, err := range sv.errors {
		errorMessages = append(errorMessages, err.Error())
	}
	return fmt.Errorf("schema validation failed:\n%s", strings.Join(errorMessages, "\n"))
}

func ValidateSchema(ast *ast.SchemaAST) error {
	validator := NewSchemaValidator(ast)
	return validator.ValidateSchema()
}

func ValidateSchemaWithDetails(ast *ast.SchemaAST) ([]ValidationError, error) {
	validator := NewSchemaValidator(ast)
	err := validator.ValidateSchema()
	return validator.errors, err
}