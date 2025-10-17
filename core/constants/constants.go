package constants

const (
	PROJECT_DIR          = "blaze"
	SCHEMA_FILE          = PROJECT_DIR + "/blaze.schema"
	MIGRATION_DIR        = PROJECT_DIR + "/migrations"
	CLIENT_DIR           = PROJECT_DIR + "/generated"
	TYPES_FILE           = CLIENT_DIR + "/types.go"
	HOOKS_FILE           = CLIENT_DIR + "/hooks.go"
	CLIENT_FILE          = CLIENT_DIR + "/client.go"
	UTIL_FILE            = CLIENT_DIR + "/utils.go"
	MIGRATION_TABLE_NAME = "_blaze_migrations"
	QUERY_FILE_NAME      = "query.sql"
	DB_MAX_CONNS_ENV     = "DB_MAX_CONNS"
	DB_MIN_CONNS_ENV     = "DB_MIN_CONNS"
	DATABASE_URI_ENV     = "DATABASE_URI"
)

const (
	RED    = "\033[31m"
	GREEN  = "\033[32m"
	YELLOW = "\033[33m"
	BLUE   = "\033[34m"
	CYAN   = "\033[36m"
	RESET  = "\033[0m"
)

type ScalarType string

const (
	INT       ScalarType = "Int"
	BIGINT    ScalarType = "BigInt"
	SMALLINT  ScalarType = "SmallInt"
	FLOAT     ScalarType = "Float"
	NUMERIC   ScalarType = "Numeric"
	STRING    ScalarType = "String"
	BOOLEAN   ScalarType = "Boolean"
	DATE      ScalarType = "Date"
	TIMESTAMP ScalarType = "Timestamp"
	JSON      ScalarType = "Json"
	BYTES     ScalarType = "Bytes"
	CHAR      ScalarType = "Char"
)

var ScalarTypes = []ScalarType{
	INT, BIGINT, SMALLINT, FLOAT, NUMERIC,
	STRING, BOOLEAN, DATE, TIMESTAMP, JSON,
	BYTES, CHAR,
}

func (st ScalarType) String() string {
	return string(st)
}

func (st ScalarType) IsValid() bool {
	for _, validType := range ScalarTypes {
		if st == validType {
			return true
		}
	}
	return false
}

const (
	KEYWORD_ENUM  = "enum"
	KEYWORD_CLASS = "class"
)

const (
	FIELD_ATTR_PRIMARY_KEY = "primaryKey"
	FIELD_ATTR_UNIQUE      = "unique"
	FIELD_ATTR_DEFAULT     = "default"
	FIELD_ATTR_RELATION    = "relation"
)

const (
	CLASS_ATTR_PRIMARY_KEY = "primaryKey"
	CLASS_ATTR_UNIQUE      = "unique"
	CLASS_ATTR_INDEX       = "index"
	CLASS_ATTR_TEXT_INDEX  = "textIndex"
	CLASS_ATTR_CHECK       = "check"
)

const (
	FIELD_KIND_SCALAR = "scalar"
	FIELD_KIND_ENUM   = "enum"
	FIELD_KIND_OBJECT = "object"
)

const (
	RELATION_ONE_TO_ONE   = "OneToOne"
	RELATION_ONE_TO_MANY  = "OneToMany"
	RELATION_MANY_TO_ONE  = "ManyToOne"
	RELATION_MANY_TO_MANY = "ManyToMany"
)

const (
	DEFAULT_NOW_CALLBACK           = "now()"
	DEFAULT_UUID_CALLBACK          = "uuid()"
	DEFAULT_AUTOINCREMENT_CALLBACK = "autoincrement()"
)

var ValidCallbacks = []string{
	DEFAULT_NOW_CALLBACK,
	DEFAULT_UUID_CALLBACK,
	DEFAULT_AUTOINCREMENT_CALLBACK,
}

const (
	ON_DELETE_CASCADE   = "Cascade"
	ON_DELETE_RESTRICT  = "Restrict"
	ON_DELETE_SET_NULL  = "SetNull"
	ON_DELETE_NO_ACTION = "NoAction"
)

const (
	ON_UPDATE_CASCADE   = "Cascade"
	ON_UPDATE_RESTRICT  = "Restrict"
	ON_UPDATE_SET_NULL  = "SetNull"
	ON_UPDATE_NO_ACTION = "NoAction"
)

const (
	INDEX_TYPE_BTREE = "BTree"
	INDEX_TYPE_TEXT  = "TextIndex"
	INDEX_TYPE_HASH  = "Hash"
	INDEX_TYPE_GIN   = "Gin"
	INDEX_TYPE_GIST  = "Gist"
)

var TypeMappings = map[string]ScalarType{
	"Int":       INT,
	"BigInt":    BIGINT,
	"SmallInt":  SMALLINT,
	"Float":     FLOAT,
	"Numeric":   NUMERIC,
	"String":    STRING,
	"Boolean":   BOOLEAN,
	"Date":      DATE,
	"Timestamp": TIMESTAMP,
	"Json":      JSON,
	"Bytes":     BYTES,
	"Char":      CHAR,
}

var PGTypeMapping = map[string]string{
	"int4":      INT.String(),
	"int8":      BIGINT.String(),
	"int2":      SMALLINT.String(),
	"float8":    FLOAT.String(),
	"numeric":   NUMERIC.String(),
	"text":      STRING.String(),
	"varchar":   STRING.String(),
	"bpchar":    CHAR.String(),
	"bool":      BOOLEAN.String(),
	"date":      DATE.String(),
	"timestamp": TIMESTAMP.String(),
	"jsonb":     JSON.String(),
	"bytea":     BYTES.String(),
	"uuid":      STRING.String(),
}

var PGConstraintActionMapping = map[string]string{
	"CASCADE":   ON_DELETE_CASCADE,
	"RESTRICT":  ON_DELETE_RESTRICT,
	"SET_NULL":  ON_DELETE_SET_NULL,
	"NO_ACTION": ON_DELETE_NO_ACTION,
}

var DMMFIndexTypeMappings = map[string]string{
	CLASS_ATTR_INDEX:      INDEX_TYPE_BTREE,
	CLASS_ATTR_TEXT_INDEX: INDEX_TYPE_TEXT,
}

var ReverseDMMFIndexTypeMappings = map[string]string{
	INDEX_TYPE_BTREE: CLASS_ATTR_INDEX,
	INDEX_TYPE_TEXT:  CLASS_ATTR_TEXT_INDEX,
}

var RelationCardinality = map[string]string{
	"OneToOne":   RELATION_ONE_TO_ONE,
	"OneToMany":  RELATION_ONE_TO_MANY,
	"ManyToOne":  RELATION_MANY_TO_ONE,
	"ManyToMany": RELATION_MANY_TO_MANY,
}

var ValidOnDeleteActions = []string{
	ON_DELETE_CASCADE,
	ON_DELETE_RESTRICT,
	ON_DELETE_SET_NULL,
	ON_DELETE_NO_ACTION,
}

var ValidOnUpdateActions = []string{
	ON_UPDATE_CASCADE,
	ON_UPDATE_RESTRICT,
	ON_UPDATE_SET_NULL,
	ON_UPDATE_NO_ACTION,
}

func IsScalarType(typeName string) bool {
	_, exists := TypeMappings[typeName]
	return exists
}

func GetScalarType(typeName string) (ScalarType, bool) {
	scalarType, exists := TypeMappings[typeName]
	return scalarType, exists
}

func GetRelationType(isArray bool, isOptional bool, hasBackReference bool) string {
	if isArray {
		return RELATION_ONE_TO_MANY
	}
	if hasBackReference {
		return RELATION_MANY_TO_ONE
	}
	return RELATION_ONE_TO_ONE
}

func IsValidCallback(callback string) bool {
	for _, validCallback := range ValidCallbacks {
		if callback == validCallback {
			return true
		}
	}
	return false
}

func IsValidOnDeleteAction(action string) bool {
	for _, validAction := range ValidOnDeleteActions {
		if action == validAction {
			return true
		}
	}
	return false
}

func IsValidOnUpdateAction(action string) bool {
	for _, validAction := range ValidOnUpdateActions {
		if action == validAction {
			return true
		}
	}
	return false
}

func GetCallbackCompatibleTypes(callback string) []ScalarType {
	switch callback {
	case DEFAULT_NOW_CALLBACK:
		return []ScalarType{TIMESTAMP, DATE}
	case DEFAULT_UUID_CALLBACK:
		return []ScalarType{STRING}
	case DEFAULT_AUTOINCREMENT_CALLBACK:
		return []ScalarType{INT, BIGINT, SMALLINT}
	default:
		return []ScalarType{}
	}
}

func IsCallbackCompatibleWithType(callback string, fieldType ScalarType) bool {
	compatibleTypes := GetCallbackCompatibleTypes(callback)
	for _, compatibleType := range compatibleTypes {
		if fieldType == compatibleType {
			return true
		}
	}
	return false
}