package directives

import (
	"fmt"

	"github.com/rit3sh-x/blaze/core/constants"
)

type FieldDirective struct {
	Name  string
	Value interface{}
}

type DirectiveValidator struct{}

func NewDirectiveValidator() *DirectiveValidator {
	return &DirectiveValidator{}
}

func (av *DirectiveValidator) ValidateDirective(attr *FieldDirective, fieldType string, isOptional bool, isArray bool) error {
	if attr == nil {
		return fmt.Errorf("directive cannot be nil")
	}

	switch attr.Name {
	case constants.FIELD_ATTR_PRIMARY_KEY:
		return av.validatePrimaryKeyDirective(attr, isOptional, isArray)
	case constants.FIELD_ATTR_UNIQUE:
		return av.validateUniqueDirective(attr, isArray)
	case constants.FIELD_ATTR_UPDATED_AT:
		return av.validateUpdatedAtDirective(attr, fieldType, isArray)
	default:
		return fmt.Errorf("unknown field directive '@%s'", attr.Name)
	}
}

func (av *DirectiveValidator) validatePrimaryKeyDirective(attr *FieldDirective, isOptional bool, isArray bool) error {
	if isOptional {
		return fmt.Errorf("@primarykey field cannot be optional")
	}
	if isArray {
		return fmt.Errorf("@primarykey field cannot be an array")
	}
	if attr.Value != nil {
		return fmt.Errorf("@primarykey directive does not accept parameters")
	}
	return nil
}

func (av *DirectiveValidator) validateUniqueDirective(attr *FieldDirective, isArray bool) error {
	if isArray {
		return fmt.Errorf("@unique field cannot be an array")
	}
	if attr.Value != nil {
		return fmt.Errorf("@unique directive does not accept parameters")
	}
	return nil
}

func (av *DirectiveValidator) validateUpdatedAtDirective(attr *FieldDirective, fieldType string, isArray bool) error {
	if fieldType != string(constants.TIMESTAMP) && fieldType != string(constants.DATE) {
		return fmt.Errorf("@updatedat can only be used on timestamp or date fields, got %s", fieldType)
	}
	if isArray {
		return fmt.Errorf("@updatedat cannot be used on array fields")
	}
	if attr.Value != nil {
		return fmt.Errorf("@updatedat directive does not accept parameters")
	}
	return nil
}

func (av *DirectiveValidator) ValidateMultipleDirectives(attrs []*FieldDirective, fieldType string, isOptional bool, isArray bool) error {
	if len(attrs) == 0 {
		return nil
	}

	DirectiveCount := make(map[string]int)
	var hasPrimaryKey, hasUnique bool

	for _, attr := range attrs {
		if attr == nil {
			continue
		}

		DirectiveCount[attr.Name]++

		if DirectiveCount[attr.Name] > 1 {
			return fmt.Errorf("duplicate directive '@%s' found", attr.Name)
		}

		if err := av.ValidateDirective(attr, fieldType, isOptional, isArray); err != nil {
			return err
		}

		switch attr.Name {
		case constants.FIELD_ATTR_PRIMARY_KEY:
			hasPrimaryKey = true
		case constants.FIELD_ATTR_UNIQUE:
			hasUnique = true
		}
	}

	if hasPrimaryKey && hasUnique {
		return fmt.Errorf("@primarykey and @unique cannot be used together (primary key is inherently unique)")
	}

	return nil
}

func GetDirectiveByName(attrs []*FieldDirective, name string) *FieldDirective {
	for _, attr := range attrs {
		if attr != nil && attr.Name == name {
			return attr
		}
	}
	return nil
}

func HasDirective(attrs []*FieldDirective, name string) bool {
	return GetDirectiveByName(attrs, name) != nil
}

func (fa *FieldDirective) String() string {
	return fmt.Sprintf("@%s", fa.Name)
}

func (fa *FieldDirective) IsParameterized() bool {
	return false
}

func (fa *FieldDirective) IsMutuallyExclusive(other *FieldDirective) bool {
	if fa.Name == constants.FIELD_ATTR_PRIMARY_KEY && other.Name == constants.FIELD_ATTR_UNIQUE {
		return true
	}
	if fa.Name == constants.FIELD_ATTR_UNIQUE && other.Name == constants.FIELD_ATTR_PRIMARY_KEY {
		return true
	}
	return false
}

func (av *DirectiveValidator) ValidateDirectiveCompatibility(attr1, attr2 *FieldDirective) error {
	if attr1 == nil || attr2 == nil {
		return nil
	}

	if attr1.IsMutuallyExclusive(attr2) {
		return fmt.Errorf("directives '@%s' and '@%s' are mutually exclusive", attr1.Name, attr2.Name)
	}

	return nil
}

func GetPrimaryKeyDirective(attrs []*FieldDirective) *FieldDirective {
	return GetDirectiveByName(attrs, constants.FIELD_ATTR_PRIMARY_KEY)
}

func GetUniqueDirective(attrs []*FieldDirective) *FieldDirective {
	return GetDirectiveByName(attrs, constants.FIELD_ATTR_UNIQUE)
}

func GetUpdatedAtDirective(attrs []*FieldDirective) *FieldDirective {
	return GetDirectiveByName(attrs, constants.FIELD_ATTR_UPDATED_AT)
}

func HasPrimaryKey(attrs []*FieldDirective) bool {
	return HasDirective(attrs, constants.FIELD_ATTR_PRIMARY_KEY)
}

func HasUnique(attrs []*FieldDirective) bool {
	return HasDirective(attrs, constants.FIELD_ATTR_UNIQUE)
}

func HasUpdatedAt(attrs []*FieldDirective) bool {
	return HasDirective(attrs, constants.FIELD_ATTR_UPDATED_AT)
}