package attributes

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast/class/directives"
	"github.com/rit3sh-x/blaze/core/ast/enum"
	"github.com/rit3sh-x/blaze/core/ast/field"
	"github.com/rit3sh-x/blaze/core/constants"
)

type ClassAttributes struct {
	Fields     []*field.Field
	Directives []*directives.ClassDirective
}

type AttributeParser struct {
	fieldValidator     *field.FieldValidator
	directiveValidator *directives.DirectiveValidator
	classNamePattern   *regexp.Regexp
	fieldPattern       *regexp.Regexp
}

func NewAttributeParser(enums map[string]*enum.Enum) *AttributeParser {
	return &AttributeParser{
		fieldValidator:     field.NewFieldValidator(enums),
		directiveValidator: directives.NewDirectiveValidator(),
		classNamePattern:   regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`),
		fieldPattern:       regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s+(.+)$`),
	}
}

func (ap *AttributeParser) ParseClassContent(classContent string, className string) (*ClassAttributes, error) {
	if strings.TrimSpace(classContent) == "" {
		return nil, fmt.Errorf("class content cannot be empty")
	}

	if !ap.classNamePattern.MatchString(className) {
		return nil, fmt.Errorf("invalid class name '%s'", className)
	}

	lines := strings.Split(classContent, "\n")
	attributes := &ClassAttributes{
		Fields:     []*field.Field{},
		Directives: []*directives.ClassDirective{},
	}

	fieldPosition := 0

	for i, line := range lines {

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "@@") {
			if err := ap.parseClassDirectiveOrRelation(line, attributes); err != nil {
				return nil, fmt.Errorf("error parsing line %d: %v", i+1, err)
			}
		} else {
			field, err := ap.parseField(line, className, fieldPosition)
			if err != nil {
				return nil, fmt.Errorf("error parsing field at line %d: %v", i+1, err)
			}
			attributes.Fields = append(attributes.Fields, field)
			fieldPosition++
		}
	}

	if err := ap.validateClassAttributes(attributes); err != nil {
		return nil, fmt.Errorf("validation failed: %v", err)
	}

	return attributes, nil
}

func (ap *AttributeParser) parseField(line, className string, position int) (*field.Field, error) {
	matches := ap.fieldPattern.FindStringSubmatch(line)
	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid field syntax: %s", line)
	}
	return ap.fieldValidator.ParseFieldFromString(matches[1], matches[2], className, position)
}

func (ap *AttributeParser) parseClassDirectiveOrRelation(line string, attributes *ClassAttributes) error {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "@@") {
		return fmt.Errorf("class directive must start with @@")
	}

	line = line[2:]

	var directiveName string
	var params string

	if strings.Contains(line, "(") {
		parenIndex := strings.Index(line, "(")
		if !strings.HasSuffix(line, ")") {
			return fmt.Errorf("directive parameters must be enclosed in parentheses")
		}
		directiveName = strings.TrimSpace(line[:parenIndex])
		params = strings.TrimSpace(line[parenIndex+1 : len(line)-1])
	} else {
		directiveName = strings.TrimSpace(line)
	}

	return ap.parseClassDirective(directiveName, params, attributes)
}

func (ap *AttributeParser) parseClassDirective(name, params string, attributes *ClassAttributes) error {
	directive := &directives.ClassDirective{
		Name: name,
	}

	switch name {
	case constants.CLASS_ATTR_PRIMARY_KEY, constants.CLASS_ATTR_UNIQUE, constants.CLASS_ATTR_INDEX, constants.CLASS_ATTR_TEXT_INDEX:
		fields, err := ap.parseFieldArray(params)
		if err != nil {
			return fmt.Errorf("failed to parse field array for @@%s: %v", name, err)
		}
		directive.Value = fields

	case constants.CLASS_ATTR_CHECK:
		if params == "" {
			return fmt.Errorf("@@check requires a constraint expression")
		}
		constraint := strings.Trim(params, "\"'")
		if strings.TrimSpace(constraint) == "" {
			return fmt.Errorf("@@check constraint cannot be empty")
		}
		directive.Value = constraint

	default:
		return fmt.Errorf("unknown class directive '@@%s'", name)
	}

	if err := ap.directiveValidator.ValidateClassDirective(directive); err != nil {
		return fmt.Errorf("directive validation failed: %v", err)
	}

	exists := false
	for _, dir := range attributes.Directives {
		if dir.Name == directive.Name && fmt.Sprintf("%v", dir.Value) == fmt.Sprintf("%v", directive.Value) {
			exists = true
			break
		}
	}
	if !exists {
		attributes.Directives = append(attributes.Directives, directive)
	}

	return nil
}

func (ap *AttributeParser) parseFieldArray(params string) ([]string, error) {
	if params == "" {
		return nil, fmt.Errorf("field array cannot be empty")
	}

	params = strings.TrimSpace(params)
	if strings.HasPrefix(params, "[") && strings.HasSuffix(params, "]") {
		params = params[1 : len(params)-1]
	}

	if strings.TrimSpace(params) == "" {
		return nil, fmt.Errorf("field array cannot be empty")
	}

	fieldParts := strings.Split(params, ",")
	var fields []string

	for _, part := range fieldParts {
		field := strings.TrimSpace(part)
		if field != "" {
			if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(field) {
				return nil, fmt.Errorf("invalid field name '%s'", field)
			}
			fields = append(fields, field)
		}
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("at least one field is required")
	}

	return fields, nil
}

func (ap *AttributeParser) validateClassAttributes(attributes *ClassAttributes) error {
	if err := ap.directiveValidator.ValidateMultipleClassDirectives(attributes.Directives); err != nil {
		return fmt.Errorf("directive validation failed: %v", err)
	}

	for i, field := range attributes.Fields {
		if err := ap.fieldValidator.ValidateField(field); err != nil {
			return fmt.Errorf("field %d validation failed: %v", i+1, err)
		}
	}

	if err := ap.validateDirectiveFieldReferences(attributes); err != nil {
		return fmt.Errorf("directive-field validation failed: %v", err)
	}

	return nil
}

func (ap *AttributeParser) validateDirectiveFieldReferences(attributes *ClassAttributes) error {
	fieldNames := make(map[string]bool)
	for _, field := range attributes.Fields {
		fieldNames[field.GetName()] = true
	}

	for _, directive := range attributes.Directives {
		switch directive.Name {
		case constants.CLASS_ATTR_PRIMARY_KEY, constants.CLASS_ATTR_UNIQUE, constants.CLASS_ATTR_INDEX, constants.CLASS_ATTR_TEXT_INDEX:
			fields, err := directive.GetFields()
			if err != nil {
				return fmt.Errorf("failed to get fields from directive @@%s: %v", directive.Name, err)
			}

			for _, fieldName := range fields {
				if !fieldNames[fieldName] {
					return fmt.Errorf("directive @@%s references non-existent field '%s'", directive.Name, fieldName)
				}
			}
		}
	}

	return nil
}

func (ca *ClassAttributes) GetFieldByName(name string) *field.Field {
	for _, field := range ca.Fields {
		if field.GetName() == name {
			return field
		}
	}
	return nil
}

func (ca *ClassAttributes) HasField(name string) bool {
	return ca.GetFieldByName(name) != nil
}

func (ca *ClassAttributes) GetDirectiveByName(name string) *directives.ClassDirective {
	return directives.GetClassDirectiveByName(ca.Directives, name)
}

func (ca *ClassAttributes) HasDirective(name string) bool {
	return ca.GetDirectiveByName(name) != nil
}

func (ca *ClassAttributes) GetPrimaryKeyDirective() *directives.ClassDirective {
	return ca.GetDirectiveByName(constants.CLASS_ATTR_PRIMARY_KEY)
}

func (ca *ClassAttributes) GetUniqueDirective() *directives.ClassDirective {
	return ca.GetDirectiveByName(constants.CLASS_ATTR_UNIQUE)
}

func (ca *ClassAttributes) GetIndexDirective() *directives.ClassDirective {
	return ca.GetDirectiveByName(constants.CLASS_ATTR_INDEX)
}

func (ca *ClassAttributes) GetTextIndexDirective() *directives.ClassDirective {
	return ca.GetDirectiveByName(constants.CLASS_ATTR_TEXT_INDEX)
}

func (ca *ClassAttributes) GetCheckDirective() *directives.ClassDirective {
	return ca.GetDirectiveByName(constants.CLASS_ATTR_CHECK)
}

func (ca *ClassAttributes) HasPrimaryKey() bool {
	return ca.HasDirective(constants.CLASS_ATTR_PRIMARY_KEY)
}

func (ca *ClassAttributes) HasUnique() bool {
	return ca.HasDirective(constants.CLASS_ATTR_UNIQUE)
}

func (ca *ClassAttributes) HasIndex() bool {
	return ca.HasDirective(constants.CLASS_ATTR_INDEX)
}

func (ca *ClassAttributes) HasTextIndex() bool {
	return ca.HasDirective(constants.CLASS_ATTR_TEXT_INDEX)
}

func (ca *ClassAttributes) HasCheck() bool {
	return ca.HasDirective(constants.CLASS_ATTR_CHECK)
}

func (ca *ClassAttributes) GetFieldNames() []string {
	var names []string
	for _, field := range ca.Fields {
		names = append(names, field.GetName())
	}
	return names
}

func (ca *ClassAttributes) GetScalarFields() []*field.Field {
	var scalarFields []*field.Field
	for _, field := range ca.Fields {
		if field.IsScalar() {
			scalarFields = append(scalarFields, field)
		}
	}
	return scalarFields
}

func (ca *ClassAttributes) GetRelationFields() []*field.Field {
	var relationFields []*field.Field
	for _, field := range ca.Fields {
		if field.HasRelation() {
			relationFields = append(relationFields, field)
		}
	}
	return relationFields
}

func (ca *ClassAttributes) GetPrimaryKeyFields() []string {
	if pkDirective := ca.GetPrimaryKeyDirective(); pkDirective != nil {
		fields, err := pkDirective.GetFields()
		if err == nil {
			return fields
		}
	}

	for _, field := range ca.Fields {
		if field.IsPrimaryKey() {
			return []string{field.GetName()}
		}
	}

	return []string{}
}

func (ca *ClassAttributes) String() string {
	var parts []string

	for _, field := range ca.Fields {
		parts = append(parts, "  "+field.String())
	}

	for _, directive := range ca.Directives {
		parts = append(parts, "  "+directive.String())
	}

	return strings.Join(parts, "\n")
}

func (ap *AttributeParser) GetFieldValidator() *field.FieldValidator {
	return ap.fieldValidator
}

func (ap *AttributeParser) GetDirectiveValidator() *directives.DirectiveValidator {
	return ap.directiveValidator
}