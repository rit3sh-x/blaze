package field

import (
	"fmt"
	"regexp"

	"github.com/rit3sh-x/blaze/core/ast/enum"
	"github.com/rit3sh-x/blaze/core/ast/field/attributes"
	"github.com/rit3sh-x/blaze/core/constants"
)

type Field struct {
	AttributeDefinition *attributes.AttributeDefinition
	Position            int
}

type FieldValidator struct {
	attributeValidator *attributes.AttributeValidator
	fieldNamePattern   *regexp.Regexp
}

func NewFieldValidator(enums map[string]*enum.Enum) *FieldValidator {
	fieldNamePattern := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s+(.+)$`)

	return &FieldValidator{
		attributeValidator: attributes.NewAttributeValidator(enums),
		fieldNamePattern:   fieldNamePattern,
	}
}

func (fv *FieldValidator) GetAttributeValidator() *attributes.AttributeValidator {
	return fv.attributeValidator
}

func (fv *FieldValidator) ParseFieldFromString(fieldName string, remainingStr string, className string, position int) (*Field, error) {
	attributeDefinition, err := fv.attributeValidator.ParseFieldDefinition(remainingStr, className, fieldName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse field definition: %v", err)
	}

	field := &Field{
		AttributeDefinition: attributeDefinition,
		Position:            position,
	}

	return field, nil
}

func (fv *FieldValidator) ValidateField(field *Field) error {
	if field == nil {
		return fmt.Errorf("field cannot be nil")
	}

	if field.AttributeDefinition == nil {
		return fmt.Errorf("field attribute definition cannot be nil")
	}

	return nil
}

func (f *Field) GetName() string {
	if f.AttributeDefinition == nil {
		return ""
	}
	return f.AttributeDefinition.Name
}

func (f *Field) GetDataType() string {
	if f.AttributeDefinition == nil {
		return ""
	}
	return f.AttributeDefinition.DataType
}

func (f *Field) IsOptional() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.IsOptional
}

func (f *Field) IsArray() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.IsArray
}

func (f *Field) HasAttribute(attrName string) bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.HasAttribute(attrName)
}

func (f *Field) GetAttribute(attrName string) *attributes.Attribute {
	if f.AttributeDefinition == nil {
		return nil
	}
	return f.AttributeDefinition.GetAttribute(attrName)
}

func (f *Field) HasDefault() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.HasDefault()
}

func (f *Field) HasRelation() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.IsRelationField()
}

func (f *Field) IsPrimaryKey() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.IsPrimaryKey()
}

func (f *Field) IsUnique() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.IsUnique()
}

func (f *Field) IsUpdatedAt() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.HasAttribute(constants.FIELD_ATTR_UPDATED_AT)
}

func (f *Field) GetKind() string {
	if f.AttributeDefinition == nil {
		return ""
	}
	return f.AttributeDefinition.Kind
}

func (f *Field) IsScalar() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.Kind == constants.FIELD_KIND_SCALAR
}

func (f *Field) IsEnum() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.Kind == constants.FIELD_KIND_ENUM
}

func (f *Field) IsObject() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.Kind == constants.FIELD_KIND_OBJECT
}

func (f *Field) GetBaseType() string {
	if f.AttributeDefinition == nil {
		return ""
	}
	return f.AttributeDefinition.DataType
}

func (f *Field) IsForeignKey() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.Kind == constants.FIELD_KIND_OBJECT && f.HasRelation()
}

func (f *Field) IsBackReference() bool {
	if f.AttributeDefinition == nil || f.HasRelation() {
		return false
	}
	return f.AttributeDefinition.Kind == constants.FIELD_KIND_OBJECT
}

func (f *Field) String() string {
	if f.AttributeDefinition == nil {
		return ""
	}
	return f.AttributeDefinition.String()
}

func (f *Field) IsRequiredField() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.IsOptional
}

func (f *Field) IsSystemManaged() bool {
	if f.AttributeDefinition == nil {
		return false
	}
	return f.AttributeDefinition.HasAttribute(constants.FIELD_ATTR_UPDATED_AT)
}

func (f *Field) Clone() *Field {
	if f.AttributeDefinition == nil {
		return &Field{Position: f.Position}
	}

	return &Field{
		AttributeDefinition: f.AttributeDefinition.Clone(),
		Position:            f.Position,
	}
}
