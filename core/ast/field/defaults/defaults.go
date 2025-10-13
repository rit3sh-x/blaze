package defaults

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rit3sh-x/blaze/core/ast/enum"
	"github.com/rit3sh-x/blaze/core/constants"
)

type DefaultValue struct {
	Value    interface{}
	Type     string
	DataType string
	IsArray  bool
}

type DefaultValidator struct {
	enums           map[string]*enum.Enum
	callbackPattern *regexp.Regexp
	stringPattern   *regexp.Regexp
	arrayPattern    *regexp.Regexp
	elementRegex    *regexp.Regexp
}

func NewDefaultValidator(enums map[string]*enum.Enum) *DefaultValidator {
	callbackPattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*\(\)$`)
	stringPattern := regexp.MustCompile(`^"([^"\\]|\\.)*"$|^'([^'\\]|\\.)*'$`)
	arrayPattern := regexp.MustCompile(`^\s*\[.*\]\s*$`)

	dv := &DefaultValidator{
		enums:           enums,
		callbackPattern: callbackPattern,
		stringPattern:   stringPattern,
		arrayPattern:    arrayPattern,
		elementRegex:    regexp.MustCompile(`(?:"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^,]+)`),
	}

	return dv
}

func (dv *DefaultValidator) ValidateDefault(defaultStr string, fieldType string, isArray bool) (*DefaultValue, error) {
	if strings.TrimSpace(defaultStr) == "" {
		return nil, fmt.Errorf("default value cannot be empty")
	}
	if strings.TrimSpace(fieldType) == "" {
		return nil, fmt.Errorf("field type cannot be empty")
	}

	defaultStr = strings.TrimSpace(defaultStr)
	fieldType = strings.TrimSpace(fieldType)

	baseType := fieldType
	if isArray {
		if strings.HasSuffix(fieldType, "[]") {
			baseType = strings.TrimSuffix(fieldType, "[]")
		}
		baseType = strings.TrimSpace(baseType)
		if baseType == "" {
			return nil, fmt.Errorf("invalid array type: missing base type")
		}
	}

	if isArray {
		if !dv.arrayPattern.MatchString(defaultStr) {
			return nil, fmt.Errorf("array field requires array default value syntax [element1, element2, ...], got: %s", defaultStr)
		}
		return dv.validateArrayDefault(defaultStr, baseType, fieldType)
	}

	if dv.arrayPattern.MatchString(defaultStr) {
		return nil, fmt.Errorf("non-array field cannot have array default value: %s", defaultStr)
	}

	if dv.callbackPattern.MatchString(defaultStr) {
		return dv.validateCallback(defaultStr, fieldType)
	}

	if dv.isEnumType(baseType) {
		return dv.validateEnumDefault(defaultStr, baseType)
	}

	if constants.IsScalarType(baseType) {
		return dv.validateScalarDefault(defaultStr, baseType)
	}

	return nil, fmt.Errorf("unsupported field type for default value: %s", fieldType)
}

func (dv *DefaultValidator) validateArrayDefault(defaultStr string, baseType string, fullType string) (*DefaultValue, error) {
	if len(defaultStr) < 2 || defaultStr[0] != '[' || defaultStr[len(defaultStr)-1] != ']' {
		return nil, fmt.Errorf("invalid array syntax: must be enclosed in brackets [...]")
	}

	arrayContent := strings.TrimSpace(defaultStr[1 : len(defaultStr)-1])

	if arrayContent == "" {
		return &DefaultValue{
			Value:    []interface{}{},
			Type:     "literal",
			DataType: fullType,
			IsArray:  true,
		}, nil
	}

	elements, err := dv.parseArrayElements(arrayContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse array elements: %v", err)
	}

	if len(elements) == 0 {
		return &DefaultValue{
			Value:    []interface{}{},
			Type:     "literal",
			DataType: fullType,
			IsArray:  true,
		}, nil
	}

	var validatedElements []interface{}
	for i, element := range elements {
		elementStr := strings.TrimSpace(element)
		if elementStr == "" {
			return nil, fmt.Errorf("array element at index %d cannot be empty", i)
		}

		if strings.Contains(elementStr, "[") || strings.Contains(elementStr, "]") {
			return nil, fmt.Errorf("nested arrays are not supported, found at element index %d: %s", i, elementStr)
		}

		var validatedElement interface{}

		if dv.isEnumType(baseType) {
			elementValue, err := dv.validateEnumDefault(elementStr, baseType)
			if err != nil {
				return nil, fmt.Errorf("invalid enum array element at index %d: %v", i, err)
			}
			validatedElement = elementValue.Value
		} else if constants.IsScalarType(baseType) {
			elementValue, err := dv.validateScalarDefault(elementStr, baseType)
			if err != nil {
				return nil, fmt.Errorf("invalid scalar array element at index %d (%s): %v", i, elementStr, err)
			}
			validatedElement = elementValue.Value
		} else {
			return nil, fmt.Errorf("unsupported array element type '%s' at index %d", baseType, i)
		}

		validatedElements = append(validatedElements, validatedElement)
	}

	return &DefaultValue{
		Value:    validatedElements,
		Type:     "literal",
		DataType: fullType,
		IsArray:  true,
	}, nil
}

func (dv *DefaultValidator) parseArrayElements(arrayContent string) ([]string, error) {
	arrayContent = strings.TrimSpace(arrayContent)
	if arrayContent == "" {
		return []string{}, nil
	}

	matches := dv.elementRegex.FindAllString(arrayContent, -1)
	if matches == nil {
		return []string{}, nil
	}

	var elements []string
	for _, match := range matches {
		element := strings.TrimSpace(match)
		if element == "" {
			continue
		}

		isQuoted := (len(element) >= 2) &&
			((element[0] == '"' && element[len(element)-1] == '"') || (element[0] == '\'' && element[len(element)-1] == '\''))

		if !isQuoted {
			if strings.Contains(element, "[") || strings.Contains(element, "]") {
				return nil, fmt.Errorf("nested arrays are not supported in 1D array defaults")
			}
		} else {
			element = element[1 : len(element)-1]
		}

		elements = append(elements, element)
	}

	return elements, nil
}

func (dv *DefaultValidator) validateCallback(callback string, fieldType string) (*DefaultValue, error) {
	if !constants.IsValidCallback(callback) {
		validCallbacks := strings.Join(constants.ValidCallbacks, ", ")
		return nil, fmt.Errorf("unknown callback function '%s'. Valid callbacks: %s", callback, validCallbacks)
	}

	if strings.HasSuffix(fieldType, "[]") {
		return nil, fmt.Errorf("callback functions are not supported for array types")
	}

	scalarType, exists := constants.GetScalarType(fieldType)
	if !exists {
		return nil, fmt.Errorf("callback '%s' cannot be used with non-scalar type '%s'", callback, fieldType)
	}

	if !constants.IsCallbackCompatibleWithType(callback, scalarType) {
		compatibleTypes := constants.GetCallbackCompatibleTypes(callback)
		var typeNames []string
		for _, t := range compatibleTypes {
			typeNames = append(typeNames, string(t))
		}
		return nil, fmt.Errorf("callback '%s' is not compatible with field type '%s'. Compatible types: %v",
			callback, fieldType, typeNames)
	}

	return &DefaultValue{
		Value:    callback,
		Type:     "callback",
		DataType: fieldType,
		IsArray:  false,
	}, nil
}

func (dv *DefaultValidator) validateEnumDefault(defaultStr string, enumType string) (*DefaultValue, error) {
	enumDef, exists := dv.enums[enumType]
	if !exists {
		var availableEnums []string
		for name := range dv.enums {
			availableEnums = append(availableEnums, name)
		}
		return nil, fmt.Errorf("unknown enum type '%s'. Available enums: %v", enumType, availableEnums)
	}

	enumValidator := enum.NewEnumValidator()
	if err := enumValidator.ValidateEnum(enumDef); err != nil {
		return nil, fmt.Errorf("enum '%s' is invalid: %v", enumType, err)
	}

	cleanValue := defaultStr
	if dv.stringPattern.MatchString(defaultStr) {
		cleanValue = defaultStr[1 : len(defaultStr)-1]
	}

	if !enumDef.HasValue(cleanValue) {
		var availableValues []string
		for _, enumValue := range enumDef.Values {
			availableValues = append(availableValues, enumValue.Name)
		}
		return nil, fmt.Errorf("invalid enum default value '%s' for enum '%s'. Valid values: %v",
			cleanValue, enumType, availableValues)
	}

	if err := enumValidator.ValidateEnumValue(cleanValue, enumType); err != nil {
		return nil, fmt.Errorf("enum value '%s' validation failed for enum '%s': %v", cleanValue, enumType, err)
	}

	return &DefaultValue{
		Value:    cleanValue,
		Type:     constants.KEYWORD_ENUM,
		DataType: enumType,
		IsArray:  false,
	}, nil
}

func (dv *DefaultValidator) validateScalarDefault(defaultStr string, fieldType string) (*DefaultValue, error) {
	scalarType, exists := constants.GetScalarType(fieldType)
	if !exists {
		return nil, fmt.Errorf("unknown scalar type: %s", fieldType)
	}

	switch scalarType {
	case constants.INT, constants.SMALLINT, constants.BIGINT:
		return dv.validateIntDefault(defaultStr, fieldType)
	case constants.FLOAT, constants.NUMERIC:
		return dv.validateFloatDefault(defaultStr, fieldType)
	case constants.STRING:
		return dv.validateStringDefault(defaultStr, fieldType)
	case constants.BOOLEAN:
		return dv.validateBooleanDefault(defaultStr, fieldType)
	case constants.CHAR:
		return dv.validateCharDefault(defaultStr, fieldType)
	case constants.DATE:
		return dv.validateDateDefault(defaultStr, fieldType)
	case constants.TIMESTAMP:
		return dv.validateTimestampDefault(defaultStr, fieldType)
	case constants.JSON:
		return dv.validateJsonDefault(defaultStr, fieldType)
	case constants.BYTES:
		return dv.validateBytesDefault(defaultStr, fieldType)
	default:
		return nil, fmt.Errorf("unsupported scalar type for default value: %s", fieldType)
	}
}

func (dv *DefaultValidator) validateIntDefault(defaultStr string, fieldType string) (*DefaultValue, error) {
	cleanValue := defaultStr
	if dv.stringPattern.MatchString(defaultStr) {
		cleanValue = defaultStr[1 : len(defaultStr)-1]
	}

	val, err := strconv.ParseInt(strings.TrimSpace(cleanValue), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid integer default value '%s' for type %s: %v", defaultStr, fieldType, err)
	}

	switch fieldType {
	case string(constants.SMALLINT):
		if val < math.MinInt16 || val > math.MaxInt16 {
			return nil, fmt.Errorf("SmallInt default value %d is out of range [%d, %d]", val, math.MinInt16, math.MaxInt16)
		}
	case string(constants.INT):
		if val < math.MinInt32 || val > math.MaxInt32 {
			return nil, fmt.Errorf("int default value %d is out of range [%d, %d]", val, math.MinInt32, math.MaxInt32)
		}
	}

	return &DefaultValue{
		Value:    val,
		Type:     "literal",
		DataType: fieldType,
		IsArray:  false,
	}, nil
}

func (dv *DefaultValidator) validateFloatDefault(defaultStr string, fieldType string) (*DefaultValue, error) {
	cleanValue := defaultStr
	if dv.stringPattern.MatchString(defaultStr) {
		cleanValue = defaultStr[1 : len(defaultStr)-1]
	}

	val, err := strconv.ParseFloat(strings.TrimSpace(cleanValue), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid float default value '%s' for type %s: %v", defaultStr, fieldType, err)
	}

	if math.IsInf(val, 0) {
		return nil, fmt.Errorf("infinite float values are not allowed as default: %s", defaultStr)
	}
	if math.IsNaN(val) {
		return nil, fmt.Errorf("NaN values are not allowed as default: %s", defaultStr)
	}

	return &DefaultValue{
		Value:    val,
		Type:     "literal",
		DataType: fieldType,
		IsArray:  false,
	}, nil
}

func (dv *DefaultValidator) validateStringDefault(defaultStr string, fieldType string) (*DefaultValue, error) {
	if dv.stringPattern.MatchString(defaultStr) {
		unquoted := defaultStr[1 : len(defaultStr)-1]
		unescaped := strings.ReplaceAll(unquoted, `\"`, `"`)
		unescaped = strings.ReplaceAll(unescaped, `\'`, `'`)
		unescaped = strings.ReplaceAll(unescaped, `\\`, `\`)
		return &DefaultValue{
			Value:    unescaped,
			Type:     "literal",
			DataType: fieldType,
			IsArray:  false,
		}, nil
	}

	return &DefaultValue{
		Value:    defaultStr,
		Type:     "literal",
		DataType: fieldType,
		IsArray:  false,
	}, nil
}

func (dv *DefaultValidator) validateBooleanDefault(defaultStr string, fieldType string) (*DefaultValue, error) {
	cleanValue := defaultStr
	if dv.stringPattern.MatchString(defaultStr) {
		cleanValue = defaultStr[1 : len(defaultStr)-1]
	}

	lowerValue := strings.ToLower(strings.TrimSpace(cleanValue))

	switch lowerValue {
	case "true", "1", "yes", "on":
		return &DefaultValue{
			Value:    true,
			Type:     "literal",
			DataType: fieldType,
			IsArray:  false,
		}, nil
	case "false", "0", "no", "off":
		return &DefaultValue{
			Value:    false,
			Type:     "literal",
			DataType: fieldType,
			IsArray:  false,
		}, nil
	default:
		return nil, fmt.Errorf("invalid boolean default value '%s'. Valid values: true, false, 1, 0, yes, no, on, off", defaultStr)
	}
}

func (dv *DefaultValidator) validateCharDefault(defaultStr string, fieldType string) (*DefaultValue, error) {
	cleanValue := defaultStr
	if dv.stringPattern.MatchString(defaultStr) {
		cleanValue = defaultStr[1 : len(defaultStr)-1]
	}

	if strings.HasPrefix(cleanValue, "\\") && len(cleanValue) == 2 {
		switch cleanValue[1] {
		case 'n':
			cleanValue = "\n"
		case 't':
			cleanValue = "\t"
		case 'r':
			cleanValue = "\r"
		case '\\':
			cleanValue = "\\"
		case '\'':
			cleanValue = "'"
		case '"':
			cleanValue = "\""
		default:
			return nil, fmt.Errorf("invalid escape sequence in char default: %s", cleanValue)
		}
	}

	if len([]rune(cleanValue)) != 1 {
		return nil, fmt.Errorf("char default value must be exactly one character, got '%s' (length: %d)",
			cleanValue, len([]rune(cleanValue)))
	}

	return &DefaultValue{
		Value:    cleanValue,
		Type:     "literal",
		DataType: fieldType,
		IsArray:  false,
	}, nil
}

func (dv *DefaultValidator) validateDateDefault(defaultStr string, fieldType string) (*DefaultValue, error) {
	cleanValue := defaultStr
	if dv.stringPattern.MatchString(defaultStr) {
		cleanValue = defaultStr[1 : len(defaultStr)-1]
	}

	cleanValue = strings.TrimSpace(cleanValue)

	dateFormats := []string{
		"2006-01-02",
		"01/02/2006",
		"02-01-2006",
		"2006/01/02",
	}

	var parsedTime time.Time
	var parseErr error

	for _, format := range dateFormats {
		if parsedTime, parseErr = time.Parse(format, cleanValue); parseErr == nil {
			standardFormat := parsedTime.Format("2006-01-02")
			return &DefaultValue{
				Value:    standardFormat,
				Type:     "literal",
				DataType: fieldType,
				IsArray:  false,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid date default value '%s'. Supported formats: YYYY-MM-DD, MM/DD/YYYY, DD-MM-YYYY, YYYY/MM/DD",
		defaultStr)
}

func (dv *DefaultValidator) validateTimestampDefault(defaultStr string, fieldType string) (*DefaultValue, error) {
	cleanValue := defaultStr
	if dv.stringPattern.MatchString(defaultStr) {
		cleanValue = defaultStr[1 : len(defaultStr)-1]
	}

	cleanValue = strings.TrimSpace(cleanValue)

	timestampFormats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.000",
		"2006-01-02T15:04:05.000",
	}

	for _, format := range timestampFormats {
		if _, err := time.Parse(format, cleanValue); err == nil {
			return &DefaultValue{
				Value:    cleanValue,
				Type:     "literal",
				DataType: fieldType,
				IsArray:  false,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid timestamp default value '%s'. Expected formats: RFC3339, YYYY-MM-DD HH:MM:SS, or YYYY-MM-DDTHH:MM:SS",
		defaultStr)
}

func (dv *DefaultValidator) validateJsonDefault(defaultStr string, fieldType string) (*DefaultValue, error) {
	cleanValue := defaultStr
	if dv.stringPattern.MatchString(defaultStr) {
		cleanValue = defaultStr[1 : len(defaultStr)-1]
	}

	trimmed := strings.TrimSpace(cleanValue)

	var js interface{}
	if err := json.Unmarshal([]byte(trimmed), &js); err != nil {
		return nil, fmt.Errorf("invalid JSON default value '%s': %v", defaultStr, err)
	}

	formatted, err := json.Marshal(js)
	if err != nil {
		return nil, fmt.Errorf("failed to format JSON default value '%s': %v", defaultStr, err)
	}

	return &DefaultValue{
		Value:    string(formatted),
		Type:     "literal",
		DataType: fieldType,
		IsArray:  false,
	}, nil
}

func (dv *DefaultValidator) validateBytesDefault(defaultStr string, fieldType string) (*DefaultValue, error) {
	cleanValue := strings.TrimSpace(defaultStr)
	if dv.stringPattern.MatchString(cleanValue) {
		cleanValue = cleanValue[1 : len(cleanValue)-1]
	}

	trimmed := strings.TrimSpace(cleanValue)
	var bytes []byte
	var err error

	if strings.HasPrefix(trimmed, "0x") || strings.HasPrefix(trimmed, "0X") {
		hexStr := trimmed[2:]
		if len(hexStr)%2 != 0 {
			return nil, fmt.Errorf("hex string must have even length: %s", trimmed)
		}
		bytes, err = hex.DecodeString(hexStr)
		if err != nil {
			return nil, fmt.Errorf("invalid hex default value '%s': %v", defaultStr, err)
		}
	} else {
		bytes, err = base64.StdEncoding.DecodeString(trimmed)
		if err != nil {
			bytes, err = base64.URLEncoding.DecodeString(trimmed)
			if err != nil {
				return nil, fmt.Errorf("invalid base64 default value '%s': %v", defaultStr, err)
			}
		}
	}

	sqlLiteral := fmt.Sprintf("\\x%x", bytes)

	return &DefaultValue{
		Value:    sqlLiteral,
		Type:     "literal",
		DataType: fieldType,
		IsArray:  false,
	}, nil
}

func (dv *DefaultValidator) isEnumType(typeName string) bool {
	_, exists := dv.enums[typeName]
	return exists
}

func (dv *DefaultValidator) GetEnum(enumName string) (*enum.Enum, bool) {
	e, exists := dv.enums[enumName]
	return e, exists
}

func (dv *DefaultValidator) IsValidCallback(callback string) bool {
	return constants.IsValidCallback(callback)
}

func (dv *DefaultValidator) GetSupportedCallbacks() []string {
	return constants.ValidCallbacks
}

func (dv *DefaultValidator) ValidateEnumExists(enumName string) error {
	if _, exists := dv.enums[enumName]; !exists {
		return fmt.Errorf("enum '%s' is not registered", enumName)
	}
	return nil
}

func (dv *DefaultValidator) ValidateScalarType(fieldType string) error {
	if !constants.IsScalarType(fieldType) {
		return fmt.Errorf("invalid scalar type: %s", fieldType)
	}
	return nil
}

func (dv *DefaultValidator) GetScalarTypeByName(typeName string) (constants.ScalarType, error) {
	scalarType, exists := constants.GetScalarType(typeName)
	if !exists {
		return "", fmt.Errorf("unknown scalar type: %s", typeName)
	}
	return scalarType, nil
}

func (dv *DefaultValue) String() string {
	if dv.Type == "callback" {
		return dv.Value.(string)
	}
	if dv.IsArray {
		if arr, ok := dv.Value.([]interface{}); ok {
			var elements []string
			for _, elem := range arr {
				if str, ok := elem.(string); ok && dv.DataType != "String[]" {
					if strings.ContainsAny(str, " ,[]{}()") {
						elements = append(elements, fmt.Sprintf(`"%s"`, str))
					} else {
						elements = append(elements, str)
					}
				} else {
					elements = append(elements, fmt.Sprintf("%v", elem))
				}
			}
			return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
		}
	}
	return fmt.Sprintf("%v", dv.Value)
}

func (dv *DefaultValue) IsCallback() bool {
	return dv.Type == "callback"
}

func (dv *DefaultValue) IsEnum() bool {
	return dv.Type == "enum"
}

func (dv *DefaultValue) IsLiteral() bool {
	return dv.Type == "literal"
}

func (dv *DefaultValue) GetArrayElements() ([]interface{}, bool) {
	if !dv.IsArray {
		return nil, false
	}
	if arr, ok := dv.Value.([]interface{}); ok {
		return arr, true
	}
	return nil, false
}

func (dv *DefaultValue) GetArrayLength() int {
	if arr, ok := dv.GetArrayElements(); ok {
		return len(arr)
	}
	return 0
}

func (dv *DefaultValue) IsEmpty() bool {
	if dv.IsArray {
		return dv.GetArrayLength() == 0
	}
	return dv.Value == nil || dv.Value == ""
}

func (dv *DefaultValue) GetValue() interface{} {
	return dv.Value
}

func (dv *DefaultValue) GetDataType() string {
	return dv.DataType
}

func (dv *DefaultValue) GetType() string {
	return dv.Type
}