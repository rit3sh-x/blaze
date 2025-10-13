package shadow

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/rit3sh-x/blaze/core/constants"
)

type ShadowSchema struct {
	enumsStr   string
	classesStr string
}

type ParsedEnum struct {
	Name   string
	Values []string
}

type ParsedTable struct {
	Name        string
	Columns     []*ParsedColumn
	PrimaryKeys []string
	Uniques     [][]string
	Checks      []string
	ForeignKeys []*ParsedForeignKey
}

type ParsedColumn struct {
	Name         string
	Type         string
	IsArray      bool
	IsOptional   bool
	DefaultValue string
	IsUnique     bool
	IsPrimaryKey bool
}

type ParsedForeignKey struct {
	FromColumns []string
	ToTable     string
	ToColumns   []string
	OnDelete    string
	OnUpdate    string
}

type ParsedIndex struct {
	Name    string
	Table   string
	Columns []string
	Type    string
	IsText  bool
}

type SQLParser struct {
	createTableRegex     *regexp.Regexp
	createEnumRegex      *regexp.Regexp
	alterEnumRegex       *regexp.Regexp
	dropTableRegex       *regexp.Regexp
	dropEnumRegex        *regexp.Regexp
	alterTableRegex      *regexp.Regexp
	addColumnRegex       *regexp.Regexp
	dropColumnRegex      *regexp.Regexp
	columnDefRegex       *regexp.Regexp
	constraintRegex      *regexp.Regexp
	indexRegex           *regexp.Regexp
	dropIndexRegex       *regexp.Regexp
	foreignKeyRegex      *regexp.Regexp
	checkConstraintRegex *regexp.Regexp
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
		columnDefRegex:       regexp.MustCompile(`"([^"]+)"\s+([A-Z][A-Z0-9_\(\)]+(?:\[\])?)\s*([^,]*?)(?:,|$)`),
		constraintRegex:      regexp.MustCompile(`(?:CONSTRAINT\s+\w+\s+)?(PRIMARY\s+KEY|UNIQUE|CHECK|FOREIGN\s+KEY)\s*\([^)]+\)(?:\s+REFERENCES[^,]*)?`),
		indexRegex:           regexp.MustCompile(`CREATE\s+INDEX\s+(\w+)\s+ON\s+"([^"]+)"\s*(?:USING\s+(\w+))?\s*\(\s*([^)]+)\s*\)`),
		dropIndexRegex:       regexp.MustCompile(`DROP\s+INDEX\s+(?:IF\s+EXISTS\s+)?(\w+)`),
		foreignKeyRegex:      regexp.MustCompile(`FOREIGN\s+KEY\s*\(\s*([^)]+)\s*\)\s+REFERENCES\s+"([^"]+)"\s*\(\s*([^)]+)\s*\)(?:\s+ON\s+DELETE\s+(\w+(?:\s+\w+)?))?(?:\s+ON\s+UPDATE\s+(\w+(?:\s+\w+)?))?`),
		checkConstraintRegex: regexp.MustCompile(`CHECK\s*\(\s*([^)]+)\s*\)`),
	}
}

func (p *SQLParser) ApplyMigrationToSchema(currentEnumsStr, currentClassesStr, migrationSQL string) (ShadowSchema, error) {
	currentEnums := p.parseEnumsFromString(currentEnumsStr)
	currentTables := p.parseClassesFromString(currentClassesStr)
	currentIndexes := make(map[string]*ParsedIndex)

	statements := p.splitSQLStatements(migrationSQL)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if err := p.applyStatement(stmt, currentEnums, currentTables, currentIndexes); err != nil {
			continue
		}
	}

	enumsStr := p.buildEnumsString(currentEnums)
	classesStr := p.buildClassesString(currentTables, currentIndexes)

	return ShadowSchema{
		enumsStr:   enumsStr,
		classesStr: classesStr,
	}, nil
}

func (p *SQLParser) parseEnumsFromString(enumsStr string) map[string]*ParsedEnum {
	enums := make(map[string]*ParsedEnum)

	if enumsStr == "" {
		return enums
	}

	enumBlocks := strings.Split(enumsStr, fmt.Sprintf(`\n\n%s `, constants.KEYWORD_ENUM))
	if len(enumBlocks) > 0 && strings.HasPrefix(enumBlocks[0], fmt.Sprintf(`%s `, constants.KEYWORD_ENUM)) {
		enumBlocks[0] = strings.TrimPrefix(enumBlocks[0], fmt.Sprintf(`%s `, constants.KEYWORD_ENUM))
	}

	for _, block := range enumBlocks {
		if block == "" {
			continue
		}

		if !strings.HasPrefix(block, fmt.Sprintf(`%s `, constants.KEYWORD_ENUM)) {
			block = fmt.Sprintf(`%s `, constants.KEYWORD_ENUM) + block
		}

		enum := p.parseEnumFromBlock(block)
		if enum != nil {
			enums[enum.Name] = enum
		}
	}

	return enums
}

func (p *SQLParser) parseClassesFromString(classesStr string) map[string]*ParsedTable {
	tables := make(map[string]*ParsedTable)

	if classesStr == "" {
		return tables
	}

	classBlocks := strings.Split(classesStr, fmt.Sprintf(`\n\n%s `, constants.KEYWORD_CLASS))
	if len(classBlocks) > 0 && strings.HasPrefix(classBlocks[0], fmt.Sprintf(`%s `, constants.KEYWORD_CLASS)) {
		classBlocks[0] = strings.TrimPrefix(classBlocks[0], fmt.Sprintf(`%s `, constants.KEYWORD_CLASS))
	}

	for _, block := range classBlocks {
		if block == "" {
			continue
		}

		if !strings.HasPrefix(block, fmt.Sprintf(`%s `, constants.KEYWORD_CLASS)) {
			block = fmt.Sprintf(`%s `, constants.KEYWORD_CLASS) + block
		}

		table := p.parseTableFromBlock(block)
		if table != nil {
			tables[table.Name] = table
		}
	}

	return tables
}

func (p *SQLParser) parseEnumFromBlock(block string) *ParsedEnum {
	lines := strings.Split(block, "\n")
	if len(lines) == 0 {
		return nil
	}

	headerPattern := regexp.MustCompile(fmt.Sprintf(`%s\s+(\w+)\s*\{`, constants.KEYWORD_ENUM))
	matches := headerPattern.FindStringSubmatch(lines[0])
	if matches == nil {
		return nil
	}

	enumName := matches[1]
	var values []string

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "}" || line == "" {
			continue
		}
		values = append(values, line)
	}

	return &ParsedEnum{
		Name:   enumName,
		Values: values,
	}
}

func (p *SQLParser) parseTableFromBlock(block string) *ParsedTable {
	lines := strings.Split(block, "\n")
	if len(lines) == 0 {
		return nil
	}

	headerPattern := regexp.MustCompile(fmt.Sprintf(`%s\s+(\w+)\s*\{`, constants.KEYWORD_CLASS))
	matches := headerPattern.FindStringSubmatch(lines[0])
	if matches == nil {
		return nil
	}

	tableName := matches[1]
	table := &ParsedTable{
		Name:        tableName,
		Columns:     []*ParsedColumn{},
		PrimaryKeys: []string{},
		Uniques:     [][]string{},
		Checks:      []string{},
		ForeignKeys: []*ParsedForeignKey{},
	}

	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "}" || line == "" {
			continue
		}

		if strings.HasPrefix(line, "@@") {
			p.parseClassDirectiveFromLine(line, table)
		} else {
			column := p.parseFieldFromLine(line)
			if column != nil {
				table.Columns = append(table.Columns, column)
			}
		}
	}

	return table
}

func (p *SQLParser) parseFieldFromLine(line string) *ParsedColumn {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	fieldName := parts[0]
	fieldType := parts[1]

	isArray := strings.HasSuffix(fieldType, "[]")
	if isArray {
		fieldType = strings.TrimSuffix(fieldType, "[]")
	}

	isOptional := strings.HasSuffix(fieldType, "?")
	if isOptional {
		fieldType = strings.TrimSuffix(fieldType, "?")
	}

	column := &ParsedColumn{
		Name:       fieldName,
		Type:       fieldType,
		IsArray:    isArray,
		IsOptional: isOptional,
	}

	for i := 2; i < len(parts); i++ {
		directive := parts[i]
		if strings.HasPrefix(directive, "@") {
			p.parseFieldDirective(directive, column)
		}
	}

	return column
}

func (p *SQLParser) parseFieldDirective(directive string, column *ParsedColumn) {
	switch {
	case directive == fmt.Sprintf("@%s", constants.FIELD_ATTR_PRIMARY_KEY):
		column.IsPrimaryKey = true
	case directive == fmt.Sprintf("@%s", constants.FIELD_ATTR_UNIQUE):
		column.IsUnique = true
	case strings.HasPrefix(directive, fmt.Sprintf("@%s(", constants.FIELD_ATTR_DEFAULT)):
		defaultVal := strings.TrimPrefix(directive, fmt.Sprintf("@%s(", constants.FIELD_ATTR_DEFAULT))
		defaultVal = strings.TrimSuffix(defaultVal, ")")
		column.DefaultValue = defaultVal
	}
}

func (p *SQLParser) parseClassDirectiveFromLine(line string, table *ParsedTable) {
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, fmt.Sprintf("@@%s(", constants.CLASS_ATTR_PRIMARY_KEY)) {
		columnList := strings.TrimPrefix(line, fmt.Sprintf("@@%s(", constants.CLASS_ATTR_PRIMARY_KEY))
		columnList = strings.TrimSuffix(columnList, ")")
		columnList = strings.Trim(columnList, "[]")
		columns := strings.Split(columnList, ",")
		for i, col := range columns {
			columns[i] = strings.TrimSpace(col)
		}
		table.PrimaryKeys = columns
	} else if strings.HasPrefix(line, fmt.Sprintf("@@%s(", constants.CLASS_ATTR_UNIQUE)) {
		columnList := strings.TrimPrefix(line, fmt.Sprintf("@@%s(", constants.CLASS_ATTR_UNIQUE))
		columnList = strings.TrimSuffix(columnList, ")")
		columnList = strings.Trim(columnList, "[]")
		columns := strings.Split(columnList, ",")
		for i, col := range columns {
			columns[i] = strings.TrimSpace(col)
		}
		table.Uniques = append(table.Uniques, columns)
	} else if strings.HasPrefix(line, "@@check(") {
		checkExpr := strings.TrimPrefix(line, "@@check(")
		checkExpr = strings.TrimSuffix(checkExpr, ")")
		checkExpr = strings.Trim(checkExpr, `"`)
		table.Checks = append(table.Checks, checkExpr)
	}
}

func (p *SQLParser) applyStatement(stmt string, enums map[string]*ParsedEnum, tables map[string]*ParsedTable, indexes map[string]*ParsedIndex) error {
	stmt = strings.TrimSpace(stmt)

	if matches := p.createEnumRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyCreateEnum(matches, enums)
	}

	if matches := p.alterEnumRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyAlterEnum(matches, enums)
	}

	if matches := p.dropEnumRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyDropEnum(matches, enums)
	}

	if matches := p.createTableRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyCreateTable(matches, tables)
	}

	if matches := p.dropTableRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyDropTable(matches, tables)
	}

	if matches := p.alterTableRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyAlterTable(matches, tables)
	}

	if matches := p.indexRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyCreateIndex(matches, indexes)
	}

	if matches := p.dropIndexRegex.FindStringSubmatch(stmt); matches != nil {
		return p.applyDropIndex(matches, indexes)
	}

	return nil
}

func (p *SQLParser) applyCreateEnum(matches []string, enums map[string]*ParsedEnum) error {
	enumName := matches[1]
	valuesStr := matches[2]

	valuePattern := regexp.MustCompile(`'([^']*)'`)
	valueMatches := valuePattern.FindAllStringSubmatch(valuesStr, -1)

	var values []string
	for _, match := range valueMatches {
		values = append(values, match[1])
	}

	enums[enumName] = &ParsedEnum{
		Name:   enumName,
		Values: values,
	}

	return nil
}

func (p *SQLParser) applyAlterEnum(matches []string, enums map[string]*ParsedEnum) error {
	enumName := matches[1]
	newValue := matches[2]

	if enum, exists := enums[enumName]; exists {
		for _, existingValue := range enum.Values {
			if existingValue == newValue {
				return nil
			}
		}
		enum.Values = append(enum.Values, newValue)
	}

	return nil
}

func (p *SQLParser) applyDropEnum(matches []string, enums map[string]*ParsedEnum) error {
	enumName := matches[1]
	delete(enums, enumName)
	return nil
}

func (p *SQLParser) applyCreateTable(matches []string, tables map[string]*ParsedTable) error {
	tableName := matches[1]
	tableContent := matches[2]

	table := &ParsedTable{
		Name:        tableName,
		Columns:     []*ParsedColumn{},
		PrimaryKeys: []string{},
		Uniques:     [][]string{},
		Checks:      []string{},
		ForeignKeys: []*ParsedForeignKey{},
	}

	parts := p.splitTableParts(tableContent)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if p.isConstraint(part) {
			p.parseConstraint(part, table)
		} else {
			p.parseColumn(part, table)
		}
	}

	tables[tableName] = table
	return nil
}

func (p *SQLParser) applyDropTable(matches []string, tables map[string]*ParsedTable) error {
	tableName := matches[1]
	delete(tables, tableName)
	return nil
}

func (p *SQLParser) applyAlterTable(matches []string, tables map[string]*ParsedTable) error {
	tableName := matches[1]
	alterAction := matches[2]

	table, exists := tables[tableName]
	if !exists {
		return fmt.Errorf("table %s does not exist", tableName)
	}

	if addMatches := p.addColumnRegex.FindStringSubmatch(alterAction); addMatches != nil {
		columnName := addMatches[1]
		columnType := addMatches[2]
		constraints := strings.TrimSpace(addMatches[3])

		column := &ParsedColumn{
			Name:         columnName,
			Type:         columnType,
			IsArray:      strings.HasSuffix(columnType, "[]"),
			IsOptional:   !strings.Contains(strings.ToUpper(constraints), "NOT NULL"),
			IsUnique:     strings.Contains(strings.ToUpper(constraints), "UNIQUE"),
			IsPrimaryKey: false,
		}

		if column.IsArray {
			column.Type = strings.TrimSuffix(column.Type, "[]")
		}

		defaultPattern := regexp.MustCompile(`DEFAULT\s+([^,\s]+(?:\s+[^,\s]+)*)`)
		if defaultMatches := defaultPattern.FindStringSubmatch(constraints); defaultMatches != nil {
			column.DefaultValue = strings.TrimSpace(defaultMatches[1])
		}

		table.Columns = append(table.Columns, column)
		return nil
	}

	if dropMatches := p.dropColumnRegex.FindStringSubmatch(alterAction); dropMatches != nil {
		columnName := dropMatches[1]

		for i, col := range table.Columns {
			if col.Name == columnName {
				table.Columns = append(table.Columns[:i], table.Columns[i+1:]...)
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

func (p *SQLParser) parseConstraint(part string, table *ParsedTable) {
	upperPart := strings.ToUpper(part)

	if strings.Contains(upperPart, "PRIMARY KEY") {
		pkPattern := regexp.MustCompile(`PRIMARY\s+KEY\s*\(\s*([^)]+)\s*\)`)
		if matches := pkPattern.FindStringSubmatch(part); matches != nil {
			columns := p.parseColumnList(matches[1])
			table.PrimaryKeys = columns
		}
	} else if strings.Contains(upperPart, "UNIQUE") && !strings.Contains(upperPart, "FOREIGN") {
		uniquePattern := regexp.MustCompile(`UNIQUE\s*\(\s*([^)]+)\s*\)`)
		if matches := uniquePattern.FindStringSubmatch(part); matches != nil {
			columns := p.parseColumnList(matches[1])
			table.Uniques = append(table.Uniques, columns)
		}
	} else if strings.Contains(upperPart, "CHECK") {
		if matches := p.checkConstraintRegex.FindStringSubmatch(part); matches != nil {
			table.Checks = append(table.Checks, matches[1])
		}
	} else if strings.Contains(upperPart, "FOREIGN KEY") {
		if matches := p.foreignKeyRegex.FindStringSubmatch(part); matches != nil {
			fk := &ParsedForeignKey{
				FromColumns: p.parseColumnList(matches[1]),
				ToTable:     matches[2],
				ToColumns:   p.parseColumnList(matches[3]),
				OnDelete:    p.mapSQLActionToConstant(matches[4]),
				OnUpdate:    p.mapSQLActionToConstant(matches[5]),
			}
			table.ForeignKeys = append(table.ForeignKeys, fk)
		}
	}
}

func (p *SQLParser) parseColumn(part string, table *ParsedTable) {
	columnPattern := regexp.MustCompile(`^"([^"]+)"\s+([A-Z][A-Z0-9_\(\)]+(?:\[\])?)\s*(.*)$`)
	matches := columnPattern.FindStringSubmatch(part)

	if matches == nil {
		return
	}

	columnName := matches[1]
	columnType := matches[2]
	constraints := strings.TrimSpace(matches[3])

	column := &ParsedColumn{
		Name:         columnName,
		Type:         columnType,
		IsArray:      strings.HasSuffix(columnType, "[]"),
		IsOptional:   !strings.Contains(strings.ToUpper(constraints), "NOT NULL"),
		IsUnique:     strings.Contains(strings.ToUpper(constraints), "UNIQUE"),
		IsPrimaryKey: false,
	}

	if column.IsArray {
		column.Type = strings.TrimSuffix(column.Type, "[]")
	}

	defaultPattern := regexp.MustCompile(`DEFAULT\s+([^,\s]+(?:\s+[^,\s]+)*)`)
	if defaultMatches := defaultPattern.FindStringSubmatch(constraints); defaultMatches != nil {
		column.DefaultValue = strings.TrimSpace(defaultMatches[1])
	}

	if strings.Contains(strings.ToUpper(constraints), "GENERATED BY DEFAULT AS IDENTITY") {
		column.DefaultValue = "autoincrement()"
	}

	table.Columns = append(table.Columns, column)
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

func (p *SQLParser) buildEnumsString(enums map[string]*ParsedEnum) string {
	if len(enums) == 0 {
		return ""
	}

	var enumStrings []string

	var enumNames []string
	for name := range enums {
		enumNames = append(enumNames, name)
	}
	sort.Strings(enumNames)

	for _, name := range enumNames {
		enum := enums[name]
		var builder strings.Builder

		builder.WriteString(fmt.Sprintf("%s %s {\n",constants.KEYWORD_ENUM, enum.Name))
		for _, value := range enum.Values {
			builder.WriteString(fmt.Sprintf("  %s\n", value))
		}
		builder.WriteString("}")

		enumStrings = append(enumStrings, builder.String())
	}

	return strings.Join(enumStrings, "\n\n")
}

func (p *SQLParser) buildClassesString(tables map[string]*ParsedTable, indexes map[string]*ParsedIndex) string {
	if len(tables) == 0 {
		return ""
	}

	var classStrings []string

	var tableNames []string
	for name := range tables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)

	for _, tableName := range tableNames {
		table := tables[tableName]
		classStr := p.buildClassString(table, indexes)
		if classStr != "" {
			classStrings = append(classStrings, classStr)
		}
	}

	return strings.Join(classStrings, "\n\n")
}

func (p *SQLParser) buildClassString(table *ParsedTable, indexes map[string]*ParsedIndex) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s %s {\n",constants.KEYWORD_CLASS, table.Name))

	pkSet := make(map[string]bool)
	for _, pk := range table.PrimaryKeys {
		pkSet[pk] = true
	}

	for _, column := range table.Columns {
		fieldStr := p.buildFieldString(column, table, pkSet)
		if fieldStr != "" {
			builder.WriteString(fmt.Sprintf("  %s\n", fieldStr))
		}
	}

	p.addClassDirectives(&builder, table, indexes)

	builder.WriteString("}")

	return builder.String()
}

func (p *SQLParser) buildFieldString(column *ParsedColumn, table *ParsedTable, pkSet map[string]bool) string {
	var parts []string

	fieldType := p.mapSQLTypeToSchemaType(column.Type)

	if column.IsArray {
		fieldType += "[]"
	}

	if column.IsOptional && !pkSet[column.Name] {
		fieldType += "?"
	}

	parts = append(parts, column.Name, fieldType)

	if pkSet[column.Name] && len(table.PrimaryKeys) == 1 {
		parts = append(parts, fmt.Sprintf("@%s", constants.FIELD_ATTR_PRIMARY_KEY))
	}

	if column.IsUnique && !pkSet[column.Name] {
		parts = append(parts, fmt.Sprintf("@%s", constants.FIELD_ATTR_UNIQUE))
	}

	if column.DefaultValue != "" {
		defaultVal := p.mapDefaultValueToSchema(column.DefaultValue)
		if defaultVal != "" {
			parts = append(parts, fmt.Sprintf("@%s(%s)",constants.FIELD_ATTR_DEFAULT , defaultVal))
		}
	}

	relationStr := p.buildRelationString(column, table)
	if relationStr != "" {
		parts = append(parts, relationStr)
	}

	return strings.Join(parts, " ")
}

func (p *SQLParser) buildRelationString(column *ParsedColumn, table *ParsedTable) string {
	for _, fk := range table.ForeignKeys {
		for i, fromCol := range fk.FromColumns {
			if fromCol == column.Name && i < len(fk.ToColumns) {
				var relationParts []string

				fromFields := fmt.Sprintf("[%s]", strings.Join(fk.FromColumns, ", "))
				toFields := fmt.Sprintf("[%s]", strings.Join(fk.ToColumns, ", "))
				relationParts = append(relationParts, fromFields, toFields)

				if fk.OnDelete != "" && fk.OnDelete != constants.ON_DELETE_NO_ACTION {
					relationParts = append(relationParts, fmt.Sprintf("onDelete: %s", fk.OnDelete))
				}

				if fk.OnUpdate != "" && fk.OnUpdate != constants.ON_UPDATE_NO_ACTION {
					relationParts = append(relationParts, fmt.Sprintf("onUpdate: %s", fk.OnUpdate))
				}

				return fmt.Sprintf("@%s(%s)", constants.FIELD_ATTR_RELATION, strings.Join(relationParts, ", "))
			}
		}
	}

	return ""
}

func (p *SQLParser) addClassDirectives(builder *strings.Builder, table *ParsedTable, indexes map[string]*ParsedIndex) {
	if len(table.PrimaryKeys) > 1 {
		pkFields := fmt.Sprintf("[%s]", strings.Join(table.PrimaryKeys, ", "))
		builder.WriteString(fmt.Sprintf("\n  @@%s(%s)", constants.CLASS_ATTR_PRIMARY_KEY, pkFields))
	}

	for _, unique := range table.Uniques {
		if len(unique) > 0 {
			uniqueFields := fmt.Sprintf("[%s]", strings.Join(unique, ", "))
			builder.WriteString(fmt.Sprintf("\n  @@%s(%s)", constants.CLASS_ATTR_UNIQUE, uniqueFields))
		}
	}

	for _, index := range indexes {
		if index.Table == table.Name {
			if index.IsText {
				indexFields := fmt.Sprintf("[%s]", strings.Join(index.Columns, ", "))
				builder.WriteString(fmt.Sprintf("\n  @@%s(%s)", constants.CLASS_ATTR_TEXT_INDEX, indexFields))
			} else {
				if !p.isSystemGeneratedIndex(index, table) {
					indexFields := fmt.Sprintf("[%s]", strings.Join(index.Columns, ", "))
					builder.WriteString(fmt.Sprintf("\n  @@%s(%s)", constants.CLASS_ATTR_INDEX, indexFields))
				}
			}
		}
	}

	for _, check := range table.Checks {
		builder.WriteString(fmt.Sprintf("\n  @@%s(\"%s\")", constants.CLASS_ATTR_CHECK, check))
	}
}

func (p *SQLParser) isSystemGeneratedIndex(index *ParsedIndex, table *ParsedTable) bool {
	if len(index.Columns) == len(table.PrimaryKeys) {
		pkSet := make(map[string]bool)
		for _, pk := range table.PrimaryKeys {
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

	expectedFKPattern := fmt.Sprintf("idx_%s_", strings.ToLower(table.Name))
	if strings.HasPrefix(strings.ToLower(index.Name), expectedFKPattern) {
		return true
	}

	for _, unique := range table.Uniques {
		if len(unique) == len(index.Columns) {
			match := true
			for i, col := range index.Columns {
				if i >= len(unique) || unique[i] != col {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}

	return false
}
