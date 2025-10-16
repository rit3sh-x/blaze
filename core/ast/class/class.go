package class

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast/class/attributes"
	"github.com/rit3sh-x/blaze/core/ast/enum"
	"github.com/rit3sh-x/blaze/core/ast/field"
	"github.com/rit3sh-x/blaze/core/constants"
)

type Class struct {
	Name       string
	Attributes *attributes.ClassAttributes
	Position   int
}

type ClassValidator struct {
	attributeParser  *attributes.AttributeParser
	enumValidator    *enum.EnumValidator
	classNamePattern *regexp.Regexp
	enumRegistry     map[string]*enum.Enum
}

func NewClassValidator(enums map[string]*enum.Enum) *ClassValidator {
	return &ClassValidator{
		attributeParser:  attributes.NewAttributeParser(enums),
		enumValidator:    enum.NewEnumValidator(),
		classNamePattern: regexp.MustCompile(`^[A-Z][a-zA-Z0-9_]{0,63}$`),
		enumRegistry:     make(map[string]*enum.Enum),
	}
}

func (cv *ClassValidator) ParseClass(definition string, position int) (*Class, error) {
	definition = strings.TrimSpace(definition)

	classPattern := regexp.MustCompile(`(?s)^` + constants.KEYWORD_CLASS + `\s+([A-Z][a-zA-Z0-9_]{0,63})\s*\{([^{}]*)\}$`)

	matches := classPattern.FindStringSubmatch(definition)
	if matches == nil {
		return nil, fmt.Errorf("invalid class definition")
	}

	namePart := matches[1]
	contentPart := strings.TrimSpace(matches[2])

	if err := cv.validateBalancedBraces(contentPart); err != nil {
        return nil, fmt.Errorf("unbalanced braces in class content: %v", err)
    }

	if err := cv.validateClassName(namePart); err != nil {
		return nil, fmt.Errorf("invalid class name: %v", err)
	}

	classAttributes, err := cv.attributeParser.ParseClassContent(contentPart, namePart)
	if err != nil {
		return nil, fmt.Errorf("failed to parse class content: %v", err)
	}
	
	class := &Class{
		Name:       namePart,
		Attributes: classAttributes,
		Position:   position,
	}

	if err := cv.ValidateClass(class); err != nil {
		return nil, fmt.Errorf("class validation failed: %v", err)
	}

	return class, nil
}

func (cv *ClassValidator) validateBalancedBraces(content string) error {
    braceCount := 0
    parenCount := 0
    
    for _, char := range content {
        switch char {
        case '{':
            braceCount++
        case '}':
            braceCount--
            if braceCount < 0 {
                return fmt.Errorf("unmatched closing brace")
            }
        case '(':
            parenCount++
        case ')':
            parenCount--
            if parenCount < 0 {
                return fmt.Errorf("unmatched closing parenthesis")
            }
        }
    }
    
    if braceCount != 0 {
        return fmt.Errorf("unmatched braces")
    }
    if parenCount != 0 {
        return fmt.Errorf("unmatched parentheses")
    }
    
    return nil
}

func (cv *ClassValidator) ValidateClass(class *Class) error {
	if class == nil {
		return fmt.Errorf("class cannot be nil")
	}

	if err := cv.validateClassName(class.Name); err != nil {
		return fmt.Errorf("invalid class name: %v", err)
	}

	if class.Attributes == nil {
		return fmt.Errorf("class attributes cannot be nil")
	}

	if err := cv.validateClassConstraints(class); err != nil {
		return fmt.Errorf("class constraint validation failed: %v", err)
	}

	if err := cv.validateFieldTypes(class); err != nil {
		return fmt.Errorf("field type validation failed: %v", err)
	}

	if err := cv.validatePrimaryKeyConstraints(class); err != nil {
		return fmt.Errorf("primary key validation failed: %v", err)
	}

	return nil
}

func (cv *ClassValidator) validateClassName(name string) error {
	name = strings.TrimSpace(name)

	if !regexp.MustCompile(`^[A-Z][a-zA-Z0-9_]{0,63}$`).MatchString(name) {
        return fmt.Errorf("class name must start with capital letter, contain only alphanumeric characters/underscores, and be at most 64 chars")
    }

	return nil
}

func (cv *ClassValidator) validateClassConstraints(class *Class) error {
	if len(class.Attributes.Fields) == 0 {
		return fmt.Errorf("class '%s' must have at least one field", class.Name)
	}

	if len(class.Attributes.Fields) > 1000 {
		return fmt.Errorf("class '%s' has too many fields (max 1000)", class.Name)
	}

	return nil
}

func (cv *ClassValidator) validateFieldTypes(class *Class) error {
	for _, field := range class.Attributes.Fields {
		baseType := field.GetBaseType()

		if constants.IsScalarType(baseType) {
			continue
		}

		if _, exists := cv.enumRegistry[baseType]; exists {
			continue
		}

		if cv.classNamePattern.MatchString(baseType) {
			continue
		}

		return fmt.Errorf("unknown type '%s' for field '%s' in class '%s'", baseType, field.GetName(), class.Name)
	}

	return nil
}

func (cv *ClassValidator) validatePrimaryKeyConstraints(class *Class) error {
	hasFieldPK := false
	hasClassPK := class.Attributes.HasPrimaryKey()

	for _, field := range class.Attributes.Fields {
		if field.IsPrimaryKey() {
			hasFieldPK = true
			break
		}
	}

	if hasFieldPK && hasClassPK {
		return fmt.Errorf("class '%s' cannot have both field-level and class-level primary keys", class.Name)
	}

	if !hasFieldPK && !hasClassPK {
		return fmt.Errorf("class '%s' must have a primary key", class.Name)
	}

	if hasClassPK {
		pkFields := class.Attributes.GetPrimaryKeyFields()
		if len(pkFields) == 0 {
			return fmt.Errorf("class '%s' primary key directive references no valid fields", class.Name)
		}

		for _, fieldName := range pkFields {
			field := class.Attributes.GetFieldByName(fieldName)
			if field == nil {
				return fmt.Errorf("primary key field '%s' not found in class '%s'", fieldName, class.Name)
			}
			if field.IsOptional() {
				return fmt.Errorf("primary key field '%s' in class '%s' cannot be optional", fieldName, class.Name)
			}
			if field.IsArray() {
				return fmt.Errorf("primary key field '%s' in class '%s' cannot be an array", fieldName, class.Name)
			}
		}
	}

	return nil
}

func (c *Class) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("class %s {", c.Name))
	parts = append(parts, c.Attributes.String())
	parts = append(parts, "}")
	return strings.Join(parts, "\n")
}

func (c *Class) HasPrimaryKey() bool {
	for _, field := range c.Attributes.Fields {
		if field.IsPrimaryKey() {
			return true
		}
	}
	return c.Attributes.HasPrimaryKey()
}

func (c *Class) GetPrimaryKeyFields() []string {
	return c.Attributes.GetPrimaryKeyFields()
}

func (c *Class) GetRelationFields() []*field.Field {
	return c.Attributes.GetRelationFields()
}

func (c *Class) GetScalarFields() []*field.Field {
	return c.Attributes.GetScalarFields()
}

func (c *Class) GetFieldNames() []string {
	return c.Attributes.GetFieldNames()
}

func (cv *ClassValidator) GetEnumRegistry() map[string]*enum.Enum {
	return cv.enumRegistry
}

func (cv *ClassValidator) GetAttributeParser() *attributes.AttributeParser {
	return cv.attributeParser
}