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
			content.WriteString(fmt.Sprintf("\t%s %s = \"%s\"\n", valueName, eg.enum.Name, valueName))
		} else {
			content.WriteString(fmt.Sprintf("\t%s %s = \"%s\"\n", valueName, eg.enum.Name, valueName))
		}
	}
	content.WriteString(")\n\n")

	return content.String()
}