package ast

type Schema struct {
	Enums   []Enum   `json:"enums"`
	Classes []Class  `json:"classes"`
	Plugins Plugins  `json:"plugins"`
	Errors  []string `json:"errors,omitempty"`
}

type Plugins struct {
	PgCrypto bool `json:"pgcrypto"`
	PgTrgm   bool `json:"pg_trgm"`
}

type Enum struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
	Line   int      `json:"line"`
}

type Class struct {
	Name        string     `json:"name"`
	Fields      []Field    `json:"fields"`
	Relations   []Relation `json:"relations"`
	Indexes     []Index    `json:"indexes"`
	Constraints []string   `json:"constraints"`
	PrimaryKey  []string   `json:"primary_key,omitempty"`
	Line        int        `json:"line"`
}

type Field struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Kind         string            `json:"kind"`
	IsOptional   bool              `json:"is_optional"`
	IsArray      bool              `json:"is_array"`
	IsPrimaryKey bool              `json:"is_primary_key"`
	IsUnique     bool              `json:"is_unique"`
	Default      *string           `json:"default,omitempty"`
	Constraints  []string          `json:"constraints,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Line         int               `json:"line"`
}

type Relation struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	RelatedClass string   `json:"related_class"`
	Fields       []string `json:"fields,omitempty"`
	References   []string `json:"references,omitempty"`
	OnDelete     string   `json:"on_delete,omitempty"`
	OnUpdate     string   `json:"on_update,omitempty"`
	RelationName string   `json:"relation_name,omitempty"`
	Line         int      `json:"line"`
}

type Index struct {
	Name     string   `json:"name,omitempty"`
	Fields   []string `json:"fields"`
	IsUnique bool     `json:"is_unique"`
	Type     string   `json:"type"`
	Line     int      `json:"line"`
}