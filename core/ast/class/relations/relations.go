package relations

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rit3sh-x/blaze/core/constants"
)

type Relation struct {
	From      []string
	FromClass string
	To        []string
	ToClass   string
	OnDelete  string
	OnUpdate  string
	Name      string
}

type RelationValidator struct {
	relationPattern  *regexp.Regexp
	fieldNamePattern *regexp.Regexp
	parameterPattern *regexp.Regexp
	compositePattern *regexp.Regexp
}

func NewRelationValidator() *RelationValidator {
	relationPattern := regexp.MustCompile(
		`^\s*\[\s*([^\]]+)\s*\]\s*,\s*\[\s*([^\]]+)\s*\]` +
			`(?:\s*,\s*([^,]+))?` +
			`(?:\s*,\s*([^,]+))?` +
			`(?:\s*,\s*([^,]+))?` +
			`\s*$`)
	fieldNamePattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(\.[a-zA-Z_][a-zA-Z0-9_]*)?$`)
	parameterPattern := regexp.MustCompile(`(\w+):\s*([a-zA-Z_]\w*|"[^"]*")`)
	compositePattern := regexp.MustCompile(`([^,]+)`)

	return &RelationValidator{
		relationPattern:  relationPattern,
		fieldNamePattern: fieldNamePattern,
		parameterPattern: parameterPattern,
		compositePattern: compositePattern,
	}
}

func (rv *RelationValidator) ValidateRelation(relation *Relation) error {
	if relation == nil {
		return fmt.Errorf("relation cannot be nil")
	}

	if len(relation.From) == 0 {
		return fmt.Errorf("relation must have at least one source field")
	}

	for i, field := range relation.From {
		if err := rv.validateFieldName(field); err != nil {
			return fmt.Errorf("invalid source field at index %d '%s': %v", i, field, err)
		}
	}

	if len(relation.To) == 0 {
		return fmt.Errorf("relation must have at least one target field")
	}

	for i, field := range relation.To {
		if err := rv.validateFieldName(field); err != nil {
			return fmt.Errorf("invalid target field at index %d '%s': %v", i, field, err)
		}
	}

	if len(relation.From) != len(relation.To) {
		return fmt.Errorf("number of source fields (%d) must match number of target fields (%d)",
			len(relation.From), len(relation.To))
	}

	if relation.OnDelete != "" {
		if !constants.IsValidOnDeleteAction(relation.OnDelete) {
			return fmt.Errorf("invalid onDelete action '%s'. Valid actions: %v",
				relation.OnDelete, constants.ValidOnDeleteActions)
		}
	} else {
		relation.OnDelete = constants.ON_DELETE_NO_ACTION
	}

	if relation.OnUpdate != "" {
		if !constants.IsValidOnUpdateAction(relation.OnUpdate) {
			return fmt.Errorf("invalid onUpdate action '%s'. Valid actions: %v",
				relation.OnUpdate, constants.ValidOnUpdateActions)
		}
	} else {
		relation.OnUpdate = constants.ON_UPDATE_NO_ACTION
	}

	if relation.Name != "" {
		if err := rv.validateRelationName(relation.Name); err != nil {
			return fmt.Errorf("invalid relation name '%s': %v", relation.Name, err)
		}
	}

	return nil
}

func (rv *RelationValidator) validateFieldName(fieldName string) error {
	fieldName = strings.TrimSpace(fieldName)
	if fieldName == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	if !rv.fieldNamePattern.MatchString(fieldName) {
		return fmt.Errorf("invalid field name format. Must start with letter/underscore, contain only alphanumeric characters/underscores, and optionally have one dot for nested references")
	}

	if strings.Contains(fieldName, ".") {
		parts := strings.Split(fieldName, ".")
		if len(parts) != 2 {
			return fmt.Errorf("nested field reference must have exactly one dot")
		}

		for i, part := range parts {
			if part == "" {
				return fmt.Errorf("field name part %d cannot be empty", i+1)
			}
			if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(part) {
				return fmt.Errorf("field name part '%s' is invalid", part)
			}
		}
	}

	return nil
}

func (rv *RelationValidator) validateRelationName(name string) error {
	if name == "" {
		return fmt.Errorf("relation name cannot be empty")
	}

	if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(name) {
		return fmt.Errorf("relation name must start with letter/underscore and contain only alphanumeric characters/underscores")
	}

	return nil
}

func (rv *RelationValidator) ParseRelationFromString(definition string) (*Relation, error) {
	if strings.TrimSpace(definition) == "" {
		return nil, fmt.Errorf("relation definition cannot be empty")
	}

	definition = strings.TrimSpace(definition)

	relation := &Relation{
		OnDelete: constants.ON_DELETE_NO_ACTION,
		OnUpdate: constants.ON_UPDATE_NO_ACTION,
	}

	matches := rv.relationPattern.FindStringSubmatch(definition)
	if len(matches) < 3 {
		return nil, fmt.Errorf("invalid relation syntax. Expected: [sourceFields], [targetFields]")
	}

	sourceFields := rv.parseFieldList(matches[1])
	if len(sourceFields) == 0 {
		return nil, fmt.Errorf("source fields cannot be empty")
	}
	relation.From = sourceFields

	targetFields := rv.parseFieldList(matches[2])
	if len(targetFields) == 0 {
		return nil, fmt.Errorf("target fields cannot be empty")
	}

	toClass, processedTargetFields, err := rv.extractAndValidateTargetClass(targetFields)
	if err != nil {
		return nil, fmt.Errorf("failed to process target fields: %v", err)
	}

	relation.ToClass = toClass
	relation.To = processedTargetFields

	if err := rv.parseRelationOptions(definition, relation); err != nil {
		return nil, fmt.Errorf("failed to parse relation options: %v", err)
	}

	if err := rv.ValidateRelation(relation); err != nil {
		return nil, fmt.Errorf("relation validation failed: %v", err)
	}

	return relation, nil
}

func (rv *RelationValidator) extractAndValidateTargetClass(targetFields []string) (string, []string, error) {
	if len(targetFields) == 0 {
		return "", nil, fmt.Errorf("target fields cannot be empty")
	}

	var toClass string
	var processedFields []string

	for i, field := range targetFields {
		field = strings.TrimSpace(field)

		if !strings.Contains(field, ".") {
			return "", nil, fmt.Errorf("target field at index %d '%s' must be in 'class.field' format", i, field)
		}

		parts := strings.Split(field, ".")
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("target field at index %d '%s' must have exactly one dot separating class and field", i, field)
		}

		fieldClass := strings.TrimSpace(parts[0])
		fieldName := strings.TrimSpace(parts[1])

		if fieldClass == "" || fieldName == "" {
			return "", nil, fmt.Errorf("target field at index %d '%s' has empty class or field name", i, field)
		}

		if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(fieldClass) {
			return "", nil, fmt.Errorf("target field at index %d has invalid class name '%s'", i, fieldClass)
		}

		if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(fieldName) {
			return "", nil, fmt.Errorf("target field at index %d has invalid field name '%s'", i, fieldName)
		}

		if i == 0 {
			toClass = fieldClass
		} else if toClass != fieldClass {
			return "", nil, fmt.Errorf("all target fields must belong to the same class. Found '%s' and '%s'", toClass, fieldClass)
		}

		processedFields = append(processedFields, fieldName)
	}

	return toClass, processedFields, nil
}

func (rv *RelationValidator) parseFieldList(fieldListStr string) []string {
	fieldListStr = strings.TrimSpace(fieldListStr)
	if fieldListStr == "" {
		return []string{}
	}

	rawFields := strings.Split(fieldListStr, ",")
	var result []string

	for _, field := range rawFields {
		field = strings.TrimSpace(field)
		if field != "" {
			result = append(result, field)
		}
	}

	return result
}

func (rv *RelationValidator) parseRelationOptions(definition string, relation *Relation) error {
	matches := rv.parameterPattern.FindAllStringSubmatch(definition, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		paramName := strings.TrimSpace(match[1])
		paramValue := strings.TrimSpace(match[2])

		if strings.HasPrefix(paramValue, "\"") && strings.HasSuffix(paramValue, "\"") {
			paramValue = paramValue[1 : len(paramValue)-1]
		}

		switch paramName {
		case "onDelete":
			relation.OnDelete = paramValue
		case "onUpdate":
			relation.OnUpdate = paramValue
		case "name":
			relation.Name = paramValue
		default:
			return fmt.Errorf("unknown relation parameter: %s", paramName)
		}
	}

	return nil
}

func (r *Relation) GetCardinality(isSourceArray, isTargetArray, isSourceOptional, isTargetOptional bool) string {
	if !isSourceArray && isTargetArray {
		return constants.RELATION_ONE_TO_MANY
	}

	if isSourceArray && !isTargetArray {
		return constants.RELATION_MANY_TO_ONE
	}

	if isSourceArray && isTargetArray {
		return constants.RELATION_MANY_TO_MANY
	}

	return constants.RELATION_ONE_TO_ONE
}

func (r *Relation) IsComposite() bool {
	return len(r.To) > 1 || len(r.From) > 1
}

func (r *Relation) HasOnDelete() bool {
	return r.OnDelete != "" && r.OnDelete != constants.ON_DELETE_NO_ACTION
}

func (r *Relation) HasOnUpdate() bool {
	return r.OnUpdate != "" && r.OnUpdate != constants.ON_UPDATE_NO_ACTION
}

func (r *Relation) HasName() bool {
	return r.Name != ""
}

func (r *Relation) IsValidForOptionalField() bool {
	if r.OnDelete == constants.ON_DELETE_SET_NULL || r.OnUpdate == constants.ON_UPDATE_SET_NULL {
		return true
	}
	return r.OnDelete != constants.ON_DELETE_SET_NULL && r.OnUpdate != constants.ON_UPDATE_SET_NULL
}

func (r *Relation) RequiresOptionalField() bool {
	return r.OnDelete == constants.ON_DELETE_SET_NULL || r.OnUpdate == constants.ON_UPDATE_SET_NULL
}

func (r *Relation) GetSourceFields() []string {
	return r.From
}

func (r *Relation) GetSourceField() string {
	if len(r.From) > 0 {
		return r.From[0]
	}
	return ""
}

func (r *Relation) GetTargetFields() []string {
	return r.To
}

func (r *Relation) GetTargetField() string {
	if len(r.To) > 0 {
		return r.To[0]
	}
	return ""
}

func (r *Relation) String() string {
	var parts []string

	var displayTargetFields []string
	for _, field := range r.To {
		if r.ToClass != "" {
			displayTargetFields = append(displayTargetFields, r.ToClass+"."+field)
		} else {
			displayTargetFields = append(displayTargetFields, field)
		}
	}

	parts = append(parts, fmt.Sprintf("[%s], [%s]",
		strings.Join(r.From, ", "),
		strings.Join(displayTargetFields, ", ")))

	if r.HasOnDelete() {
		parts = append(parts, fmt.Sprintf("onDelete: %s", r.OnDelete))
	}

	if r.HasOnUpdate() {
		parts = append(parts, fmt.Sprintf("onUpdate: %s", r.OnUpdate))
	}

	if r.HasName() {
		parts = append(parts, fmt.Sprintf("name: \"%s\"", r.Name))
	}

	return strings.Join(parts, ", ")
}

func (r *Relation) ValidateOneToOneRelation(isSourceOptional bool) error {
	if r.OnDelete == constants.ON_DELETE_SET_NULL && !isSourceOptional {
		return fmt.Errorf("onDelete: SetNull requires the source field to be optional")
	}

	return nil
}

func (r *Relation) ValidateOneToManyRelation() error {
	return nil
}

func (r *Relation) ValidateManyToManyRelation() error {
	return nil
}

func (r *Relation) ValidateCompositeRelation() error {
	if !r.IsComposite() {
		return fmt.Errorf("expected composite relation but got simple relation")
	}

	if len(r.From) != len(r.To) {
		return fmt.Errorf("composite relation must have matching number of source (%d) and target (%d) fields",
			len(r.From), len(r.To))
	}

	return nil
}

func (r *Relation) GetRelationDirection() string {
	return "forward"
}

func (r *Relation) IsBackReference() bool {
	return false
}

func (r *Relation) GetRelationKey() string {
	key := fmt.Sprintf("%s->%s",
		strings.Join(r.From, ","),
		strings.Join(r.To, ","))
	if r.HasName() {
		key += ":" + r.Name
	}
	return key
}
