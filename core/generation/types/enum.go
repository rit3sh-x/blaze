package types

import (
	"fmt"
	"strings"

	"github.com/rit3sh-x/blaze/core/ast/enum"
)

type EnumGenerator struct {
	enum *enum.Enum
}

func NewEnumGenerator(e *enum.Enum) *EnumGenerator {
	return &EnumGenerator{
		enum: e,
	}
}

func (eg *EnumGenerator) Generate() string {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("type %s string\n\n", eg.enum.Name))

	content.WriteString("const (\n")
	for i, value := range eg.enum.Values {
		valueName := value.Name
		if i == 0 {
			content.WriteString(fmt.Sprintf("\t%s_%s %s = \"%s\"\n",
				eg.enum.Name, valueName, eg.enum.Name, valueName))
		} else {
			content.WriteString(fmt.Sprintf("\t%s_%s %s = \"%s\"\n",
				eg.enum.Name, valueName, eg.enum.Name, valueName))
		}
	}
	content.WriteString(")\n\n")
	eg.generateHelperMethods(&content)

	return content.String()
}

func (eg *EnumGenerator) generateHelperMethods(content *strings.Builder) {
	content.WriteString(fmt.Sprintf("func (e %s) IsValid() bool {\n", eg.enum.Name))
	content.WriteString("\tswitch e {\n")
	for _, value := range eg.enum.Values {
		content.WriteString(fmt.Sprintf("\tcase %s_%s:\n", eg.enum.Name, value.Name))
	}
	content.WriteString("\t\treturn true\n")
	content.WriteString("\t}\n")
	content.WriteString("\treturn false\n")
	content.WriteString("}\n\n")

	content.WriteString(fmt.Sprintf("func (e %s) String() string {\n", eg.enum.Name))
	content.WriteString("\treturn string(e)\n")
	content.WriteString("}\n\n")

	content.WriteString(fmt.Sprintf("func %sValues() []%s {\n", eg.enum.Name, eg.enum.Name))
	content.WriteString("\treturn []" + eg.enum.Name + "{\n")
	for _, value := range eg.enum.Values {
		content.WriteString(fmt.Sprintf("\t\t%s_%s,\n", eg.enum.Name, value.Name))
	}
	content.WriteString("\t}\n")
	content.WriteString("}\n\n")
}
