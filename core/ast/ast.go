package ast

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast/class"
	"github.com/rit3sh-x/blaze/core/ast/enum"
	"github.com/rit3sh-x/blaze/core/constants"
)

type SchemaAST struct {
	Enums   map[string]*enum.Enum
	Classes []*class.Class
}

type ASTBuilder struct {
	enumValidator  *enum.EnumValidator
	classValidator *class.ClassValidator
}

func NewASTBuilder() *ASTBuilder {
	return &ASTBuilder{
		enumValidator: enum.NewEnumValidator(),
	}
}

func (ab *ASTBuilder) BuildAST(enumContent string, classContent string) (*SchemaAST, error) {
	ast := &SchemaAST{
		Enums:   make(map[string]*enum.Enum),
		Classes: []*class.Class{},
	}

	if err := ab.parseEnums(enumContent, ast); err != nil {
		return nil, fmt.Errorf("enum parsing failed: %v", err)
	}

	ab.classValidator = class.NewClassValidator(ast.Enums)

	if err := ab.parseClasses(classContent, ast); err != nil {
		return nil, fmt.Errorf("class parsing failed: %v", err)
	}
	return ast, nil
}

func (ab *ASTBuilder) parseEnums(enumContent string, ast *SchemaAST) error {
    if strings.TrimSpace(enumContent) == "" {
        return nil
    }

    enumPattern := regexp.MustCompile(`(?s)` + constants.KEYWORD_ENUM + `\s+[A-Z][a-zA-Z0-9_]{0,63}\s*\{[^{}]*\}`)
    enumDefs := enumPattern.FindAllString(enumContent, -1)

    for i, enumDef := range enumDefs {
        parsedEnum, err := ab.enumValidator.ParseEnum(enumDef, i)
        if err != nil {
            return fmt.Errorf("failed to parse enum at position %d: %v", i, err)
        }

        ast.Enums[parsedEnum.Name] = parsedEnum
    }

    return nil
}

func (ab *ASTBuilder) parseClasses(classContent string, ast *SchemaAST) error {
    if strings.TrimSpace(classContent) == "" {
        return nil
    }

    classPattern := regexp.MustCompile(`(?s)` + constants.KEYWORD_CLASS + `\s+[A-Z][a-zA-Z0-9_]{0,63}\s*\{[^{}]*\}`)
    classDefs := classPattern.FindAllString(classContent, -1)

    for i, classDef := range classDefs {
        parsedClass, err := ab.classValidator.ParseClass(classDef, i)
        if err != nil {
            return fmt.Errorf("failed to parse class at position %d: %v", i, err)
        }
        ast.Classes = append(ast.Classes, parsedClass)
    }

    return nil
}

func (ast *SchemaAST) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Schema AST: %d enums, %d classes", len(ast.Enums), len(ast.Classes)))

	for _, e := range ast.Enums {
		parts = append(parts, fmt.Sprintf("  enum %s", e.Name))
	}

	for _, c := range ast.Classes {
		parts = append(parts, fmt.Sprintf("  class %s (%d fields)", c.Name, len(c.Attributes.Fields)))
	}

	return strings.Join(parts, "\n")
}

func (ast *SchemaAST) GetEnumByName(name string) *enum.Enum {
	for _, e := range ast.Enums {
		if e.Name == name {
			return e
		}
	}
	return nil
}

func (ast *SchemaAST) GetClassByName(name string) *class.Class {
	for _, c := range ast.Classes {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func (ast *SchemaAST) GetAllFieldTypes() []string {
	typeSet := make(map[string]bool)

	for _, cls := range ast.Classes {
		for _, field := range cls.Attributes.Fields {
			typeSet[field.GetBaseType()] = true
		}
	}

	var types []string
	for t := range typeSet {
		types = append(types, t)
	}

	sort.Strings(types)
	return types
}

func (ast *SchemaAST) GetDependencyOrder() ([]*class.Class, error) {
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	for _, cls := range ast.Classes {
		graph[cls.Name] = []string{}
		inDegree[cls.Name] = 0
	}

	for _, cls := range ast.Classes {
		for _, field := range cls.Attributes.Fields {
			baseType := field.GetBaseType()
			if ast.GetClassByName(baseType) != nil {
				graph[cls.Name] = append(graph[cls.Name], baseType)
				inDegree[baseType]++
			}
		}
	}

	var queue []string
	for className, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, className)
		}
	}

	var result []*class.Class
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if cls := ast.GetClassByName(current); cls != nil {
			result = append(result, cls)
		}

		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(ast.Classes) {
		return nil, fmt.Errorf("circular dependency detected")
	}

	return result, nil
}

func BuildSchemaAST(enumContent string, classContent string) (*SchemaAST, error) {
	builder := NewASTBuilder()
	return builder.BuildAST(enumContent, classContent)
}