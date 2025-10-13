package migration

import (
	"github.com/rit3sh-x/blaze/core/constants"
)

func (me *MigrationEngine) generateExtensions() []MigrationStatement {
	var statements []MigrationStatement

	needsPgTrgm := me.needsPgTrgmExtension()
	if needsPgTrgm {
		statements = append(statements, MigrationStatement{
			SQL:      "CREATE EXTENSION IF NOT EXISTS pg_trgm",
			Type:     "extension",
			Priority: 1,
		})
	}

	needsPgCrypto := me.needsPgCryptoExtension()
	if needsPgCrypto {
		statements = append(statements, MigrationStatement{
			SQL:      "CREATE EXTENSION IF NOT EXISTS pgcrypto",
			Type:     "extension",
			Priority: 1,
		})
	}

	return statements
}

func (me *MigrationEngine) needsPgTrgmExtension() bool {
	for _, cls := range me.toSchema.Classes {
		if cls.Attributes.HasTextIndex() {
			return true
		}
	}
	return false
}

func (me *MigrationEngine) needsPgCryptoExtension() bool {
	for _, cls := range me.toSchema.Classes {
		for _, field := range cls.Attributes.Fields {
			if field.HasDefault() && field.AttributeDefinition.DefaultValue != nil && field.AttributeDefinition.DefaultValue.Value == constants.DEFAULT_UUID_CALLBACK {
				return true
			}
		}
	}
	return false
}