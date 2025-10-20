package hooks

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast"
	"github.com/rit3sh-x/blaze/core/ast/class"
	"github.com/rit3sh-x/blaze/core/ast/field"
	"github.com/rit3sh-x/blaze/core/constants"
	"github.com/rit3sh-x/blaze/core/utils"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func GenerateFiller() string {
	return GenerateCoreTypes()
}

func GenerateModelPredicates(cls *class.Class, ast *ast.SchemaAST) string {
	return GenerateClassPredicates(cls, ast)
}

func GenerateCoreTypes() string {
	return `// ==================== CORE TYPES ====================

type Predicate func(interface{})

type Order struct {
    field string
    desc  bool
}

type AggregateResult struct {
    Count int
    Sum   int
    Avg   float64
    Min   interface{}
    Max   interface{}
}

type GroupValue struct {
    Key   interface{}
    Value int
}`
}

func GeneratePredicateVars(classNames []string) string {
	var res strings.Builder

	for _, name := range classNames {
		lowerName := cases.Lower(language.English).String(name)
		res.WriteString(fmt.Sprintf("type %s struct{}\n", lowerName))
	}
	res.WriteString("\n")

	for _, name := range classNames {
		lowerName := cases.Lower(language.English).String(name)
		res.WriteString(fmt.Sprintf("var %sModel = %s{}\n", name, lowerName))
	}

	return res.String()
}

func GenerateClassPredicates(cls *class.Class, ast *ast.SchemaAST) string {
	var res strings.Builder
	lowerClass := cases.Lower(language.English).String(cls.Name)

	for _, fld := range cls.Attributes.Fields {
		if fld.IsScalar() {
			scalarType := fld.GetBaseType()
			switch constants.ScalarType(scalarType) {
			case constants.STRING:
				if fld.AttributeDefinition.DefaultValue != nil && fld.AttributeDefinition.DefaultValue.Value == constants.DEFAULT_UUID_CALLBACK {
					res.WriteString(generateUUIDPredicates(lowerClass, fld, ast))
				} else {
					res.WriteString(generateStringPredicates(lowerClass, fld, ast))
				}
			case constants.CHAR:
				res.WriteString(generateStringPredicates(lowerClass, fld, ast))
			case constants.INT, constants.BIGINT, constants.SMALLINT:
				res.WriteString(generateIntegerPredicates(lowerClass, fld, ast))
			case constants.FLOAT, constants.NUMERIC:
				res.WriteString(generateFloatPredicates(lowerClass, fld, ast))
			case constants.BOOLEAN:
				res.WriteString(generateBooleanPredicates(lowerClass, fld, ast))
			case constants.DATE, constants.TIMESTAMP:
				res.WriteString(generateDateTimePredicates(lowerClass, fld, ast))
			case constants.JSON:
				res.WriteString(generateJsonPredicates(lowerClass, fld))
			case constants.BYTES:
				res.WriteString(generateBytesPredicates(lowerClass, fld, ast))
			}
		} else if fld.IsEnum() {
			res.WriteString(generateEnumPredicates(lowerClass, fld, ast))
		} else if fld.IsObject() {
			baseType := fld.GetBaseType()
			if utils.GetScalarType(baseType) == "" && ast.GetEnumByName(baseType) == nil {
				res.WriteString(generateRelationPredicates(lowerClass, fld))
			}
		}

		if fld.IsArray() {
			res.WriteString(generateArrayPredicates(lowerClass, fld))
		}
	}

	return res.String()
}

func generateUUIDPredicates(lowerClass string, fld *field.Field, ast *ast.SchemaAST) string {
	var res strings.Builder
	fieldName := utils.ToExportedName(fld.GetName())
	goType := strings.TrimPrefix(utils.GetGoType(fld, ast), "*")

	res.WriteString(fmt.Sprintf("func (%s) %sEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sNEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sIn(vs ...%s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sNotIn(vs ...%s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))

	if fld.IsOptional() {
		res.WriteString(fmt.Sprintf("func (%s) %sIsNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
		res.WriteString(fmt.Sprintf("func (%s) %sIsNotNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	}

	return res.String()
}

func generateStringPredicates(lowerClass string, fld *field.Field, ast *ast.SchemaAST) string {
	var res strings.Builder
	fieldName := utils.ToExportedName(fld.GetName())
	goType := strings.TrimPrefix(utils.GetGoType(fld, ast), "*")

	res.WriteString(fmt.Sprintf("func (%s) %sEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sNEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sContains(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sHasPrefix(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sHasSuffix(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sIn(vs ...%s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sNotIn(vs ...%s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))

	if fld.IsOptional() {
		res.WriteString(fmt.Sprintf("func (%s) %sIsNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
		res.WriteString(fmt.Sprintf("func (%s) %sIsNotNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	}

	return res.String()
}

func generateIntegerPredicates(lowerClass string, fld *field.Field, ast *ast.SchemaAST) string {
	var res strings.Builder
	fieldName := utils.ToExportedName(fld.GetName())
	goType := strings.TrimPrefix(utils.GetGoType(fld, ast), "*")

	res.WriteString(fmt.Sprintf("func (%s) %sEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sNEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sGT(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sGTE(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sLT(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sLTE(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sIn(vs ...%s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sNotIn(vs ...%s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))

	if fld.IsOptional() {
		res.WriteString(fmt.Sprintf("func (%s) %sIsNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
		res.WriteString(fmt.Sprintf("func (%s) %sIsNotNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	}

	return res.String()
}

func generateFloatPredicates(lowerClass string, fld *field.Field, ast *ast.SchemaAST) string {
	return generateIntegerPredicates(lowerClass, fld, ast)
}

func generateBooleanPredicates(lowerClass string, fld *field.Field, ast *ast.SchemaAST) string {
	var res strings.Builder
	fieldName := utils.ToExportedName(fld.GetName())
	goType := strings.TrimPrefix(utils.GetGoType(fld, ast), "*")

	res.WriteString(fmt.Sprintf("func (%s) %sEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sNEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))

	if fld.IsOptional() {
		res.WriteString(fmt.Sprintf("func (%s) %sIsNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
		res.WriteString(fmt.Sprintf("func (%s) %sIsNotNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	}

	return res.String()
}

func generateDateTimePredicates(lowerClass string, fld *field.Field, ast *ast.SchemaAST) string {
	return generateIntegerPredicates(lowerClass, fld, ast)
}

func generateEnumPredicates(lowerClass string, fld *field.Field, ast *ast.SchemaAST) string {
	var res strings.Builder
	fieldName := utils.ToExportedName(fld.GetName())
	goType := strings.TrimPrefix(utils.GetGoType(fld, ast), "*")

	res.WriteString(fmt.Sprintf("func (%s) %sEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sNEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sIn(vs ...%s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sNotIn(vs ...%s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))

	if fld.IsOptional() {
		res.WriteString(fmt.Sprintf("func (%s) %sIsNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
		res.WriteString(fmt.Sprintf("func (%s) %sIsNotNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	}

	return res.String()
}

func generateRelationPredicates(lowerClass string, fld *field.Field) string {
	var res strings.Builder
	fieldName := utils.ToExportedName(fld.GetName())

	res.WriteString(fmt.Sprintf("func (%s) Has%s() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	res.WriteString(fmt.Sprintf("func (%s) Has%sWith(preds ...Predicate) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))

	if fld.IsArray() {
		res.WriteString(fmt.Sprintf("func (%s) %sSome(preds ...Predicate) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
		res.WriteString(fmt.Sprintf("func (%s) %sEvery(preds ...Predicate) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
		res.WriteString(fmt.Sprintf("func (%s) %sNone(preds ...Predicate) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	}

	return res.String()
}

func generateJsonPredicates(lowerClass string, fld *field.Field) string {
	var res strings.Builder
	fieldName := utils.ToExportedName(fld.GetName())

	res.WriteString(fmt.Sprintf("func (%s) %sEQ(v interface{}) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	res.WriteString(fmt.Sprintf("func (%s) %sNEQ(v interface{}) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	res.WriteString(fmt.Sprintf("func (%s) %sContains(v interface{}) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	res.WriteString(fmt.Sprintf("func (%s) %sHasKey(key string) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))

	if fld.IsOptional() {
		res.WriteString(fmt.Sprintf("func (%s) %sIsNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
		res.WriteString(fmt.Sprintf("func (%s) %sIsNotNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	}

	return res.String()
}

func generateBytesPredicates(lowerClass string, fld *field.Field, ast *ast.SchemaAST) string {
	var res strings.Builder
	fieldName := utils.ToExportedName(fld.GetName())
	goType := strings.TrimPrefix(utils.GetGoType(fld, ast), "*")

	res.WriteString(fmt.Sprintf("func (%s) %sEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))
	res.WriteString(fmt.Sprintf("func (%s) %sNEQ(v %s) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName, goType))

	if fld.IsOptional() {
		res.WriteString(fmt.Sprintf("func (%s) %sIsNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
		res.WriteString(fmt.Sprintf("func (%s) %sIsNotNull() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	}

	return res.String()
}

func generateArrayPredicates(lowerClass string, fld *field.Field) string {
	var res strings.Builder
	fieldName := utils.ToExportedName(fld.GetName())

	res.WriteString(fmt.Sprintf("func (%s) %sIsEmpty() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	res.WriteString(fmt.Sprintf("func (%s) %sIsNotEmpty() Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	res.WriteString(fmt.Sprintf("func (%s) %sLengthEQ(n int) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	res.WriteString(fmt.Sprintf("func (%s) %sLengthGT(n int) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))
	res.WriteString(fmt.Sprintf("func (%s) %sLengthLT(n int) Predicate { return func(interface{}) {} }\n", lowerClass, fieldName))

	return res.String()
}

func GenerateCompositePredicates(cls *class.Class, ast *ast.SchemaAST) string {
	var res strings.Builder
	lowerClass := cases.Lower(language.English).String(cls.Name)

	for _, directive := range cls.Attributes.Directives {
		if directive.Name == constants.CLASS_ATTR_UNIQUE {
			fields, err := directive.GetFields()
			if err != nil || len(fields) < 2 {
				continue
			}

			sortedFields := make([]string, len(fields))
			copy(sortedFields, fields)
			sort.Strings(sortedFields)

			var params []string
			var predicateCalls []string
			var fieldNameParts []string

			for _, fieldName := range sortedFields {
				fld := cls.Attributes.GetFieldByName(fieldName)
				if fld == nil {
					continue
				}
				goType := strings.TrimPrefix(utils.GetGoType(fld, ast), "*")
				params = append(params, fmt.Sprintf("%s %s", fieldName, goType))
				predicateCalls = append(predicateCalls, fmt.Sprintf("%s{}.%sEQ(%s)", lowerClass, utils.ToExportedName(fieldName), fieldName))
				fieldNameParts = append(fieldNameParts, utils.ToExportedName(fieldName))
			}

			funcName := strings.Join(fieldNameParts, "")
			res.WriteString(fmt.Sprintf("func (%s) %sEQ(%s) Predicate {\n", lowerClass, funcName, strings.Join(params, ", ")))
			res.WriteString("\treturn And(\n")
			for _, call := range predicateCalls {
				res.WriteString(fmt.Sprintf("\t\t%s,\n", call))
			}
			res.WriteString("\t)\n")
			res.WriteString("}\n\n")
		}
	}

	return res.String()
}

func GenerateLogicalOperators() string {
	return `func And(predicates ...Predicate) Predicate {
    return func(interface{}) {}
}

func Or(predicates ...Predicate) Predicate {
    return func(interface{}) {}
}

func Not(p Predicate) Predicate {
    return func(interface{}) {}
}`
}

func GenerateOrderFunctions() string {
	return `type OrderFunc func(*[]*Order)

func Asc(field string) OrderFunc {
    return func(orders *[]*Order) {
        *orders = append(*orders, &Order{field: field, desc: false})
    }
}

func Desc(field string) OrderFunc {
    return func(orders *[]*Order) {
        *orders = append(*orders, &Order{field: field, desc: true})
    }
}`
}

func GenerateFieldConstants(ast *ast.SchemaAST) string {
	var res strings.Builder

	res.WriteString("// ==================== FIELD CONSTANTS ====================\n\n")

	for _, cls := range ast.Classes {
		structName := cls.Name + "FieldsType"
		varName := cls.Name + "Fields"

		res.WriteString(fmt.Sprintf("type %s struct{}\n\n", structName))
		res.WriteString(fmt.Sprintf("var %s = %s{}\n\n", varName, structName))

		for _, fld := range cls.Attributes.Fields {
			if fld.IsObject() {
				continue
			}
			fieldName := utils.ToExportedName(fld.GetName())
			res.WriteString(fmt.Sprintf("func (%s) %s() string {\n", structName, fieldName))
			res.WriteString(fmt.Sprintf("\treturn \"%s\"\n", fld.GetName()))
			res.WriteString("}\n\n")
		}
	}

	return res.String()
}

func GenerateUniqueConstructors(cls *class.Class, ast *ast.SchemaAST) string {
	var res strings.Builder

	if cls.HasPrimaryKey() {
		pkFields := cls.GetPrimaryKeyFields()
		if len(pkFields) == 1 {
			pkField := cls.Attributes.GetFieldByName(pkFields[0])
			if pkField != nil {
				fieldName := utils.ToExportedName(pkField.GetName())
				goType := strings.TrimPrefix(utils.GetGoType(pkField, ast), "*")
				paramName := pkField.GetName()

				res.WriteString(fmt.Sprintf("func %sWhere%s(%s %s) %sWhereUniqueInput {\n",
					cls.Name, fieldName, paramName, goType, cls.Name))
				res.WriteString(fmt.Sprintf("\treturn %sWhereUniqueInput{%s: &%s}\n", cls.Name, fieldName, paramName))
				res.WriteString("}\n\n")
			}
		}
	}

	for _, fld := range cls.Attributes.Fields {
		if fld.IsUnique() && !fld.IsPrimaryKey() {
			fieldName := utils.ToExportedName(fld.GetName())
			goType := strings.TrimPrefix(utils.GetGoType(fld, ast), "*")
			paramName := fld.GetName()

			res.WriteString(fmt.Sprintf("func %sWhere%s(%s %s) %sWhereUniqueInput {\n",
				cls.Name, fieldName, paramName, goType, cls.Name))
			res.WriteString(fmt.Sprintf("\treturn %sWhereUniqueInput{%s: &%s}\n", cls.Name, fieldName, paramName))
			res.WriteString("}\n\n")
		}
	}

	for _, directive := range cls.Attributes.Directives {
		if directive.Name == constants.CLASS_ATTR_UNIQUE {
			fields, err := directive.GetFields()
			if err != nil || len(fields) < 2 {
				continue
			}

			sortedFields := make([]string, len(fields))
			copy(sortedFields, fields)
			sort.Strings(sortedFields)

			var params []string
			var structFields []string
			var funcNameParts []string

			for _, fieldName := range sortedFields {
				fld := cls.Attributes.GetFieldByName(fieldName)
				if fld == nil {
					continue
				}
				goType := strings.TrimPrefix(utils.GetGoType(fld, ast), "*")
				exportedName := utils.ToExportedName(fieldName)
				params = append(params, fmt.Sprintf("%s %s", fieldName, goType))
				structFields = append(structFields, fmt.Sprintf("%s: %s", exportedName, fieldName))
				funcNameParts = append(funcNameParts, exportedName)
			}

			funcName := strings.Join(funcNameParts, "")
			compositeName := cls.Name + funcName + "Composite"
			inputFieldName := strings.Join(funcNameParts, "")

			res.WriteString(fmt.Sprintf("func %sWhere%s(%s) %sWhereUniqueInput {\n",
				cls.Name, funcName, strings.Join(params, ", "), cls.Name))
			res.WriteString(fmt.Sprintf("\treturn %sWhereUniqueInput{\n", cls.Name))
			res.WriteString(fmt.Sprintf("\t\t%s: &%s{\n", inputFieldName, compositeName))
			for _, structField := range structFields {
				res.WriteString(fmt.Sprintf("\t\t\t%s,\n", structField))
			}
			res.WriteString("\t\t},\n")
			res.WriteString("\t}\n")
			res.WriteString("}\n\n")
		}
	}

	return res.String()
}

func GenerateClassClient(cls *class.Class, ast *ast.SchemaAST) string {
	var res strings.Builder

	res.WriteString(fmt.Sprintf("type %sClient struct {\n", cls.Name))
	res.WriteString("\tdb *BlazeDatabaseClient\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (c *%sClient) Query() *%sQuery {\n", cls.Name, cls.Name))
	res.WriteString(fmt.Sprintf("\treturn &%sQuery{\n", cls.Name))
	res.WriteString("\t\tclient: c,\n")
	res.WriteString("\t\tpredicates: []Predicate{},\n")
	res.WriteString("\t}\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (c *%sClient) Create() *%sCreate {\n", cls.Name, cls.Name))
	res.WriteString(fmt.Sprintf("\treturn &%sCreate{\n", cls.Name))
	res.WriteString("\t\tclient: c,\n")
	res.WriteString("\t}\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (c *%sClient) Update() *%sUpdate {\n", cls.Name, cls.Name))
	res.WriteString(fmt.Sprintf("\treturn &%sUpdate{\n", cls.Name))
	res.WriteString("\t\tclient: c,\n")
	res.WriteString("\t\tpredicates: []Predicate{},\n")
	res.WriteString("\t}\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (c *%sClient) UpdateOne(where %sWhereUniqueInput) *%sUpdateOne {\n", cls.Name, cls.Name, cls.Name))
	res.WriteString(fmt.Sprintf("\treturn &%sUpdateOne{\n", cls.Name))
	res.WriteString("\t\tclient: c,\n")
	res.WriteString("\t\twhere: where,\n")
	res.WriteString("\t}\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (c *%sClient) Delete() *%sDelete {\n", cls.Name, cls.Name))
	res.WriteString(fmt.Sprintf("\treturn &%sDelete{\n", cls.Name))
	res.WriteString("\t\tclient: c,\n")
	res.WriteString("\t\tpredicates: []Predicate{},\n")
	res.WriteString("\t}\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (c *%sClient) DeleteOne(where %sWhereUniqueInput) *%sDelete {\n", cls.Name, cls.Name, cls.Name))
	res.WriteString(fmt.Sprintf("\treturn &%sDelete{\n", cls.Name))
	res.WriteString("\t\tclient: c,\n")
	res.WriteString("\t\twhere: &where,\n")
	res.WriteString("\t}\n")
	res.WriteString("}\n\n")

	return res.String()
}

func GenerateQueryBuilder(cls *class.Class, ast *ast.SchemaAST) string {
	var res strings.Builder

	res.WriteString(fmt.Sprintf("type %sQuery struct {\n", cls.Name))
	res.WriteString(fmt.Sprintf("\tclient *%sClient\n", cls.Name))
	res.WriteString("\tpredicates []Predicate\n")
	res.WriteString("\torders []*Order\n")
	res.WriteString("\tlimitValue *int\n")
	res.WriteString("\toffsetValue *int\n")
	res.WriteString("\tinclude []string\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (q *%sQuery) Where(predicates ...Predicate) *%sQuery {\n", cls.Name, cls.Name))
	res.WriteString("\tq.predicates = append(q.predicates, predicates...)\n")
	res.WriteString("\treturn q\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (q *%sQuery) OrderBy(orders ...OrderFunc) *%sQuery {\n", cls.Name, cls.Name))
	res.WriteString("\tfor _, order := range orders {\n")
	res.WriteString("\t\torder(&q.orders)\n")
	res.WriteString("\t}\n")
	res.WriteString("\treturn q\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (q *%sQuery) Limit(limit int) *%sQuery {\n", cls.Name, cls.Name))
	res.WriteString("\tq.limitValue = &limit\n")
	res.WriteString("\treturn q\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (q *%sQuery) Offset(offset int) *%sQuery {\n", cls.Name, cls.Name))
	res.WriteString("\tq.offsetValue = &offset\n")
	res.WriteString("\treturn q\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (q *%sQuery) Include(relations ...string) *%sQuery {\n", cls.Name, cls.Name))
	res.WriteString("\tq.include = append(q.include, relations...)\n")
	res.WriteString("\treturn q\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (q *%sQuery) Find() ([]*%s, error) {\n", cls.Name, cls.Name))
	res.WriteString("\tctx := q.client.db.Context()\n")
	res.WriteString("\t_ = ctx\n")
	res.WriteString("\t// TODO: Implement query execution\n")
	res.WriteString("\treturn nil, fmt.Errorf(\"not implemented\")\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (q *%sQuery) First() (*%s, error) {\n", cls.Name, cls.Name))
	res.WriteString("\tctx := q.client.db.Context()\n")
	res.WriteString("\t_ = ctx\n")
	res.WriteString("\t// TODO: Implement query execution\n")
	res.WriteString("\treturn nil, fmt.Errorf(\"not implemented\")\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (q *%sQuery) Count() (int64, error) {\n", cls.Name))
	res.WriteString("\tctx := q.client.db.Context()\n")
	res.WriteString("\t_ = ctx\n")
	res.WriteString("\t// TODO: Implement count\n")
	res.WriteString("\treturn 0, fmt.Errorf(\"not implemented\")\n")
	res.WriteString("}\n\n")

	return res.String()
}

func GenerateCreateBuilder(cls *class.Class, ast *ast.SchemaAST) string {
	var res strings.Builder

	res.WriteString(fmt.Sprintf("type %sCreate struct {\n", cls.Name))
	res.WriteString(fmt.Sprintf("\tclient *%sClient\n", cls.Name))
	res.WriteString("\tdata map[string]interface{}\n")
	res.WriteString("}\n\n")

	for _, fld := range cls.Attributes.Fields {
		if fld.IsObject() {
			continue
		}
		fieldName := utils.ToExportedName(fld.GetName())
		goType := utils.GetGoType(fld, ast)

		res.WriteString(fmt.Sprintf("func (c *%sCreate) Set%s(value %s) *%sCreate {\n",
			cls.Name, fieldName, goType, cls.Name))
		res.WriteString("\tif c.data == nil {\n")
		res.WriteString("\t\tc.data = make(map[string]interface{})\n")
		res.WriteString("\t}\n")
		res.WriteString(fmt.Sprintf("\tc.data[\"%s\"] = value\n", fld.GetName()))
		res.WriteString("\treturn c\n")
		res.WriteString("}\n\n")
	}

	res.WriteString(fmt.Sprintf("func (c *%sCreate) Save() (*%s, error) {\n", cls.Name, cls.Name))
	res.WriteString("\tctx := c.client.db.Context()\n")
	res.WriteString("\t_ = ctx\n")
	res.WriteString("\t// TODO: Implement insert\n")
	res.WriteString("\treturn nil, fmt.Errorf(\"not implemented\")\n")
	res.WriteString("}\n\n")

	return res.String()
}

func GenerateUpdateBuilder(cls *class.Class, ast *ast.SchemaAST) string {
	var res strings.Builder

	res.WriteString(fmt.Sprintf("type %sUpdate struct {\n", cls.Name))
	res.WriteString(fmt.Sprintf("\tclient *%sClient\n", cls.Name))
	res.WriteString("\tpredicates []Predicate\n")
	res.WriteString("\tdata map[string]interface{}\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (u *%sUpdate) Where(predicates ...Predicate) *%sUpdate {\n", cls.Name, cls.Name))
	res.WriteString("\tu.predicates = append(u.predicates, predicates...)\n")
	res.WriteString("\treturn u\n")
	res.WriteString("}\n\n")

	for _, fld := range cls.Attributes.Fields {
		if fld.IsObject() || fld.IsPrimaryKey() {
			continue
		}
		fieldName := utils.ToExportedName(fld.GetName())
		goType := utils.GetGoType(fld, ast)

		res.WriteString(fmt.Sprintf("func (u *%sUpdate) Set%s(value %s) *%sUpdate {\n",
			cls.Name, fieldName, goType, cls.Name))
		res.WriteString("\tif u.data == nil {\n")
		res.WriteString("\t\tu.data = make(map[string]interface{})\n")
		res.WriteString("\t}\n")
		res.WriteString(fmt.Sprintf("\tu.data[\"%s\"] = value\n", fld.GetName()))
		res.WriteString("\treturn u\n")
		res.WriteString("}\n\n")
	}

	res.WriteString(fmt.Sprintf("func (u *%sUpdate) Save() (int64, error) {\n", cls.Name))
	res.WriteString("\tctx := u.client.db.Context()\n")
	res.WriteString("\t_ = ctx\n")
	res.WriteString("\t// TODO: Implement update\n")
	res.WriteString("\treturn 0, fmt.Errorf(\"not implemented\")\n")
	res.WriteString("}\n\n")

	return res.String()
}

func GenerateUpdateOneBuilder(cls *class.Class, ast *ast.SchemaAST) string {
	var res strings.Builder

	res.WriteString(fmt.Sprintf("type %sUpdateOne struct {\n", cls.Name))
	res.WriteString(fmt.Sprintf("\tclient *%sClient\n", cls.Name))
	res.WriteString(fmt.Sprintf("\twhere %sWhereUniqueInput\n", cls.Name))
	res.WriteString("\tdata map[string]interface{}\n")
	res.WriteString("}\n\n")

	for _, fld := range cls.Attributes.Fields {
		if fld.IsObject() || fld.IsPrimaryKey() {
			continue
		}
		fieldName := utils.ToExportedName(fld.GetName())
		goType := utils.GetGoType(fld, ast)

		res.WriteString(fmt.Sprintf("func (u *%sUpdateOne) Set%s(value %s) *%sUpdateOne {\n",
			cls.Name, fieldName, goType, cls.Name))
		res.WriteString("\tif u.data == nil {\n")
		res.WriteString("\t\tu.data = make(map[string]interface{})\n")
		res.WriteString("\t}\n")
		res.WriteString(fmt.Sprintf("\tu.data[\"%s\"] = value\n", fld.GetName()))
		res.WriteString("\treturn u\n")
		res.WriteString("}\n\n")
	}

	res.WriteString(fmt.Sprintf("func (u *%sUpdateOne) Save() (*%s, error) {\n", cls.Name, cls.Name))
	res.WriteString("\tctx := u.client.db.Context()\n")
	res.WriteString("\t_ = ctx\n")
	res.WriteString("\t// TODO: Implement update one\n")
	res.WriteString("\treturn nil, fmt.Errorf(\"not implemented\")\n")
	res.WriteString("}\n\n")

	return res.String()
}

func GenerateDeleteBuilder(cls *class.Class, ast *ast.SchemaAST) string {
	var res strings.Builder

	res.WriteString(fmt.Sprintf("type %sDelete struct {\n", cls.Name))
	res.WriteString(fmt.Sprintf("\tclient *%sClient\n", cls.Name))
	res.WriteString("\tpredicates []Predicate\n")
	res.WriteString(fmt.Sprintf("\twhere *%sWhereUniqueInput\n", cls.Name))
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (d *%sDelete) Where(predicates ...Predicate) *%sDelete {\n", cls.Name, cls.Name))
	res.WriteString("\td.predicates = append(d.predicates, predicates...)\n")
	res.WriteString("\treturn d\n")
	res.WriteString("}\n\n")

	res.WriteString(fmt.Sprintf("func (d *%sDelete) Exec() (int64, error) {\n", cls.Name))
	res.WriteString("\tctx := d.client.db.Context()\n")
	res.WriteString("\t_ = ctx\n")
	res.WriteString("\t// TODO: Implement delete\n")
	res.WriteString("\treturn 0, fmt.Errorf(\"not implemented\")\n")
	res.WriteString("}\n\n")

	return res.String()
}
