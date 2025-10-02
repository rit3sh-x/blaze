package enum

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rit3sh-x/blaze/core/constants"
)

type EnumData struct {
	Name      string
	Value     string
	SortOrder int8
}

type EnumDefinition struct {
	Name   string
	Values []string
}

func GenerateEnumSchema(enumData []EnumData) string {
	if len(enumData) == 0 {
		return ""
	}

	enumMap := make(map[string][]EnumData)
	for _, data := range enumData {
		enumMap[data.Name] = append(enumMap[data.Name], data)
	}

	var enums []EnumDefinition
	for enumName, values := range enumMap {
		sort.Slice(values, func(i, j int) bool {
			return values[i].SortOrder < values[j].SortOrder
		})

		var enumValues []string
		for _, v := range values {
			enumValues = append(enumValues, v.Value)
		}

		enums = append(enums, EnumDefinition{
			Name:   enumName,
			Values: enumValues,
		})
	}

	sort.Slice(enums, func(i, j int) bool {
		return enums[i].Name < enums[j].Name
	})

	var schemaBuilder strings.Builder
	for i, enum := range enums {
		if i > 0 {
			schemaBuilder.WriteString("\n\n")
		}
		schemaBuilder.WriteString(formatEnum(enum))
	}

	return schemaBuilder.String()
}

func formatEnum(enum EnumDefinition) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(constants.KEYWORD_ENUM+" %s { \n", enum.Name))

	for _, value := range enum.Values {
		builder.WriteString(fmt.Sprintf("  %s \n", value))
	}

	builder.WriteString("}")
	return builder.String()
}

func GetEnumNames(enumData []EnumData) []string {
	nameSet := make(map[string]bool)
	var names []string

	for _, data := range enumData {
		if !nameSet[data.Name] {
			nameSet[data.Name] = true
			names = append(names, data.Name)
		}
	}

	sort.Strings(names)
	return names
}