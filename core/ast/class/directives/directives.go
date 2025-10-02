package directives

import (
	"fmt"
	"strings"

	"github.com/rit3sh-x/blaze/core/constants"
)

type ClassDirective struct {
	Name       string
	PseudoName string
	Value      interface{}
}

type DirectiveValidator struct{}

func NewDirectiveValidator() *DirectiveValidator {
	return &DirectiveValidator{}
}

func (av *DirectiveValidator) ValidateClassDirective(attr *ClassDirective) error {
	if attr == nil {
		return fmt.Errorf("class directive cannot be nil")
	}

	switch attr.Name {
	case constants.CLASS_ATTR_PRIMARY_KEY:
		return av.validateClassPrimaryKeyDirective(attr)
	case constants.CLASS_ATTR_UNIQUE:
		return av.validateClassUniqueDirective(attr)
	case constants.CLASS_ATTR_INDEX:
		return av.validateClassIndexDirective(attr)
	case constants.CLASS_ATTR_TEXT_INDEX:
		return av.validateClassTextIndexDirective(attr)
	case constants.CLASS_ATTR_CHECK:
		return av.validateClassCheckDirective(attr)
	default:
		return fmt.Errorf("unknown class directive '@%s'", attr.Name)
	}
}

func (av *DirectiveValidator) validateClassPrimaryKeyDirective(attr *ClassDirective) error {
	if attr.Value == nil {
		return fmt.Errorf("@primaryKey directive requires field array parameter")
	}

	fields, ok := attr.Value.([]string)
	if !ok {
		return fmt.Errorf("@primaryKey directive value must be an array of field names")
	}

	if len(fields) == 0 {
		return fmt.Errorf("@primaryKey directive requires at least one field")
	}

	return nil
}

func (av *DirectiveValidator) validateClassUniqueDirective(attr *ClassDirective) error {
	if attr.Value == nil {
		return fmt.Errorf("@unique directive requires field array parameter")
	}

	fields, ok := attr.Value.([]string)
	if !ok {
		return fmt.Errorf("@unique directive value must be an array of field names")
	}

	if len(fields) == 0 {
		return fmt.Errorf("@unique directive requires at least one field")
	}

	return nil
}

func (av *DirectiveValidator) validateClassIndexDirective(attr *ClassDirective) error {
	if attr.Value == nil {
		return fmt.Errorf("@index directive requires field array parameter")
	}

	fields, ok := attr.Value.([]string)
	if !ok {
		return fmt.Errorf("@index directive value must be an array of field names")
	}

	if len(fields) == 0 {
		return fmt.Errorf("@index directive requires at least one field")
	}

	return nil
}

func (av *DirectiveValidator) validateClassTextIndexDirective(attr *ClassDirective) error {
	if attr.Value == nil {
		return fmt.Errorf("@textIndex directive requires field array parameter")
	}

	fields, ok := attr.Value.([]string)
	if !ok {
		return fmt.Errorf("@textIndex directive value must be an array of field names")
	}

	if len(fields) == 0 {
		return fmt.Errorf("@textIndex directive requires at least one field")
	}

	return nil
}

func (av *DirectiveValidator) validateClassCheckDirective(attr *ClassDirective) error {
	if attr.Value == nil {
		return fmt.Errorf("@check directive requires a constraint expression")
	}

	constraint, ok := attr.Value.(string)
	if !ok {
		return fmt.Errorf("@check directive value must be a string expression")
	}

	constraint = strings.TrimSpace(constraint)
	if len(constraint) == 0 {
		return fmt.Errorf("@check directive requires a non-empty constraint expression")
	}

	return nil
}

func (av *DirectiveValidator) ValidateMultipleClassDirectives(attrs []*ClassDirective) error {
	if len(attrs) == 0 {
		return nil
	}

	directiveCount := make(map[string]int)

	for _, attr := range attrs {
		if attr == nil {
			continue
		}

		directiveCount[attr.Name]++

		if directiveCount[attr.Name] > 1 {
			return fmt.Errorf("duplicate class directive '@%s' found", attr.Name)
		}

		if err := av.ValidateClassDirective(attr); err != nil {
			return err
		}
	}

	return nil
}

func GetClassDirectiveByName(attrs []*ClassDirective, name string) *ClassDirective {
	for _, attr := range attrs {
		if attr != nil && attr.Name == name {
			return attr
		}
	}
	return nil
}

func HasClassDirective(attrs []*ClassDirective, name string) bool {
	return GetClassDirectiveByName(attrs, name) != nil
}

func GetClassPrimaryKeyDirective(attrs []*ClassDirective) *ClassDirective {
	return GetClassDirectiveByName(attrs, constants.CLASS_ATTR_PRIMARY_KEY)
}

func GetClassUniqueDirective(attrs []*ClassDirective) *ClassDirective {
	return GetClassDirectiveByName(attrs, constants.CLASS_ATTR_UNIQUE)
}

func GetClassIndexDirective(attrs []*ClassDirective) *ClassDirective {
	return GetClassDirectiveByName(attrs, constants.CLASS_ATTR_INDEX)
}

func GetClassTextIndexDirective(attrs []*ClassDirective) *ClassDirective {
	return GetClassDirectiveByName(attrs, constants.CLASS_ATTR_TEXT_INDEX)
}

func GetClassCheckDirective(attrs []*ClassDirective) *ClassDirective {
	return GetClassDirectiveByName(attrs, constants.CLASS_ATTR_CHECK)
}

func HasClassPrimaryKey(attrs []*ClassDirective) bool {
	return HasClassDirective(attrs, constants.CLASS_ATTR_PRIMARY_KEY)
}

func HasClassUnique(attrs []*ClassDirective) bool {
	return HasClassDirective(attrs, constants.CLASS_ATTR_UNIQUE)
}

func HasClassIndex(attrs []*ClassDirective) bool {
	return HasClassDirective(attrs, constants.CLASS_ATTR_INDEX)
}

func HasClassTextIndex(attrs []*ClassDirective) bool {
	return HasClassDirective(attrs, constants.CLASS_ATTR_TEXT_INDEX)
}

func HasClassCheck(attrs []*ClassDirective) bool {
	return HasClassDirective(attrs, constants.CLASS_ATTR_CHECK)
}

func (cd *ClassDirective) String() string {
	return fmt.Sprintf("@%s", cd.Name)
}

func (cd *ClassDirective) IsParameterized() bool {
	return cd.Value != nil
}

func (cd *ClassDirective) GetFields() ([]string, error) {
	if cd.Value == nil {
		return nil, fmt.Errorf("directive @%s has no value", cd.Name)
	}

	fields, ok := cd.Value.([]string)
	if !ok {
		return nil, fmt.Errorf("directive @%s value is not a string array", cd.Name)
	}

	return fields, nil
}

func (cd *ClassDirective) GetConstraint() (string, error) {
	if cd.Value == nil {
		return "", fmt.Errorf("directive @%s has no value", cd.Name)
	}

	constraint, ok := cd.Value.(string)
	if !ok {
		return "", fmt.Errorf("directive @%s value is not a string", cd.Name)
	}

	return constraint, nil
}
