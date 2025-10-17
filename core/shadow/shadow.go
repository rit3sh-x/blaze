package shadow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast"
	"github.com/rit3sh-x/blaze/core/ast/class"
	"github.com/rit3sh-x/blaze/core/ast/class/attributes"
	"github.com/rit3sh-x/blaze/core/ast/class/directives"
	"github.com/rit3sh-x/blaze/core/ast/enum"
	"github.com/rit3sh-x/blaze/core/ast/field"
	fieldattributes "github.com/rit3sh-x/blaze/core/ast/field/attributes"
	"github.com/rit3sh-x/blaze/core/ast/field/defaults"
	fielddirectives "github.com/rit3sh-x/blaze/core/ast/field/directives"
	"github.com/rit3sh-x/blaze/core/ast/field/relations"
	"github.com/rit3sh-x/blaze/core/constants"
)

type SQLParser struct {
	createTableRegex     *regexp.Regexp
	createEnumRegex      *regexp.Regexp
	alterEnumRegex       *regexp.Regexp
	dropTableRegex       *regexp.Regexp
	dropEnumRegex        *regexp.Regexp
	alterTableRegex      *regexp.Regexp
	addColumnRegex       *regexp.Regexp
	dropColumnRegex      *regexp.Regexp
	indexRegex           *regexp.Regexp
	dropIndexRegex       *regexp.Regexp
	foreignKeyRegex      *regexp.Regexp
	checkConstraintRegex *regexp.Regexp
}

type ParsedIndex struct {
	Name    string
	Table   string
	Columns []string
	Type    string
	IsText  bool
}

func NewSQLParser() *SQLParser {
	return &SQLParser{
		createTableRegex:     regexp.MustCompile(`CREATE\s+TABLE\s+"([^"]+)"\s*\(\s*((?:[^;])*?)\s*\)`),
		createEnumRegex:      regexp.MustCompile(`CREATE\s+TYPE\s+"([^"]+)"\s+AS\s+ENUM\s*\(\s*([^)]+)\s*\)`),
		alterEnumRegex:       regexp.MustCompile(`ALTER\s+TYPE\s+"([^"]+)"\s+ADD\s+VALUE\s+'([^']+)'`),
		dropTableRegex:       regexp.MustCompile(`DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?"([^"]+)"`),
		dropEnumRegex:        regexp.MustCompile(`DROP\s+TYPE\s+(?:IF\s+EXISTS\s+)?"([^"]+)"`),
		alterTableRegex:      regexp.MustCompile(`ALTER\s+TABLE\s+"([^"]+)"\s+(.*)`),
		addColumnRegex:       regexp.MustCompile(`ADD\s+COLUMN\s+"([^"]+)"\s+([A-Z][A-Z0-9_\(\)]+(?:\[\])?)\s*(.*)`),
		dropColumnRegex:      regexp.MustCompile(`DROP\s+COLUMN\s+(?:IF\s+EXISTS\s+)?"([^"]+)"`),
		indexRegex:           regexp.MustCompile(`CREATE\s+INDEX\s+(\w+)\s+ON\s+"([^"]+)"\s*(?:USING\s+(\w+))?\s*\(\s*([^)]+)\s*\)`),
		dropIndexRegex:       regexp.MustCompile(`DROP\s+INDEX\s+(?:IF\s+EXISTS\s+)?(\w+)`),
		foreignKeyRegex:      regexp.MustCompile(`FOREIGN\s+KEY\s*\(\s*([^)]+)\s*\)\s+REFERENCES\s+"([^"]+)"\s*\(\s*([^)]+)\s*\)(?:\s+ON\s+DELETE\s+(\w+(?:\s+\w+)?))?(?:\s+ON\s+UPDATE\s+(\w+(?:\s+\w+)?))?`),
		checkConstraintRegex: regexp.MustCompile(`CHECK\s*\(\s*([^)]+)\s*\)`),
	}
}

func (p *SQLParser) ApplyMigrationToAST(currentAST *ast.SchemaAST, migrationSQL string) (*ast.SchemaAST, error) {
	if currentAST == nil {
		currentAST = &ast.SchemaAST{
			Enums:   make(map[string]*enum.Enum),
			Classes: []*class.Class{},
		}
	}

	newAST := &ast.SchemaAST{
		Enums:   make(map[string]*enum.Enum),
		Classes: make([]*class.Class, len(currentAST.Classes)),
	}

	for k, v := range currentAST.Enums {
		newAST.Enums[k] = p.copyEnum(v)
	}

	copy(newAST.Classes, currentAST.Classes)

	indexes := make(map[string]*ParsedIndex)
	statements := p.splitSQLStatements(migrationSQL)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if err := p.applyStatement(stmt, newAST, indexes); err != nil {
			continue
		}
	}

	p.applyIndexesToAST(newAST, indexes)

	return newAST, nil
}

func (p *SQLParser) copyEnum(e *enum.Enum) *enum.Enum {
	if e == nil {
		return nil
	}

	valuesCopy := make([]enum.EnumValue, len(e.Values))
	copy(valuesCopy, e.Values)

	return &enum.Enum{
		Name:     e.Name,
		Values:   valuesCopy,
		Position: e.Position,
	}
}

func (p *SQLParser) applyStatement(stmt string, ast *ast.SchemaAST, indexes map[string]*ParsedIndex) error {
	stmt = strings.TrimSpace(stmt)

	if matches := p.createEnumRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyCreateEnum(matches, ast)
	}

	if matches := p.alterEnumRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyAlterEnum(matches, ast)
	}

	if matches := p.dropEnumRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyDropEnum(matches, ast)
	}

	if matches := p.createTableRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyCreateTable(matches, ast)
	}

	if matches := p.dropTableRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyDropTable(matches, ast)
	}

	if matches := p.alterTableRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyAlterTable(matches, ast)
	}

	if matches := p.indexRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyCreateIndex(matches, indexes)
	}

	if matches := p.dropIndexRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyDropIndex(matches, indexes)
	}

	return nil
}

func (p *SQLParser) applyCreateEnum(matches []string, ast *ast.SchemaAST) error {
	enumName := matches[1]
	valuesStr := matches[2]

	if _, exists := ast.Enums[enumName]; exists {
		return nil
	}

	valuePattern := regexp.MustCompile(`'([^']*)'`)
	valueMatches := valuePattern.FindAllStringSubmatch(valuesStr, -1)

	var values []enum.EnumValue
	for i, match := range valueMatches {
		values = append(values, enum.EnumValue{
			Name:     match[1],
			Position: i,
		})
	}

	ast.Enums[enumName] = &enum.Enum{
		Name:     enumName,
		Values:   values,
		Position: len(ast.Enums),
	}

	return nil
}

func (p *SQLParser) applyAlterEnum(matches []string, ast *ast.SchemaAST) error {
	enumName := matches[1]
	newValue := matches[2]

	if enumDef, exists := ast.Enums[enumName]; exists {
		for _, existingValue := range enumDef.Values {
			if existingValue.Name == newValue {
				return nil
			}
		}

		newValues := make([]enum.EnumValue, len(enumDef.Values)+1)
		copy(newValues, enumDef.Values)
		newValues[len(enumDef.Values)] = enum.EnumValue{
			Name:     newValue,
			Position: len(enumDef.Values),
		}

		enumDef.Values = newValues
	}

	return nil
}

func (p *SQLParser) applyDropEnum(matches []string, ast *ast.SchemaAST) error {
	enumName := matches[1]
	delete(ast.Enums, enumName)
	return nil
}

func (p *SQLParser) applyCreateTable(matches []string, ast *ast.SchemaAST) error {
	tableName := matches[1]

	for _, existingClass := range ast.Classes {
		if existingClass.Name == tableName {
			return nil
		}
	}

	tableContent := matches[2]
	parts := p.splitTableParts(tableContent)

	var fields []*field.Field
	var classDirectives []*directives.ClassDirective
	var foreignKeys []ForeignKeyInfo
	var primaryKeys []string
	var uniques [][]string
	var checks []string

	fieldPosition := 0

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if p.isConstraint(part) {
			p.parseConstraintForAST(part, &primaryKeys, &uniques, &checks, &foreignKeys)
		} else {
			parsedField, err := p.parseColumnForAST(part, ast.Enums, fieldPosition)
			if err != nil {
				continue
			}
			fields = append(fields, parsedField)
			fieldPosition++
		}
	}

	if len(primaryKeys) == 1 {
		for _, f := range fields {
			if f.GetName() == primaryKeys[0] {
				f.AttributeDefinition.Directives = append(f.AttributeDefinition.Directives, &fielddirectives.FieldDirective{
					Name: constants.FIELD_ATTR_PRIMARY_KEY,
				})
				break
			}
		}
	} else if len(primaryKeys) > 1 {
		pkDirective := &directives.ClassDirective{
			Name:  constants.CLASS_ATTR_PRIMARY_KEY,
			Value: primaryKeys,
		}
		classDirectives = append(classDirectives, pkDirective)
	}

	for _, unique := range uniques {
		uniqueDirective := &directives.ClassDirective{
			Name:  constants.CLASS_ATTR_UNIQUE,
			Value: unique,
		}
		classDirectives = append(classDirectives, uniqueDirective)
	}

	for _, check := range checks {
		checkDirective := &directives.ClassDirective{
			Name:  constants.CLASS_ATTR_CHECK,
			Value: check,
		}
		classDirectives = append(classDirectives, checkDirective)
	}

	p.applyForeignKeysToFields(fields, foreignKeys)

	classAttrs := &attributes.ClassAttributes{
		Fields:     fields,
		Directives: classDirectives,
	}

	newClass := &class.Class{
		Name:       tableName,
		Attributes: classAttrs,
		Position:   len(ast.Classes),
	}

	ast.Classes = append(ast.Classes, newClass)
	return nil
}

type ForeignKeyInfo struct {
	FromColumns []string
	ToTable     string
	ToColumns   []string
	OnDelete    string
	OnUpdate    string
}

func (p *SQLParser) parseConstraintForAST(part string, primaryKeys *[]string, uniques *[][]string, checks *[]string, foreignKeys *[]ForeignKeyInfo) {
	upperPart := strings.ToUpper(part)

	if strings.Contains(upperPart, "PRIMARY KEY") {
		pkPattern := regexp.MustCompile(`PRIMARY\s+KEY\s*\(\s*([^)]+)\s*\)`)
		if matches := pkPattern.FindStringSubmatch(part); matches != nil {
			*primaryKeys = p.parseColumnList(matches[1])
		}
	} else if strings.Contains(upperPart, "UNIQUE") && !strings.Contains(upperPart, "FOREIGN") {
		uniquePattern := regexp.MustCompile(`UNIQUE\s*\(\s*([^)]+)\s*\)`)
		if matches := uniquePattern.FindStringSubmatch(part); matches != nil {
			*uniques = append(*uniques, p.parseColumnList(matches[1]))
		}
	} else if strings.Contains(upperPart, "CHECK") {
		if matches := p.checkConstraintRegex.FindStringSubmatch(part); matches != nil {
			*checks = append(*checks, matches[1])
		}
	} else if strings.Contains(upperPart, "FOREIGN KEY") {
		if matches := p.foreignKeyRegex.FindStringSubmatch(part); matches != nil {
			fk := ForeignKeyInfo{
				FromColumns: p.parseColumnList(matches[1]),
				ToTable:     matches[2],
				ToColumns:   p.parseColumnList(matches[3]),
				OnDelete:    p.mapSQLActionToConstant(matches[4]),
				OnUpdate:    p.mapSQLActionToConstant(matches[5]),
			}
			*foreignKeys = append(*foreignKeys, fk)
		}
	}
}

func (p *SQLParser) parseColumnForAST(part string, enums map[string]*enum.Enum, position int) (*field.Field, error) {
	columnPattern := regexp.MustCompile(`^"([^"]+)"\s+([A-Z][A-Z0-9_\(\)]+(?:\[\])?)\s*(.*)$`)
	matches := columnPattern.FindStringSubmatch(part)

	if matches == nil {
		return nil, fmt.Errorf("invalid column definition")
	}

	columnName := matches[1]
	columnType := matches[2]
	constraints := strings.TrimSpace(matches[3])

	isArray := strings.HasSuffix(columnType, "[]")
	if isArray {
		columnType = strings.TrimSuffix(columnType, "[]")
	}

	schemaType := p.mapSQLTypeToSchemaType(columnType)
	isOptional := !strings.Contains(strings.ToUpper(constraints), "NOT NULL")

	var fieldAttrs []*fieldattributes.Attribute
	var fieldDirectivesList []*fielddirectives.FieldDirective

	if strings.Contains(strings.ToUpper(constraints), "UNIQUE") {
		fieldDirectivesList = append(fieldDirectivesList, &fielddirectives.FieldDirective{
			Name: constants.FIELD_ATTR_UNIQUE,
		})
	}

	defaultPattern := regexp.MustCompile(`DEFAULT\s+([^,\s]+(?:\s+[^,\s]+)*)`)
	if defaultMatches := defaultPattern.FindStringSubmatch(constraints); defaultMatches != nil {
		defaultVal := strings.TrimSpace(defaultMatches[1])
		schemaDefault := p.mapDefaultValueToSchema(defaultVal)

		if schemaDefault != "" {
			validator := defaults.NewDefaultValidator(enums)
			defaultValue, err := validator.ValidateDefault(schemaDefault, schemaType, isArray)
			if err == nil {
				fieldAttrs = append(fieldAttrs, &fieldattributes.Attribute{
					Name:  constants.FIELD_ATTR_DEFAULT,
					Value: defaultValue,
				})
			}
		}
	}

	if strings.Contains(strings.ToUpper(constraints), "GENERATED BY DEFAULT AS IDENTITY") {
		validator := defaults.NewDefaultValidator(enums)
		defaultValue, err := validator.ValidateDefault(constants.DEFAULT_AUTOINCREMENT_CALLBACK, schemaType, false)
		if err == nil {
			fieldAttrs = append(fieldAttrs, &fieldattributes.Attribute{
				Name:  constants.FIELD_ATTR_DEFAULT,
				Value: defaultValue,
			})
		}
	}

	kind := p.determineFieldKind(schemaType, enums)

	attrDef := &fieldattributes.AttributeDefinition{
		Name:       columnName,
		DataType:   schemaType,
		Kind:       kind,
		IsArray:    isArray,
		IsOptional: isOptional,
		Attributes: fieldAttrs,
		Directives: fieldDirectivesList,
	}

	return &field.Field{
		AttributeDefinition: attrDef,
		Position:            position,
	}, nil
}

func (p *SQLParser) applyForeignKeysToFields(fields []*field.Field, foreignKeys []ForeignKeyInfo) {
	for _, fk := range foreignKeys {
		if len(fk.FromColumns) == 0 || len(fk.ToColumns) == 0 {
			continue
		}

		if len(fk.FromColumns) == 1 {
			for _, f := range fields {
				if f.GetName() == fk.FromColumns[0] {
					relationValidator := relations.NewRelationValidator()
					relation := &relations.Relation{
						From:      fk.FromColumns,
						To:        fk.ToColumns,
						ToClass:   fk.ToTable,
						FromClass: "",
						OnDelete:  fk.OnDelete,
						OnUpdate:  fk.OnUpdate,
					}

					if err := relationValidator.ValidateRelation(relation); err == nil {
						f.AttributeDefinition.Attributes = append(f.AttributeDefinition.Attributes, &fieldattributes.Attribute{
							Name:  constants.FIELD_ATTR_RELATION,
							Value: relation,
						})

						f.AttributeDefinition.DataType = fk.ToTable
						f.AttributeDefinition.Kind = constants.FIELD_KIND_OBJECT
					}
					break
				}
			}
		}
	}
}

func (p *SQLParser) applyIndexesToAST(ast *ast.SchemaAST, indexes map[string]*ParsedIndex) {
	for _, index := range indexes {
		for _, cls := range ast.Classes {
			if cls.Name == index.Table {
				directiveName := constants.CLASS_ATTR_INDEX
				if index.IsText {
					directiveName = constants.CLASS_ATTR_TEXT_INDEX
				}

				if !p.isSystemGeneratedIndexForAST(index, cls) {
					indexDirective := &directives.ClassDirective{
						Name:  directiveName,
						Value: index.Columns,
					}
					cls.Attributes.Directives = append(cls.Attributes.Directives, indexDirective)
				}
				break
			}
		}
	}
}

func (p *SQLParser) isSystemGeneratedIndexForAST(index *ParsedIndex, cls *class.Class) bool {
	pkFields := cls.GetPrimaryKeyFields()
	if len(index.Columns) == len(pkFields) && len(pkFields) > 0 {
		pkSet := make(map[string]bool)
		for _, pk := range pkFields {
			pkSet[pk] = true
		}

		allPK := true
		for _, col := range index.Columns {
			if !pkSet[col] {
				allPK = false
				break
			}
		}
		if allPK {
			return true
		}
	}

	return false
}

func (p *SQLParser) applyDropTable(matches []string, ast *ast.SchemaAST) error {
	tableName := matches[1]

	for i, cls := range ast.Classes {
		if cls.Name == tableName {
			ast.Classes = append(ast.Classes[:i], ast.Classes[i+1:]...)

			for j := i; j < len(ast.Classes); j++ {
				ast.Classes[j].Position = j
			}
			break
		}
	}

	return nil
}

func (p *SQLParser) applyAlterTable(matches []string, ast *ast.SchemaAST) error {
	tableName := matches[1]
	alterAction := matches[2]

	var targetClass *class.Class
	for _, cls := range ast.Classes {
		if cls.Name == tableName {
			targetClass = cls
			break
		}
	}

	if targetClass == nil {
		return fmt.Errorf("table %s does not exist", tableName)
	}

	if addMatches := p.addColumnRegex.FindStringSubmatch(alterAction); addMatches != nil {
		columnName := addMatches[1]

		for _, existingField := range targetClass.Attributes.Fields {
			if existingField.GetName() == columnName {
				return nil
			}
		}

		columnType := addMatches[2]
		constraints := strings.TrimSpace(addMatches[3])

		isArray := strings.HasSuffix(columnType, "[]")
		if isArray {
			columnType = strings.TrimSuffix(columnType, "[]")
		}

		schemaType := p.mapSQLTypeToSchemaType(columnType)
		isOptional := !strings.Contains(strings.ToUpper(constraints), "NOT NULL")

		kind := p.determineFieldKind(schemaType, ast.Enums)

		attrDef := &fieldattributes.AttributeDefinition{
			Name:       columnName,
			DataType:   schemaType,
			Kind:       kind,
			IsArray:    isArray,
			IsOptional: isOptional,
			Attributes: []*fieldattributes.Attribute{},
			Directives: []*fielddirectives.FieldDirective{},
		}

		newField := &field.Field{
			AttributeDefinition: attrDef,
			Position:            len(targetClass.Attributes.Fields),
		}

		targetClass.Attributes.Fields = append(targetClass.Attributes.Fields, newField)
		return nil
	}

	if dropMatches := p.dropColumnRegex.FindStringSubmatch(alterAction); dropMatches != nil {
		columnName := dropMatches[1]

		for i, f := range targetClass.Attributes.Fields {
			if f.GetName() == columnName {
				targetClass.Attributes.Fields = append(
					targetClass.Attributes.Fields[:i],
					targetClass.Attributes.Fields[i+1:]...,
				)

				for j := i; j < len(targetClass.Attributes.Fields); j++ {
					targetClass.Attributes.Fields[j].Position = j
				}
				break
			}
		}
		return nil
	}

	return nil
}

func (p *SQLParser) applyCreateIndex(matches []string, indexes map[string]*ParsedIndex) error {
	indexName := matches[1]
	tableName := matches[2]
	indexType := matches[3]
	columnsStr := matches[4]

	isTextIndex := strings.ToUpper(indexType) == "GIN" ||
		strings.Contains(strings.ToUpper(columnsStr), "GIN_TRGM_OPS") ||
		strings.Contains(columnsStr, "||")

	var columns []string
	if isTextIndex && strings.Contains(columnsStr, "||") {
		concatPattern := regexp.MustCompile(`"([^"]+)"`)
		columnMatches := concatPattern.FindAllStringSubmatch(columnsStr, -1)
		for _, match := range columnMatches {
			columns = append(columns, match[1])
		}
	} else {
		columns = p.parseColumnList(columnsStr)
	}

	indexes[indexName] = &ParsedIndex{
		Name:    indexName,
		Table:   tableName,
		Columns: columns,
		Type:    indexType,
		IsText:  isTextIndex,
	}

	return nil
}

func (p *SQLParser) applyDropIndex(matches []string, indexes map[string]*ParsedIndex) error {
	indexName := matches[1]
	delete(indexes, indexName)
	return nil
}

func (p *SQLParser) determineFieldKind(fieldType string, enums map[string]*enum.Enum) string {
	if constants.IsScalarType(fieldType) {
		return constants.FIELD_KIND_SCALAR
	}

	if _, exists := enums[fieldType]; exists {
		return constants.FIELD_KIND_ENUM
	}

	return constants.FIELD_KIND_OBJECT
}

func (p *SQLParser) splitSQLStatements(sqlContent string) []string {
	lines := strings.Split(sqlContent, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "--") {
			cleanLines = append(cleanLines, line)
		}
	}

	content := strings.Join(cleanLines, " ")

	var statements []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(content); i++ {
		char := content[i]

		if !inQuotes && (char == '\'' || char == '"') {
			inQuotes = true
			quoteChar = char
		} else if inQuotes && char == quoteChar {
			inQuotes = false
		} else if !inQuotes && char == ';' {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			continue
		}

		current.WriteByte(char)
	}

	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

func (p *SQLParser) splitTableParts(content string) []string {
	var parts []string
	var current strings.Builder
	parenLevel := 0
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(content); i++ {
		char := content[i]

		if !inQuotes && (char == '\'' || char == '"') {
			inQuotes = true
			quoteChar = char
		} else if inQuotes && char == quoteChar {
			inQuotes = false
		} else if !inQuotes {
			if char == '(' {
				parenLevel++
			} else if char == ')' {
				parenLevel--
			} else if char == ',' && parenLevel == 0 {
				part := strings.TrimSpace(current.String())
				if part != "" {
					parts = append(parts, part)
				}
				current.Reset()
				continue
			}
		}

		current.WriteByte(char)
	}

	part := strings.TrimSpace(current.String())
	if part != "" {
		parts = append(parts, part)
	}

	return parts
}

func (p *SQLParser) isConstraint(part string) bool {
	upperPart := strings.ToUpper(part)
	return strings.Contains(upperPart, "PRIMARY KEY") ||
		strings.Contains(upperPart, "UNIQUE") ||
		strings.Contains(upperPart, "CHECK") ||
		strings.Contains(upperPart, "FOREIGN KEY")
}

func (p *SQLParser) parseColumnList(columnList string) []string {
	var columns []string
	parts := strings.Split(columnList, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"`)
		if part != "" {
			columns = append(columns, part)
		}
	}

	return columns
}

func (p *SQLParser) mapSQLActionToConstant(action string) string {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "CASCADE":
		return constants.ON_DELETE_CASCADE
	case "RESTRICT":
		return constants.ON_DELETE_RESTRICT
	case "SET NULL":
		return constants.ON_DELETE_SET_NULL
	case "NO ACTION", "":
		return constants.ON_DELETE_NO_ACTION
	default:
		return constants.ON_DELETE_NO_ACTION
	}
}

func (p *SQLParser) mapSQLTypeToSchemaType(sqlType string) string {
	sqlType = strings.ToUpper(sqlType)

	switch {
	case strings.HasPrefix(sqlType, "INTEGER"):
		return string(constants.INT)
	case strings.HasPrefix(sqlType, "BIGINT"):
		return string(constants.BIGINT)
	case strings.HasPrefix(sqlType, "SMALLINT"):
		return string(constants.SMALLINT)
	case strings.HasPrefix(sqlType, "DOUBLE PRECISION"):
		return string(constants.FLOAT)
	case strings.HasPrefix(sqlType, "NUMERIC"):
		return string(constants.NUMERIC)
	case strings.HasPrefix(sqlType, "TEXT"), strings.HasPrefix(sqlType, "VARCHAR"):
		return string(constants.STRING)
	case strings.HasPrefix(sqlType, "UUID"):
		return string(constants.STRING)
	case strings.HasPrefix(sqlType, "BOOLEAN"):
		return string(constants.BOOLEAN)
	case strings.HasPrefix(sqlType, "DATE"):
		return string(constants.DATE)
	case strings.HasPrefix(sqlType, "TIMESTAMP"):
		return string(constants.TIMESTAMP)
	case strings.HasPrefix(sqlType, "JSONB"):
		return string(constants.JSON)
	case strings.HasPrefix(sqlType, "BYTEA"):
		return string(constants.BYTES)
	case strings.HasPrefix(sqlType, "CHAR(1)"):
		return string(constants.CHAR)
	default:
		return sqlType
	}
}

func (p *SQLParser) mapDefaultValueToSchema(defaultVal string) string {
	if defaultVal == "" {
		return ""
	}

	defaultVal = strings.TrimSpace(defaultVal)

	switch {
	case defaultVal == "CURRENT_TIMESTAMP(3)":
		return constants.DEFAULT_NOW_CALLBACK
	case defaultVal == "gen_random_uuid()":
		return constants.DEFAULT_UUID_CALLBACK
	case defaultVal == "autoincrement()":
		return constants.DEFAULT_AUTOINCREMENT_CALLBACK
	case strings.HasPrefix(defaultVal, "'") && strings.HasSuffix(defaultVal, "'"):
		return defaultVal
	default:
		return defaultVal
	}
}