package parser

type RawAST struct {
	Package     string
	Packages    []string // all distinct package names seen across scanned files
	Functions   []RawFunc
	Structs     []RawStruct
	TypeAliases []RawTypeAlias
	Consts      []RawConst
}

type RawFunc struct {
	Name     string
	FilePath string
	Line     int
	Comments []string
	Receiver string
	Params   []RawParam
	Results  []RawParam
}

type RawStruct struct {
	Name        string
	PackageName string // Go package name of the file that declares this struct
	FilePath    string
	FuncScope   string // enclosing function name; empty for top-level
	Fields      []RawField
	Comments    []string
}

type RawField struct {
	Name      string
	TypeName  string
	Tag       string
	Comments  []string

	// Parsed from struct tags
	JSONName    string   // `json:"name"`, "-" means excluded
	Required    bool     // `binding:"required"` or `validate:"required"`
	Omitempty   bool     // `json:"name,omitempty"`
	Example     string   // `example:"value"`
	Enums       []string // `enums:"a,b,c"`
	Format      string   // `format:"date-time"`
	Default     string   // `default:"active"`
	Description string   // `description:"field doc"`
	ReadOnly    bool     // `readonly:"true"`
	WriteOnly   bool     // `writeonly:"true"`
	Deprecated  bool     // `deprecated:"true"`
	Minimum     *float64 // `minimum:"0"`
	Maximum     *float64 // `maximum:"100"`
	MinLength   *int64   // `minLength:"1"`
	MaxLength   *int64   // `maxLength:"255"`
	Pattern     string   // `pattern:"^[a-z]+$"`
	MinItems    *int64   // `minItems:"1"`
	MaxItems    *int64   // `maxItems:"50"`
	UniqueItems bool     // `uniqueItems:"true"`
}

type RawParam struct {
	Name     string
	TypeName string
}

type RawTypeAlias struct {
	Name        string
	PackageName string
	Underlying  string
	FilePath    string
	FuncScope   string
	Comments    []string
}

type RawConst struct {
	Name        string
	PackageName string
	TypeName    string
	Value       string
	FilePath    string
	FuncScope   string
	Comments    []string
}
