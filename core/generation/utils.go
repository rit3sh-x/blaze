package generation

import (
    "github.com/rit3sh-x/blaze/core/ast/class"
    "github.com/rit3sh-x/blaze/core/ast/field"
    "github.com/rit3sh-x/blaze/core/constants"
)

func GetUniqueFields(cls *class.Class) []*field.Field {
    var uniqueFields []*field.Field
    seen := make(map[string]bool)

    if cls.HasPrimaryKey() {
        pkFields := cls.GetPrimaryKeyFields()
        if len(pkFields) == 1 {
            if f := cls.Attributes.GetFieldByName(pkFields[0]); f != nil {
                uniqueFields = append(uniqueFields, f)
                seen[pkFields[0]] = true
            }
        }
    }

    for _, field := range cls.Attributes.Fields {
        fieldName := field.GetName()
        if field.IsUnique() && !seen[fieldName] {
            uniqueFields = append(uniqueFields, field)
            seen[fieldName] = true
        }
    }

    for _, directive := range cls.Attributes.Directives {
        if directive.Name == constants.CLASS_ATTR_UNIQUE {
            fields, err := directive.GetFields()
            if err != nil || len(fields) != 1 {
                continue
            }
            if f := cls.Attributes.GetFieldByName(fields[0]); f != nil && !seen[fields[0]] {
                uniqueFields = append(uniqueFields, f)
                seen[fields[0]] = true
            }
        }
    }

    return uniqueFields
}

func GetCompositeUniqueConstraints(cls *class.Class) [][]string {
    var composites [][]string

    if cls.HasPrimaryKey() {
        pkFields := cls.GetPrimaryKeyFields()
        if len(pkFields) > 1 {
            composites = append(composites, pkFields)
        }
    }

    for _, directive := range cls.Attributes.Directives {
        if directive.Name == constants.CLASS_ATTR_UNIQUE {
            fields, err := directive.GetFields()
            if err != nil || len(fields) < 2 {
                continue
            }
            composites = append(composites, fields)
        }
    }

    return composites
}

func GetRelationFields(cls *class.Class) []*field.Field {
    var relations []*field.Field
    for _, f := range cls.Attributes.Fields {
        if f.IsObject() {
            relations = append(relations, f)
        }
    }
    return relations
}