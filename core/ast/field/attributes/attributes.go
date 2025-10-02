package attributes

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/rit3sh-x/blaze/core/ast/enum"
	"github.com/rit3sh-x/blaze/core/ast/field/defaults"
	"github.com/rit3sh-x/blaze/core/ast/field/directives"
	"github.com/rit3sh-x/blaze/core/ast/field/relations"
	"github.com/rit3sh-x/blaze/core/constants"
)

type Attribute struct {
	Name  string
	Value interface{}
}

type AttributeDefinition struct {
	Name         string
	DataType     string
	Kind         string
	IsOptional   bool
	IsArray      bool
	Attributes   []*Attribute
	Directives   []*directives.FieldDirective
	DefaultValue *defaults.DefaultValue
	Relation     *relations.Relation
}

type AttributeValidator struct {
	relationValidator  *relations.RelationValidator
	defaultValidator   *defaults.DefaultValidator
	directiveValidator *directives.DirectiveValidator
	fieldPattern       *regexp.Regexp
}

func NewAttributeValidator(enums map[string]*enum.Enum) *AttributeValidator {
	fieldPattern := regexp.MustCompile(`^\s*([A-Za-z_]\w*(?:\[\])?)(\?)?\s*(.*)$`)

	return &AttributeValidator{
		relationValidator:  relations.NewRelationValidator(),
		defaultValidator:   defaults.NewDefaultValidator(enums),
		directiveValidator: directives.NewDirectiveValidator(),
		fieldPattern:       fieldPattern,
	}
}

func (av *AttributeValidator) ParseFieldDefinition(line string, className string, fieldName string) (*AttributeDefinition, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("field definition cannot be empty")
	}

	matches := av.fieldPattern.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil, fmt.Errorf("invalid field definition syntax: %s", line)
	}

	dataType := matches[1]
	isOptional := len(matches) > 2 && matches[2] == "?"
	attributesStr := ""
	if len(matches) > 3 {
		attributesStr = strings.TrimSpace(matches[3])
	}

	isArray := strings.HasSuffix(dataType, "[]")
	if isArray {
		dataType = strings.TrimSuffix(dataType, "[]")
	}

	fieldDef := &AttributeDefinition{
		Name:         fieldName,
		DataType:     dataType,
		IsOptional:   isOptional,
		IsArray:      isArray,
		Attributes:   nil,
		Directives:   []*directives.FieldDirective{},
		DefaultValue: nil,
		Relation:     nil,
	}

	fieldDef.Kind = av.determineFieldKind(dataType)

	if attributesStr != "" {
		if err := av.parseFieldAttributes(attributesStr, fieldDef); err != nil {
			return nil, fmt.Errorf("failed to parse attributes for field '%s': %v", fieldName, err)
		}
	}

	if err := av.processDefaultValue(fieldDef); err != nil {
		return nil, fmt.Errorf("failed to process default value for field '%s': %v", fieldName, err)
	}

	if err := av.processRelation(fieldDef, className); err != nil {
		return nil, fmt.Errorf("failed to process relation for field '%s': %v", fieldName, err)
	}

	if err := av.ValidateFieldDefinition(fieldDef, className); err != nil {
		return nil, fmt.Errorf("field validation failed for '%s': %v", fieldName, err)
	}

	return fieldDef, nil
}

func (av *AttributeValidator) processDefaultValue(fieldDef *AttributeDefinition) error {
	defaultAttr := fieldDef.GetAttribute(constants.FIELD_ATTR_DEFAULT)
	if defaultAttr == nil {
		return nil
	}

	defaultValue, err := defaultAttr.GetDefaultValue(fieldDef.DataType, fieldDef.IsOptional, fieldDef.IsArray, av.defaultValidator)
	if err != nil {
		return fmt.Errorf("invalid default value: %v", err)
	}

	fieldDef.DefaultValue = defaultValue
	return nil
}

func (av *AttributeValidator) determineFieldKind(baseType string) string {
	if constants.IsScalarType(baseType) {
		return constants.FIELD_KIND_SCALAR
	}

	if _, exists := av.defaultValidator.GetEnum(baseType); exists {
		return constants.FIELD_KIND_ENUM
	}

	return constants.FIELD_KIND_OBJECT
}

func (av *AttributeValidator) processRelation(fieldDef *AttributeDefinition, className string) error {
	relationAttr := fieldDef.GetAttribute(constants.FIELD_ATTR_RELATION)
	if relationAttr == nil {
		return nil
	}

	relation, err := relationAttr.GetRelationFromParams(av.relationValidator, fieldDef, className)
	if err != nil {
		return fmt.Errorf("invalid relation: %v", err)
	}

	fieldDef.Relation = relation
	return nil
}

func (av *AttributeValidator) parseFieldAttributes(attributesStr string, fieldDef *AttributeDefinition) error {
	i := 0
	for i < len(attributesStr) {
		if attributesStr[i] == '@' {
			i++

			nameStart := i
			for i < len(attributesStr) && (unicode.IsLetter(rune(attributesStr[i])) || unicode.IsDigit(rune(attributesStr[i])) || attributesStr[i] == '_') {
				i++
			}
			attrName := attributesStr[nameStart:i]

			var attrValue interface{}

			if i < len(attributesStr) && attributesStr[i] == '(' {
				i++
				level := 1
				valueStart := i
				for i < len(attributesStr) && level > 0 {
					switch attributesStr[i] {
					case '(':
						level++
					case ')':
						level--
					}
					i++
				}
				attrValue = strings.TrimSpace(attributesStr[valueStart : i-1])
			}

			if av.isDirective(attrName) {
				directive := &directives.FieldDirective{
					Name:  attrName,
					Value: attrValue,
				}
				fieldDef.Directives = append(fieldDef.Directives, directive)
			} else {
				attribute := &Attribute{
					Name:  attrName,
					Value: attrValue,
				}
				fieldDef.Attributes = append(fieldDef.Attributes, attribute)
			}
		} else {
			i++
		}
	}
	return nil
}

func (av *AttributeValidator) isDirective(attrName string) bool {
	directives := []string{
		constants.FIELD_ATTR_PRIMARY_KEY,
		constants.FIELD_ATTR_UNIQUE,
		constants.FIELD_ATTR_UPDATED_AT,
	}

	for _, directive := range directives {
		if attrName == directive {
			return true
		}
	}
	return false
}

func (av *AttributeValidator) ValidateFieldDefinition(fieldDef *AttributeDefinition, className string) error {
	if err := av.validateFieldName(fieldDef.Name); err != nil {
		return err
	}

	if err := av.validateDataType(fieldDef.DataType); err != nil {
		return err
	}

	if err := av.directiveValidator.ValidateMultipleDirectives(
		fieldDef.Directives,
		fieldDef.DataType,
		fieldDef.IsOptional,
		fieldDef.IsArray,
	); err != nil {
		return err
	}

	for _, attr := range fieldDef.Attributes {
		switch attr.Name {
		case constants.FIELD_ATTR_DEFAULT:
			if err := av.validateDefaultAttribute(attr, fieldDef.DataType, fieldDef.IsArray); err != nil {
				return err
			}
		case constants.FIELD_ATTR_RELATION:
			if err := av.validateRelationAttribute(attr, fieldDef, className); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown field attribute '@%s'", attr.Name)
		}
	}

	if err := av.ValidateMultipleAttributes(fieldDef.Attributes, fieldDef.DataType, fieldDef.IsOptional, fieldDef.IsArray); err != nil {
		return err
	}

	return nil
}

func (av *AttributeValidator) validateFieldName(name string) error {
	if name == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(name) {
		return fmt.Errorf("invalid field name '%s': must start with letter or underscore", name)
	}

	return nil
}

func (av *AttributeValidator) validateDataType(dataType string) error {
	if constants.IsScalarType(dataType) {
		return nil
	}

	if _, exists := av.defaultValidator.GetEnum(dataType); exists {
		return nil
	}

	return nil
}

func (av *AttributeValidator) validateDefaultAttribute(attr *Attribute, fieldType string, isArray bool) error {
	if attr.Value == nil {
		return fmt.Errorf("@default attribute requires a value")
	}

	defaultStr, ok := attr.Value.(string)
	if !ok {
		return fmt.Errorf("@default value must be a string")
	}
	if strings.TrimSpace(defaultStr) == "" {
		return fmt.Errorf("@default value cannot be empty")
	}

	_, err := av.defaultValidator.ValidateDefault(defaultStr, fieldType, isArray)
	if err != nil {
		return fmt.Errorf("invalid @default value: %v", err)
	}

	return nil
}

func (av *AttributeValidator) validateRelationAttribute(attr *Attribute, fieldDef *AttributeDefinition, className string) error {
	if attr.Value == nil {
		return fmt.Errorf("@relation attribute requires parameters")
	}

	relationStr, ok := attr.Value.(string)
	if !ok {
		return fmt.Errorf("@relation value must be defined correctly")
	}

	targetClassName := fieldDef.DataType

	relation, err := av.relationValidator.ParseRelationFromString(relationStr, className, targetClassName)
	if err != nil {
		return fmt.Errorf("invalid @relation: %v", err)
	}

	if err := av.validateRelationFieldType(fieldDef.DataType); err != nil {
		return fmt.Errorf("invalid relation field type: %v", err)
	}

	if err := av.validateRelationConsistency(relation, fieldDef.IsOptional); err != nil {
		return fmt.Errorf("relation consistency error: %v", err)
	}

	return nil
}

func (av *AttributeValidator) validateRelationFieldType(fieldType string) error {
	baseType := fieldType

	if constants.IsScalarType(baseType) {
		scalarType := constants.ScalarType(baseType)
		switch scalarType {
		case constants.INT, constants.BIGINT, constants.SMALLINT, constants.STRING:
			return nil
		default:
			return fmt.Errorf("scalar type '%s' is not commonly used for relations", baseType)
		}
	}

	return nil
}

func (av *AttributeValidator) validateRelationConsistency(relation *relations.Relation, isOptional bool) error {
	if relation.IsComposite() {
		if len(relation.To) < 2 {
			return fmt.Errorf("composite relation must reference at least 2 fields")
		}
	}

	if relation.HasOnDelete() && relation.OnDelete == constants.ON_DELETE_SET_NULL {
		if !isOptional {
			return fmt.Errorf("onDelete: SetNull can only be used with optional fields")
		}
	}

	return nil
}

func (av *AttributeValidator) SetDefaultValidator(dv *defaults.DefaultValidator) {
	av.defaultValidator = dv
}

func (av *AttributeValidator) GetDefaultValidator() *defaults.DefaultValidator {
	return av.defaultValidator
}

func (av *AttributeValidator) GetRelationValidator() *relations.RelationValidator {
	return av.relationValidator
}

func (av *AttributeValidator) GetDirectiveValidator() *directives.DirectiveValidator {
	return av.directiveValidator
}

func (av *AttributeValidator) ValidateAttribute(attr *Attribute, fieldType string, isOptional bool, isArray bool) error {
	switch attr.Name {
	case constants.FIELD_ATTR_DEFAULT:
		return av.validateDefaultAttribute(attr, fieldType, isArray)
	case constants.FIELD_ATTR_RELATION:
		return nil
	default:
		return fmt.Errorf("unknown field attribute '@%s'", attr.Name)
	}
}

func (av *AttributeValidator) ParseAttributeFromString(attrStr string) (*Attribute, error) {
	attrStr = strings.TrimSpace(attrStr)
	if !strings.HasPrefix(attrStr, "@") {
		return nil, fmt.Errorf("attribute must start with '@'")
	}

	attrStr = attrStr[1:]

	if strings.Contains(attrStr, "(") {
		parenIndex := strings.Index(attrStr, "(")
		if !strings.HasSuffix(attrStr, ")") {
			return nil, fmt.Errorf("attribute parameters must be enclosed in parentheses")
		}

		attrName := strings.TrimSpace(attrStr[:parenIndex])
		attrValue := strings.TrimSpace(attrStr[parenIndex+1 : len(attrStr)-1])

		if !av.isValidAttributeName(attrName) {
			return nil, fmt.Errorf("invalid attribute name '@%s'", attrName)
		}

		switch attrName {
		case constants.FIELD_ATTR_DEFAULT, constants.FIELD_ATTR_RELATION:
			if attrValue == "" {
				return nil, fmt.Errorf("@%s attribute requires non-empty parameters", attrName)
			}
		case constants.FIELD_ATTR_PRIMARY_KEY, constants.FIELD_ATTR_UNIQUE, constants.FIELD_ATTR_UPDATED_AT:
			return nil, fmt.Errorf("@%s attribute does not accept parameters", attrName)
		}

		return &Attribute{
			Name:  attrName,
			Value: attrValue,
		}, nil
	} else {
		attrName := strings.TrimSpace(attrStr)

		if !av.isValidAttributeName(attrName) {
			return nil, fmt.Errorf("invalid attribute name '@%s'", attrName)
		}

		switch attrName {
		case constants.FIELD_ATTR_DEFAULT:
			return nil, fmt.Errorf("@default attribute requires parameters")
		case constants.FIELD_ATTR_RELATION:
			return nil, fmt.Errorf("@relation attribute requires parameters")
		}

		return &Attribute{
			Name:  attrName,
			Value: nil,
		}, nil
	}
}

func (av *AttributeValidator) isValidAttributeName(name string) bool {
	validAttributes := []string{
		constants.FIELD_ATTR_PRIMARY_KEY,
		constants.FIELD_ATTR_UNIQUE,
		constants.FIELD_ATTR_DEFAULT,
		constants.FIELD_ATTR_RELATION,
		constants.FIELD_ATTR_UPDATED_AT,
	}

	for _, valid := range validAttributes {
		if name == valid {
			return true
		}
	}
	return false
}

func (av *AttributeValidator) ValidateMultipleAttributes(attrs []*Attribute, fieldType string, isOptional bool, isArray bool) error {
	if len(attrs) == 0 {
		return nil
	}

	attributeCount := make(map[string]int)
	var hasDefault, hasRelation bool

	for _, attr := range attrs {
		if attr == nil {
			continue
		}

		attributeCount[attr.Name]++

		if attributeCount[attr.Name] > 1 {
			return fmt.Errorf("duplicate attribute '@%s' found", attr.Name)
		}

		if err := av.ValidateAttribute(attr, fieldType, isOptional, isArray); err != nil {
			return err
		}

		switch attr.Name {
		case constants.FIELD_ATTR_DEFAULT:
			hasDefault = true
		case constants.FIELD_ATTR_RELATION:
			hasRelation = true
		}
	}

	if hasRelation && hasDefault {
	}

	return nil
}

func (fd *AttributeDefinition) HasAttribute(name string) bool {
	return GetAttributeByName(fd.Attributes, name) != nil
}

func (fd *AttributeDefinition) HasDirective(name string) bool {
	return directives.HasDirective(fd.Directives, name)
}

func (fd *AttributeDefinition) GetAttribute(name string) *Attribute {
	return GetAttributeByName(fd.Attributes, name)
}

func (fd *AttributeDefinition) GetDirective(name string) *directives.FieldDirective {
	return directives.GetDirectiveByName(fd.Directives, name)
}

func (fd *AttributeDefinition) IsRelationField() bool {
	return fd.Relation != nil
}

func (fd *AttributeDefinition) IsPrimaryKey() bool {
	return fd.HasDirective(constants.FIELD_ATTR_PRIMARY_KEY)
}

func (fd *AttributeDefinition) IsUnique() bool {
	return fd.HasDirective(constants.FIELD_ATTR_UNIQUE)
}

func (fd *AttributeDefinition) HasDefault() bool {
	return fd.DefaultValue != nil
}

func (fd *AttributeDefinition) GetDefaultValue() *defaults.DefaultValue {
	return fd.DefaultValue
}

func (fd *AttributeDefinition) GetRelation() *relations.Relation {
	return fd.Relation
}

func (fd *AttributeDefinition) String() string {
	var builder strings.Builder

	builder.WriteString(fd.Name)
	builder.WriteString(" ")
	builder.WriteString(fd.DataType)

	if fd.IsArray {
		builder.WriteString("[]")
	}

	if fd.IsOptional {
		builder.WriteString("?")
	}

	for _, directive := range fd.Directives {
		builder.WriteString(" ")
		builder.WriteString(directive.String())
	}

	for _, attribute := range fd.Attributes {
		builder.WriteString(" ")
		builder.WriteString(attribute.String())
	}

	return builder.String()
}

func (fa *Attribute) String() string {
	if fa.Value != nil {
		return fmt.Sprintf("@%s(%v)", fa.Name, fa.Value)
	}
	return fmt.Sprintf("@%s", fa.Name)
}

func (fa *Attribute) IsParameterized() bool {
	return fa.Value != nil
}

func (fa *Attribute) GetStringValue() (string, bool) {
	if str, ok := fa.Value.(string); ok {
		return str, true
	}
	return "", false
}

func (fa *Attribute) GetIntValue() (int, bool) {
	if val, ok := fa.Value.(string); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal, true
		}
	}
	if intVal, ok := fa.Value.(int); ok {
		return intVal, true
	}
	return 0, false
}

func (fa *Attribute) GetDefaultValue(fieldType string, isOptional bool, isArray bool, defaultValidator *defaults.DefaultValidator) (*defaults.DefaultValue, error) {
	if fa.Name != constants.FIELD_ATTR_DEFAULT {
		return nil, fmt.Errorf("attribute is not a default attribute")
	}

	defaultStr, ok := fa.GetStringValue()
	if !ok {
		return nil, fmt.Errorf("default value is not a string")
	}

	return defaultValidator.ValidateDefault(defaultStr, fieldType, isArray)
}

func (fa *Attribute) GetRelationFromParams(relationValidator *relations.RelationValidator, fieldDef *AttributeDefinition, className string) (*relations.Relation, error) {
	if fa.Name != constants.FIELD_ATTR_RELATION {
		return nil, fmt.Errorf("attribute is not a relation attribute")
	}

	relationStr, ok := fa.GetStringValue()
	if !ok {
		return nil, fmt.Errorf("relation value is not a string")
	}

	targetClassName := fieldDef.DataType

	return relationValidator.ParseRelationFromString(relationStr, className, targetClassName)
}

func GetAttributeByName(attrs []*Attribute, name string) *Attribute {
	for _, attr := range attrs {
		if attr != nil && attr.Name == name {
			return attr
		}
	}
	return nil
}

func HasAttribute(attrs []*Attribute, name string) bool {
	return GetAttributeByName(attrs, name) != nil
}

func GetDefaultAttribute(attrs []*Attribute) *Attribute {
	return GetAttributeByName(attrs, constants.FIELD_ATTR_DEFAULT)
}

func GetRelationAttribute(attrs []*Attribute) *Attribute {
	return GetAttributeByName(attrs, constants.FIELD_ATTR_RELATION)
}

func HasDefault(attrs []*Attribute) bool {
	return HasAttribute(attrs, constants.FIELD_ATTR_DEFAULT)
}

func HasRelation(attrs []*Attribute) bool {
	return HasAttribute(attrs, constants.FIELD_ATTR_RELATION)
}

func (ad *AttributeDefinition) Clone() *AttributeDefinition {
	clone := &AttributeDefinition{
		Name:         ad.Name,
		DataType:     ad.DataType,
		Kind:         ad.Kind,
		IsOptional:   ad.IsOptional,
		IsArray:      ad.IsArray,
		DefaultValue: ad.DefaultValue,
		Relation:     ad.Relation,
	}

	for _, attr := range ad.Attributes {
		clonedAttr := &Attribute{
			Name:  attr.Name,
			Value: attr.Value,
		}
		clone.Attributes = append(clone.Attributes, clonedAttr)
	}

	for _, directive := range ad.Directives {
		clonedDirective := &directives.FieldDirective{
			Name:  directive.Name,
			Value: directive.Value,
		}
		clone.Directives = append(clone.Directives, clonedDirective)
	}

	return clone
}
