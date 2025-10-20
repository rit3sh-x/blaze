package hooks

import (
	"fmt"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast"
	"github.com/rit3sh-x/blaze/core/ast/class"
	"github.com/rit3sh-x/blaze/core/constants"
	"github.com/rit3sh-x/blaze/core/utils"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ClientGenerator struct {
	ast     *ast.SchemaAST
	builder strings.Builder
}

func NewClientGenerator(schemaAST *ast.SchemaAST) *ClientGenerator {
	return &ClientGenerator{
		ast: schemaAST,
	}
}

func (cg *ClientGenerator) Generate() string {
	cg.builder.Reset()

	cg.generatePredicates()

	cg.generateCompositePredicates()

	cg.generateQueryBuilders()

	cg.generateCreateBuilders()

	cg.generateUpdateBuilders()

	cg.generateDeleteBuilders()

	cg.generateClients()

	return cg.builder.String()
}

func (cg *ClientGenerator) generatePredicates() {
	cg.builder.WriteString("// ==================== PREDICATES ====================\n\n")

	for _, cls := range cg.ast.Classes {
		predicates := GenerateModelPredicates(cls, cg.ast)
		cg.builder.WriteString(predicates)
		cg.builder.WriteString("\n")
	}
}

func (cg *ClientGenerator) generateCompositePredicates() {
	for _, cls := range cg.ast.Classes {
		cg.generateClassCompositePredicates(cls)
	}
}

func (cg *ClientGenerator) generateClassCompositePredicates(cls *class.Class) {
	lowerClass := cases.Lower(language.English).String(cls.Name)

	for _, directive := range cls.Attributes.Directives {
		if directive.Name == constants.CLASS_ATTR_UNIQUE {
			fields, err := directive.GetFields()
			if err != nil || len(fields) < 2 {
				continue
			}

			var params []string
			var fieldNameParts []string

			for _, fieldName := range fields {
				fld := cls.Attributes.GetFieldByName(fieldName)
				if fld == nil {
					continue
				}
				goType := utils.GetGoType(fld, cg.ast)
				goType = strings.TrimPrefix(goType, "*")
				params = append(params, fmt.Sprintf("%s %s", fieldName, goType))
				fieldNameParts = append(fieldNameParts, utils.ToExportedName(fieldName))
			}

			funcName := strings.Join(fieldNameParts, "")
			cg.builder.WriteString(fmt.Sprintf("func (%s) %sEQ(%s) Predicate { return func(interface{}) {} }\n\n",
				lowerClass, funcName, strings.Join(params, ", ")))
		}
	}
}

func (cg *ClientGenerator) generateQueryBuilders() {
	cg.builder.WriteString("// ==================== QUERY BUILDERS ====================\n\n")

	for _, cls := range cg.ast.Classes {
		cg.generateClassQuery(cls)
	}
}

func (cg *ClientGenerator) generateClassQuery(cls *class.Class) {
	className := cls.Name
	relationFields := cls.GetRelationFields()

	cg.builder.WriteString(fmt.Sprintf("type %sQuery struct {\n", className))
	cg.builder.WriteString("\tdb *BlazeDB\n")
	cg.builder.WriteString("\tpredicates []Predicate\n")
	cg.builder.WriteString("\torder []*Order\n")
	cg.builder.WriteString("\tlimit *int\n")
	cg.builder.WriteString("\toffset *int\n")
	cg.builder.WriteString("\tunique bool\n")

	for _, relField := range relationFields {
		relType := relField.GetBaseType()
		fieldName := utils.ToExportedName(relField.GetName())
		cg.builder.WriteString(fmt.Sprintf("\twith%s *%sQuery\n", fieldName, relType))
	}
	cg.builder.WriteString("}\n\n")

	cg.generateQueryMethods(cls)
}

func (cg *ClientGenerator) generateQueryMethods(cls *class.Class) {
	className := cls.Name

	cg.builder.WriteString(fmt.Sprintf("func (q *%sQuery) Where(ps ...Predicate) *%sQuery {\n", className, className))
	cg.builder.WriteString("\tq.predicates = append(q.predicates, ps...)\n")
	cg.builder.WriteString("\treturn q\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (q *%sQuery) Order(opts ...OrderFunc) *%sQuery {\n", className, className))
	cg.builder.WriteString("\tfor _, opt := range opts {\n")
	cg.builder.WriteString("\t\topt(&q.order)\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("\treturn q\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (q *%sQuery) Limit(limit int) *%sQuery {\n", className, className))
	cg.builder.WriteString("\tq.limit = &limit\n")
	cg.builder.WriteString("\treturn q\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (q *%sQuery) Offset(offset int) *%sQuery {\n", className, className))
	cg.builder.WriteString("\tq.offset = &offset\n")
	cg.builder.WriteString("\treturn q\n")
	cg.builder.WriteString("}\n\n")

	for _, relField := range cls.GetRelationFields() {
		relType := relField.GetBaseType()
		fieldName := utils.ToExportedName(relField.GetName())

		cg.builder.WriteString(fmt.Sprintf("func (q *%sQuery) With%s(opts ...func(*%sQuery)) *%sQuery {\n",
			className, fieldName, relType, className))
		cg.builder.WriteString(fmt.Sprintf("\tq.with%s = &%sQuery{db: q.db}\n", fieldName, relType))
		cg.builder.WriteString("\tfor _, opt := range opts {\n")
		cg.builder.WriteString(fmt.Sprintf("\t\topt(q.with%s)\n", fieldName))
		cg.builder.WriteString("\t}\n")
		cg.builder.WriteString("\treturn q\n")
		cg.builder.WriteString("}\n\n")
	}

	cg.builder.WriteString(fmt.Sprintf("func (q *%sQuery) All(ctx context.Context) ([]*%s, error) {\n", className, className))
	cg.builder.WriteString("\t// TODO: Implementation\n")
	cg.builder.WriteString("\treturn nil, nil\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (q *%sQuery) First(ctx context.Context) (*%s, error) {\n", className, className))
	cg.builder.WriteString("\t// TODO: Implementation\n")
	cg.builder.WriteString("\treturn nil, nil\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (q *%sQuery) Only(ctx context.Context) (*%s, error) {\n", className, className))
	cg.builder.WriteString("\t// TODO: Implementation\n")
	cg.builder.WriteString("\treturn nil, nil\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (q *%sQuery) Count(ctx context.Context) (int, error) {\n", className))
	cg.builder.WriteString("\t// TODO: Implementation\n")
	cg.builder.WriteString("\treturn 0, nil\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (q *%sQuery) Exist(ctx context.Context) (bool, error) {\n", className))
	cg.builder.WriteString("\t// TODO: Implementation\n")
	cg.builder.WriteString("\treturn false, nil\n")
	cg.builder.WriteString("}\n\n")
}

func (cg *ClientGenerator) generateCreateBuilders() {
	cg.builder.WriteString("// ==================== CREATE BUILDERS ====================\n\n")

	for _, cls := range cg.ast.Classes {
		cg.generateClassCreate(cls)
	}
}

func (cg *ClientGenerator) generateClassCreate(cls *class.Class) {
	className := cls.Name

	cg.builder.WriteString(fmt.Sprintf("type %sCreate struct {\n", className))
	cg.builder.WriteString("\tdb *BlazeDB\n")
	cg.builder.WriteString("\tfields map[string]interface{}\n")
	cg.builder.WriteString("\tedges map[string]interface{}\n")
	cg.builder.WriteString("}\n\n")

	for _, fld := range cls.Attributes.Fields {
		if fld.IsObject() {
			continue
		}

		fieldName := utils.ToExportedName(fld.GetName())
		goType := utils.GetGoType(fld, cg.ast)
		baseType := strings.TrimPrefix(goType, "*")

		cg.builder.WriteString(fmt.Sprintf("func (c *%sCreate) Set%s(v %s) *%sCreate {\n",
			className, fieldName, baseType, className))
		cg.builder.WriteString(fmt.Sprintf("\tc.fields[\"%s\"] = v\n", fld.GetName()))
		cg.builder.WriteString("\treturn c\n")
		cg.builder.WriteString("}\n\n")

		if fld.IsOptional() {
			cg.builder.WriteString(fmt.Sprintf("func (c *%sCreate) SetNillable%s(v *%s) *%sCreate {\n",
				className, fieldName, baseType, className))
			cg.builder.WriteString("\tif v != nil {\n")
			cg.builder.WriteString(fmt.Sprintf("\t\tc.fields[\"%s\"] = *v\n", fld.GetName()))
			cg.builder.WriteString("\t}\n")
			cg.builder.WriteString("\treturn c\n")
			cg.builder.WriteString("}\n\n")
		}
	}

	cg.builder.WriteString(fmt.Sprintf("func (c *%sCreate) Save(ctx context.Context) (*%s, error) {\n",
		className, className))
	cg.builder.WriteString("\t// TODO: Implementation\n")
	cg.builder.WriteString("\treturn nil, nil\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (c *%sCreate) SaveX(ctx context.Context) *%s {\n",
		className, className))
	cg.builder.WriteString("\tv, err := c.Save(ctx)\n")
	cg.builder.WriteString("\tif err != nil {\n")
	cg.builder.WriteString("\t\tpanic(err)\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("\treturn v\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("type %sCreateBulk struct {\n", className))
	cg.builder.WriteString("\tdb *BlazeDB\n")
	cg.builder.WriteString(fmt.Sprintf("\tbuilders []*%sCreate\n", className))
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (b *%sCreateBulk) Save(ctx context.Context) ([]*%s, error) {\n",
		className, className))
	cg.builder.WriteString("\t// TODO: Implementation\n")
	cg.builder.WriteString("\treturn nil, nil\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (b *%sCreateBulk) SaveX(ctx context.Context) []*%s {\n",
		className, className))
	cg.builder.WriteString("\tv, err := b.Save(ctx)\n")
	cg.builder.WriteString("\tif err != nil {\n")
	cg.builder.WriteString("\t\tpanic(err)\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("\treturn v\n")
	cg.builder.WriteString("}\n\n")
}

func (cg *ClientGenerator) generateUpdateBuilders() {
	cg.builder.WriteString("// ==================== UPDATE BUILDERS ====================\n\n")

	for _, cls := range cg.ast.Classes {
		cg.generateClassUpdate(cls)
		cg.generateClassUpdateOne(cls)
	}
}

func (cg *ClientGenerator) generateClassUpdate(cls *class.Class) {
	className := cls.Name

	cg.builder.WriteString(fmt.Sprintf("type %sUpdate struct {\n", className))
	cg.builder.WriteString("\tdb *BlazeDB\n")
	cg.builder.WriteString("\tpredicates []Predicate\n")
	cg.builder.WriteString("\tfields map[string]interface{}\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (u *%sUpdate) Where(ps ...Predicate) *%sUpdate {\n",
		className, className))
	cg.builder.WriteString("\tu.predicates = append(u.predicates, ps...)\n")
	cg.builder.WriteString("\treturn u\n")
	cg.builder.WriteString("}\n\n")

	for _, fld := range cls.Attributes.Fields {
		if fld.IsObject() || fld.IsPrimaryKey() {
			continue
		}

		fieldName := utils.ToExportedName(fld.GetName())
		goType := utils.GetGoType(fld, cg.ast)
		baseType := strings.TrimPrefix(goType, "*")

		cg.builder.WriteString(fmt.Sprintf("func (u *%sUpdate) Set%s(v %s) *%sUpdate {\n",
			className, fieldName, baseType, className))
		cg.builder.WriteString(fmt.Sprintf("\tu.fields[\"%s\"] = v\n", fld.GetName()))
		cg.builder.WriteString("\treturn u\n")
		cg.builder.WriteString("}\n\n")

		if fld.IsScalar() {
			scalarType := fld.GetBaseType()
			switch constants.ScalarType(scalarType) {
			case constants.INT, constants.BIGINT, constants.SMALLINT, constants.FLOAT, constants.NUMERIC:
				cg.builder.WriteString(fmt.Sprintf("func (u *%sUpdate) Add%s(v %s) *%sUpdate {\n",
					className, fieldName, baseType, className))
				cg.builder.WriteString(fmt.Sprintf("\tu.fields[\"add_%s\"] = v\n", fld.GetName()))
				cg.builder.WriteString("\treturn u\n")
				cg.builder.WriteString("}\n\n")
			}
		}

		if fld.IsOptional() {
			cg.builder.WriteString(fmt.Sprintf("func (u *%sUpdate) Clear%s() *%sUpdate {\n",
				className, fieldName, className))
			cg.builder.WriteString(fmt.Sprintf("\tu.fields[\"clear_%s\"] = true\n", fld.GetName()))
			cg.builder.WriteString("\treturn u\n")
			cg.builder.WriteString("}\n\n")
		}
	}

	cg.builder.WriteString(fmt.Sprintf("func (u *%sUpdate) Save(ctx context.Context) (int, error) {\n", className))
	cg.builder.WriteString("\t// TODO: Implementation\n")
	cg.builder.WriteString("\treturn 0, nil\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (u *%sUpdate) SaveX(ctx context.Context) int {\n", className))
	cg.builder.WriteString("\tv, err := u.Save(ctx)\n")
	cg.builder.WriteString("\tif err != nil {\n")
	cg.builder.WriteString("\t\tpanic(err)\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("\treturn v\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (u *%sUpdate) Exec(ctx context.Context) error {\n", className))
	cg.builder.WriteString("\t_, err := u.Save(ctx)\n")
	cg.builder.WriteString("\treturn err\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (u *%sUpdate) ExecX(ctx context.Context) {\n", className))
	cg.builder.WriteString("\tif err := u.Exec(ctx); err != nil {\n")
	cg.builder.WriteString("\t\tpanic(err)\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("}\n\n")
}

func (cg *ClientGenerator) generateClassUpdateOne(cls *class.Class) {
	className := cls.Name

	cg.builder.WriteString(fmt.Sprintf("type %sUpdateOne struct {\n", className))
	cg.builder.WriteString("\tdb *BlazeDB\n")
	cg.builder.WriteString("\tfields map[string]interface{}\n")
	cg.builder.WriteString(fmt.Sprintf("\twhere %sWhereUniqueInput\n", className))
	cg.builder.WriteString("}\n\n")

	for _, fld := range cls.Attributes.Fields {
		if fld.IsObject() || fld.IsPrimaryKey() {
			continue
		}

		fieldName := utils.ToExportedName(fld.GetName())
		goType := utils.GetGoType(fld, cg.ast)
		baseType := strings.TrimPrefix(goType, "*")

		cg.builder.WriteString(fmt.Sprintf("func (u *%sUpdateOne) Set%s(v %s) *%sUpdateOne {\n",
			className, fieldName, baseType, className))
		cg.builder.WriteString(fmt.Sprintf("\tu.fields[\"%s\"] = v\n", fld.GetName()))
		cg.builder.WriteString("\treturn u\n")
		cg.builder.WriteString("}\n\n")
	}
	
	cg.builder.WriteString(fmt.Sprintf("func (u *%sUpdateOne) Save(ctx context.Context) (*%s, error) {\n",
		className, className))
	cg.builder.WriteString("\t// TODO: Implementation\n")
	cg.builder.WriteString("\treturn nil, nil\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (u *%sUpdateOne) SaveX(ctx context.Context) *%s {\n",
		className, className))
	cg.builder.WriteString("\tv, err := u.Save(ctx)\n")
	cg.builder.WriteString("\tif err != nil {\n")
	cg.builder.WriteString("\t\tpanic(err)\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("\treturn v\n")
	cg.builder.WriteString("}\n\n")
}

func (cg *ClientGenerator) generateDeleteBuilders() {
	cg.builder.WriteString("// ==================== DELETE BUILDERS ====================\n\n")

	for _, cls := range cg.ast.Classes {
		cg.generateClassDelete(cls)
	}
}

func (cg *ClientGenerator) generateClassDelete(cls *class.Class) {
	className := cls.Name

	cg.builder.WriteString(fmt.Sprintf("type %sDelete struct {\n", className))
	cg.builder.WriteString("\tdb *BlazeDB\n")
	cg.builder.WriteString("\tpredicates []Predicate\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (d *%sDelete) Where(ps ...Predicate) *%sDelete {\n",
		className, className))
	cg.builder.WriteString("\td.predicates = append(d.predicates, ps...)\n")
	cg.builder.WriteString("\treturn d\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (d *%sDelete) Exec(ctx context.Context) (int, error) {\n", className))
	cg.builder.WriteString("\t// TODO: Implementation\n")
	cg.builder.WriteString("\treturn 0, nil\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (d *%sDelete) ExecX(ctx context.Context) int {\n", className))
	cg.builder.WriteString("\tv, err := d.Exec(ctx)\n")
	cg.builder.WriteString("\tif err != nil {\n")
	cg.builder.WriteString("\t\tpanic(err)\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("\treturn v\n")
	cg.builder.WriteString("}\n\n")
}

func (cg *ClientGenerator) generateClients() {
	cg.builder.WriteString("// ==================== CLIENTS ====================\n\n")

	for _, cls := range cg.ast.Classes {
		cg.generateClassClient(cls)
	}
}

func (cg *ClientGenerator) generateClassClient(cls *class.Class) {
	className := cls.Name

	cg.builder.WriteString(fmt.Sprintf("type %sClient struct {\n", className))
	cg.builder.WriteString("\tdb *BlazeDB\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (c *%sClient) Create() *%sCreate {\n", className, className))
	cg.builder.WriteString(fmt.Sprintf("\treturn &%sCreate{\n", className))
	cg.builder.WriteString("\t\tdb: c.db,\n")
	cg.builder.WriteString("\t\tfields: make(map[string]interface{}),\n")
	cg.builder.WriteString("\t\tedges: make(map[string]interface{}),\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (c *%sClient) CreateBulk(builders ...*%sCreate) *%sCreateBulk {\n",
		className, className, className))
	cg.builder.WriteString(fmt.Sprintf("\treturn &%sCreateBulk{\n", className))
	cg.builder.WriteString("\t\tdb: c.db,\n")
	cg.builder.WriteString("\t\tbuilders: builders,\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (c *%sClient) Update() *%sUpdate {\n", className, className))
	cg.builder.WriteString(fmt.Sprintf("\treturn &%sUpdate{\n", className))
	cg.builder.WriteString("\t\tdb: c.db,\n")
	cg.builder.WriteString("\t\tfields: make(map[string]interface{}),\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (c *%sClient) UpdateOne(entity *%s) *%sUpdateOne {\n",
		className, className, className))
	cg.builder.WriteString(fmt.Sprintf("\treturn &%sUpdateOne{\n", className))
	cg.builder.WriteString("\t\tdb: c.db,\n")
	cg.builder.WriteString("\t\tfields: make(map[string]interface{}),\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("}\n\n")

	pkFields := cls.GetPrimaryKeyFields()
	if len(pkFields) == 1 {
		pkField := cls.Attributes.GetFieldByName(pkFields[0])
		if pkField != nil {
			pkType := utils.GetGoType(pkField, cg.ast)
			pkType = strings.TrimPrefix(pkType, "*")

			cg.builder.WriteString(fmt.Sprintf("func (c *%sClient) UpdateOneID(id %s) *%sUpdateOne {\n",
				className, pkType, className))
			cg.builder.WriteString(fmt.Sprintf("\treturn &%sUpdateOne{\n", className))
			cg.builder.WriteString("\t\tdb: c.db,\n")
			cg.builder.WriteString("\t\tfields: make(map[string]interface{}),\n")
			cg.builder.WriteString(fmt.Sprintf("\t\twhere: %sWhereUniqueInput{%s: &id},\n",
				className, utils.ToExportedName(pkFields[0])))
			cg.builder.WriteString("\t}\n")
			cg.builder.WriteString("}\n\n")
		}
	}

	cg.builder.WriteString(fmt.Sprintf("func (c *%sClient) UpdateOneWhere(where %sWhereUniqueInput) *%sUpdateOne {\n",
		className, className, className))
	cg.builder.WriteString(fmt.Sprintf("\treturn &%sUpdateOne{\n", className))
	cg.builder.WriteString("\t\tdb: c.db,\n")
	cg.builder.WriteString("\t\tfields: make(map[string]interface{}),\n")
	cg.builder.WriteString("\t\twhere: where,\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (c *%sClient) Delete() *%sDelete {\n", className, className))
	cg.builder.WriteString(fmt.Sprintf("\treturn &%sDelete{db: c.db}\n", className))
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (c *%sClient) Query() *%sQuery {\n", className, className))
	cg.builder.WriteString(fmt.Sprintf("\treturn &%sQuery{db: c.db}\n", className))
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (c *%sClient) Get(ctx context.Context, where %sWhereUniqueInput) (*%s, error) {\n",
		className, className, className))
	cg.builder.WriteString("\t// TODO: Implementation\n")
	cg.builder.WriteString("\treturn nil, nil\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (c *%sClient) GetX(ctx context.Context, where %sWhereUniqueInput) *%s {\n",
		className, className, className))
	cg.builder.WriteString("\tv, err := c.Get(ctx, where)\n")
	cg.builder.WriteString("\tif err != nil {\n")
	cg.builder.WriteString("\t\tpanic(err)\n")
	cg.builder.WriteString("\t}\n")
	cg.builder.WriteString("\treturn v\n")
	cg.builder.WriteString("}\n\n")

	cg.builder.WriteString(fmt.Sprintf("func (c *%sClient) FindUnique(where %sWhereUniqueInput) *%sQuery {\n",
		className, className, className))
	cg.builder.WriteString(fmt.Sprintf("\tq := &%sQuery{db: c.db, unique: true}\n", className))
	cg.builder.WriteString("\t// TODO: Set unique predicates from where\n")
	cg.builder.WriteString("\treturn q\n")
	cg.builder.WriteString("}\n\n")
}
