package parser

import "testing"

func TestExternal_StdLibFieldTypes(t *testing.T) {
	result := parseComplex(t)

	s := findStruct(result.Structs, "StdLibFields")
	if s == nil {
		t.Fatal("StdLibFields not found")
	}

	cases := []struct {
		fieldName string
		wantType  string
	}{
		{"CreatedAt", "time.Time"},
		{"DeletedAt", "*time.Time"},
		{"Duration", "time.Duration"},
		{"NullString", "sql.NullString"},
		{"NullInt64", "sql.NullInt64"},
		{"RawMessage", "json.RawMessage"},
		{"IP", "net.IP"},
		{"URL", "*url.URL"},
		{"BigInt", "*big.Int"},
		{"Mutex", "sync.Mutex"},
	}

	for _, tc := range cases {
		f := findField(s.Fields, tc.fieldName)
		if f == nil {
			t.Errorf("field %q not found", tc.fieldName)
			continue
		}
		if f.TypeName != tc.wantType {
			t.Errorf("%s.TypeName = %q, want %q", tc.fieldName, f.TypeName, tc.wantType)
		}
	}
}

func TestExternal_CollectionsOfExternalTypes(t *testing.T) {
	result := parseComplex(t)

	s := findStruct(result.Structs, "ExternalCollections")
	if s == nil {
		t.Fatal("ExternalCollections not found")
	}

	cases := []struct {
		fieldName string
		wantType  string
	}{
		{"Timestamps", "[]time.Time"},
		{"NullStrings", "[]*sql.NullString"},
		{"IPMap", "map[string]net.IP"},
		{"DurationMap", "map[string]time.Duration"},
		{"URLSlice", "[]*url.URL"},
		{"NestedExtMap", "map[string][]*sql.NullInt64"},
	}

	for _, tc := range cases {
		f := findField(s.Fields, tc.fieldName)
		if f == nil {
			t.Errorf("field %q not found", tc.fieldName)
			continue
		}
		if f.TypeName != tc.wantType {
			t.Errorf("%s.TypeName = %q, want %q", tc.fieldName, f.TypeName, tc.wantType)
		}
	}
}

func TestExternal_EmbeddedExternalTypes(t *testing.T) {
	result := parseComplex(t)

	s := findStruct(result.Structs, "EmbeddedExternal")
	if s == nil {
		t.Fatal("EmbeddedExternal not found")
	}

	if len(s.Fields) != 3 {
		t.Fatalf("EmbeddedExternal fields len = %d, want 3", len(s.Fields))
	}

	// Embedded sync.Mutex — anonymous, selector type
	if s.Fields[0].Name != "" {
		t.Errorf("Fields[0].Name = %q, want empty (embedded)", s.Fields[0].Name)
	}
	if s.Fields[0].TypeName != "sync.Mutex" {
		t.Errorf("Fields[0].TypeName = %q, want sync.Mutex", s.Fields[0].TypeName)
	}

	// Embedded sync.Once
	if s.Fields[1].Name != "" {
		t.Errorf("Fields[1].Name = %q, want empty (embedded)", s.Fields[1].Name)
	}
	if s.Fields[1].TypeName != "sync.Once" {
		t.Errorf("Fields[1].TypeName = %q, want sync.Once", s.Fields[1].TypeName)
	}
}

func TestExternal_NestedExternalReferences(t *testing.T) {
	result := parseComplex(t)

	s := findStruct(result.Structs, "ScheduleConfig")
	if s == nil {
		t.Fatal("ScheduleConfig not found")
	}

	cases := []struct {
		fieldName string
		wantType  string
	}{
		{"Ranges", "[]TimeRange"},
		{"Interval", "time.Duration"},
		{"Timezone", "*time.Location"},
		{"Backoff", "map[int]time.Duration"},
	}

	for _, tc := range cases {
		f := findField(s.Fields, tc.fieldName)
		if f == nil {
			t.Errorf("field %q not found", tc.fieldName)
			continue
		}
		if f.TypeName != tc.wantType {
			t.Errorf("%s.TypeName = %q, want %q", tc.fieldName, f.TypeName, tc.wantType)
		}
	}
}

func TestExternal_FuncWithExternalParams(t *testing.T) {
	result := parseComplex(t)

	// func ParseTime(layout string, value string) (time.Time, error)
	fn := findFunc(result.Functions, "ParseTime")
	if fn == nil {
		t.Fatal("ParseTime not found")
	}
	if len(fn.Results) != 2 {
		t.Fatalf("ParseTime.Results len = %d, want 2", len(fn.Results))
	}
	if fn.Results[0].TypeName != "time.Time" {
		t.Errorf("Results[0].TypeName = %q, want time.Time", fn.Results[0].TypeName)
	}

	// func ResolveAddr(host string, port int) (*net.TCPAddr, error)
	fn2 := findFunc(result.Functions, "ResolveAddr")
	if fn2 == nil {
		t.Fatal("ResolveAddr not found")
	}
	if len(fn2.Results) != 2 {
		t.Fatalf("ResolveAddr.Results len = %d, want 2", len(fn2.Results))
	}
	if fn2.Results[0].TypeName != "*net.TCPAddr" {
		t.Errorf("Results[0].TypeName = %q, want *net.TCPAddr", fn2.Results[0].TypeName)
	}

	// func BatchInsert(tx *sql.Tx, items []json.RawMessage) (sql.Result, error)
	fn3 := findFunc(result.Functions, "BatchInsert")
	if fn3 == nil {
		t.Fatal("BatchInsert not found")
	}
	if len(fn3.Params) != 2 {
		t.Fatalf("BatchInsert.Params len = %d, want 2", len(fn3.Params))
	}
	if fn3.Params[0].TypeName != "*sql.Tx" {
		t.Errorf("Params[0].TypeName = %q, want *sql.Tx", fn3.Params[0].TypeName)
	}
	if fn3.Params[1].TypeName != "[]json.RawMessage" {
		t.Errorf("Params[1].TypeName = %q, want []json.RawMessage", fn3.Params[1].TypeName)
	}
	if len(fn3.Results) != 2 {
		t.Fatalf("BatchInsert.Results len = %d, want 2", len(fn3.Results))
	}
	if fn3.Results[0].TypeName != "sql.Result" {
		t.Errorf("Results[0].TypeName = %q, want sql.Result", fn3.Results[0].TypeName)
	}
}
