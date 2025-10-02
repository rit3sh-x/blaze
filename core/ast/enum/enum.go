package enum

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/rit3sh-x/blaze/core/constants"
)

type EnumValue struct {
	Name     string
	Position int
}

type Enum struct {
	Name     string
	Values   []EnumValue
	Position int
}

type EnumValidator struct {
	valueSplitRegex   *regexp.Regexp
	enumRegex         *regexp.Regexp
	reservedKeywords  map[string]bool
	identifierPattern *regexp.Regexp
}

func NewEnumValidator() *EnumValidator {
	reserved := map[string]bool{
		constants.KEYWORD_ENUM:  true,
		constants.KEYWORD_CLASS: true,
		"true":                  true,
		"false":                 true,
		"null":                  true,
	}

	for _, scalarType := range constants.ScalarTypes {
		reserved[strings.ToLower(string(scalarType))] = true
	}

	identifierPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	return &EnumValidator{
		enumRegex:         regexp.MustCompile(`^` + constants.KEYWORD_ENUM + `\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{([^}]*)\}\s*$`),
		reservedKeywords:  reserved,
		identifierPattern: identifierPattern,
		valueSplitRegex:   regexp.MustCompile(`\s+`),
	}
}

func (v *EnumValidator) ValidateEnumName(name string) error {
	if name == "" {
		return fmt.Errorf("enum name cannot be empty")
	}

	if !v.identifierPattern.MatchString(name) {
		return fmt.Errorf("invalid enum name '%s': must start with letter or underscore, followed by letters, digits, or underscores", name)
	}

	if v.reservedKeywords[strings.ToLower(name)] {
		return fmt.Errorf("enum name '%s' is a reserved keyword", name)
	}

	if constants.IsScalarType(name) {
		return fmt.Errorf("enum name '%s' conflicts with scalar type", name)
	}

	if len(name) > 64 {
		return fmt.Errorf("enum name '%s' is too long (max 64 characters)", name)
	}

	return nil
}

func (v *EnumValidator) ValidateEnumValue(value string, enumName string) error {
	if value == "" {
		return fmt.Errorf("enum value cannot be empty in enum '%s'", enumName)
	}

	if !v.identifierPattern.MatchString(value) {
		return fmt.Errorf("invalid enum value '%s' in enum '%s': must start with letter or underscore, followed by letters, digits, or underscores", value, enumName)
	}

	if v.reservedKeywords[strings.ToLower(value)] {
		return fmt.Errorf("enum value '%s' in enum '%s' is a reserved keyword", value, enumName)
	}

	if constants.IsScalarType(value) {
		return fmt.Errorf("enum value '%s' in enum '%s' conflicts with scalar type", value, enumName)
	}

	if !v.isValidEnumValueNaming(value) {
		return fmt.Errorf("enum value '%s' in enum '%s' should be in UPPERCASE or PascalCase", value, enumName)
	}

	if len(value) > 64 {
		return fmt.Errorf("enum value '%s' in enum '%s' is too long (max 64 characters)", value, enumName)
	}

	return nil
}

func (v *EnumValidator) isValidEnumValueNaming(value string) bool {
	if strings.ToUpper(value) == value && !strings.Contains(value, "_") {
		return true
	}

	if strings.ToUpper(value) == value {
		return true
	}

	if unicode.IsUpper(rune(value[0])) {
		return true
	}

	return false
}

func (v *EnumValidator) ValidateEnum(enum *Enum) error {
	if err := v.ValidateEnumName(enum.Name); err != nil {
		return err
	}

	if len(enum.Values) == 0 {
		return fmt.Errorf("enum '%s' must have at least one value", enum.Name)
	}

	if len(enum.Values) > 255 {
		return fmt.Errorf("enum '%s' has too many values (max 255)", enum.Name)
	}

	valueMap := make(map[string]bool)

	for i, enumValue := range enum.Values {
		if err := v.ValidateEnumValue(enumValue.Name, enum.Name); err != nil {
			return err
		}

		lowerValue := strings.ToLower(enumValue.Name)
		if valueMap[lowerValue] {
			return fmt.Errorf("duplicate enum value '%s' in enum '%s'", enumValue.Name, enum.Name)
		}
		valueMap[lowerValue] = true

		if enumValue.Position != i + 1 {
			return fmt.Errorf("invalid position for enum value '%s' in enum '%s'", enumValue.Name, enum.Name)
		}
	}

	return nil
}

func (v *EnumValidator) ParseEnum(definition string, position int) (*Enum, error) {
	definition = strings.TrimSpace(definition)

	matches := v.enumRegex.FindStringSubmatch(definition)
	if matches == nil {
		return nil, fmt.Errorf("invalid enum definition: must match pattern 'enum Name { VAL1 VAL2 ... }'")
	}

	namePart := matches[1]
	valuesPart := matches[2]

	if namePart == "" {
		return nil, fmt.Errorf("enum name is required")
	}
	if valuesPart == "" {
		return nil, fmt.Errorf("enum must have at least one value")
	}

	rawValues := v.valueSplitRegex.Split(valuesPart, -1)
	var values []EnumValue

	for i, raw := range rawValues {
		val := strings.TrimSpace(raw)
		if val == "" {
			continue
		}
		values = append(values, EnumValue{
			Name:     val,
			Position: i,
		})
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("enum must have at least one non-empty value")
	}

	enum := &Enum{
		Name:     namePart,
		Values:   values,
		Position: position,
	}

	if err := v.ValidateEnum(enum); err != nil {
		return nil, err
	}

	return enum, nil
}

func (e *Enum) GetEnumValue(name string) (*EnumValue, error) {
	for _, value := range e.Values {
		if value.Name == name {
			return &value, nil
		}
	}
	return nil, fmt.Errorf("enum value '%s' not found in enum '%s'", name, e.Name)
}

func (e *Enum) HasValue(name string) bool {
	_, err := e.GetEnumValue(name)
	return err == nil
}

func (e *Enum) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s %s {\n", constants.KEYWORD_ENUM, e.Name))
	for _, value := range e.Values {
		builder.WriteString(fmt.Sprintf("  %s\n", value.Name))
	}
	builder.WriteString("}")
	return builder.String()
}

func (ev *EnumValue) String() string {
	return ev.Name
}