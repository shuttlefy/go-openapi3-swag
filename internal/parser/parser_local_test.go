package parser

import "testing"

func TestLocal_TopLevelExtracted(t *testing.T) {
	result := parseComplex(t)

	for _, name := range []string{"TopLevelResult", "TopLevelRequest", "LocalTypeService"} {
		if findStruct(result.Structs, name) == nil {
			t.Errorf("top-level struct %q should be extracted", name)
		}
	}
}

func TestLocal_TopLevelHasEmptyFuncScope(t *testing.T) {
	result := parseComplex(t)

	s := findStruct(result.Structs, "TopLevelResult")
	if s == nil {
		t.Fatal("TopLevelResult not found")
	}
	if s.FuncScope != "" {
		t.Errorf("TopLevelResult.FuncScope = %q, want empty", s.FuncScope)
	}
}

func TestLocal_FunctionLocalTypesExtracted(t *testing.T) {
	result := parseComplex(t)

	// struct types inside FuncWithLocalTypes
	lr := findStructInScope(result.Structs, "localRequest", "FuncWithLocalTypes")
	if lr == nil {
		t.Fatal("localRequest in FuncWithLocalTypes not found")
	}
	if len(lr.Fields) != 2 {
		t.Errorf("localRequest fields len = %d, want 2", len(lr.Fields))
	}
	if lr.Fields[0].Name != "Field1" || lr.Fields[0].TypeName != "string" {
		t.Errorf("localRequest.Fields[0] = %+v, want {Field1 string}", lr.Fields[0])
	}

	lresp := findStructInScope(result.Structs, "localResponse", "FuncWithLocalTypes")
	if lresp == nil {
		t.Fatal("localResponse in FuncWithLocalTypes not found")
	}
	if len(lresp.Fields) != 2 {
		t.Errorf("localResponse fields len = %d, want 2", len(lresp.Fields))
	}
}

func TestLocal_FunctionLocalTypeAliasExtracted(t *testing.T) {
	result := parseComplex(t)

	le := findTypeAliasInScope(result.TypeAliases, "localEnum", "FuncWithLocalTypeAlias")
	if le == nil {
		t.Fatal("localEnum in FuncWithLocalTypeAlias not found")
	}
	if le.Underlying != "string" {
		t.Errorf("localEnum.Underlying = %q, want string", le.Underlying)
	}
}

func TestLocal_FunctionLocalConstsExtracted(t *testing.T) {
	result := parseComplex(t)

	la := findConstInScope(result.Consts, "localA", "FuncWithLocalTypeAlias")
	if la == nil {
		t.Fatal("localA in FuncWithLocalTypeAlias not found")
	}
	if la.TypeName != "localEnum" {
		t.Errorf("localA.TypeName = %q, want localEnum", la.TypeName)
	}
	if la.Value != `"a"` {
		t.Errorf("localA.Value = %q, want \"a\"", la.Value)
	}

	lb := findConstInScope(result.Consts, "localB", "FuncWithLocalTypeAlias")
	if lb == nil {
		t.Fatal("localB in FuncWithLocalTypeAlias not found")
	}
}

func TestLocal_SameNameCollision(t *testing.T) {
	result := parseComplex(t)

	// Top-level TopLevelRequest (FuncScope == "")
	topLevel := findStructInScope(result.Structs, "TopLevelRequest", "")
	if topLevel == nil {
		t.Fatal("top-level TopLevelRequest not found")
	}
	if findField(topLevel.Fields, "Input") == nil {
		t.Error("top-level TopLevelRequest should have field Input")
	}

	// Function-local TopLevelRequest (FuncScope == "FuncSameNameAsTopLevel")
	local := findStructInScope(result.Structs, "TopLevelRequest", "FuncSameNameAsTopLevel")
	if local == nil {
		t.Fatal("function-local TopLevelRequest not found")
	}
	if findField(local.Fields, "LocalOnly") == nil {
		t.Error("function-local TopLevelRequest should have field LocalOnly")
	}

	// Both exist, distinguished by FuncScope
	count := 0
	for _, s := range result.Structs {
		if s.Name == "TopLevelRequest" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("TopLevelRequest count = %d, want 2 (top-level + function-local)", count)
	}
}

func TestLocal_MethodLocalTypesExtracted(t *testing.T) {
	result := parseComplex(t)

	im := findStructInScope(result.Structs, "intermediate", "Process")
	if im == nil {
		t.Fatal("intermediate in Process method not found")
	}
	if im.FuncScope != "Process" {
		t.Errorf("intermediate.FuncScope = %q, want Process", im.FuncScope)
	}
	if len(im.Fields) != 2 {
		t.Errorf("intermediate fields len = %d, want 2", len(im.Fields))
	}
}

func TestLocal_FunctionsStillExtracted(t *testing.T) {
	result := parseComplex(t)

	fn := findFunc(result.Functions, "FuncWithLocalTypes")
	if fn == nil {
		t.Fatal("FuncWithLocalTypes not found")
	}
	assertContains(t, fn.Comments, "// @Router  /local [post]")
	if fn.Results[0].TypeName != "TopLevelResult" {
		t.Errorf("FuncWithLocalTypes result type = %q, want TopLevelResult", fn.Results[0].TypeName)
	}

	fn2 := findFunc(result.Functions, "FuncWithAnonymousStruct")
	if fn2 == nil {
		t.Fatal("FuncWithAnonymousStruct not found")
	}
	if fn2.Results[0].TypeName != "interface{}" {
		t.Errorf("FuncWithAnonymousStruct result type = %q, want interface{}", fn2.Results[0].TypeName)
	}
}

// --- scope-aware helpers ---

func findStructInScope(structs []RawStruct, name, funcScope string) *RawStruct {
	for i := range structs {
		if structs[i].Name == name && structs[i].FuncScope == funcScope {
			return &structs[i]
		}
	}
	return nil
}

func findTypeAliasInScope(aliases []RawTypeAlias, name, funcScope string) *RawTypeAlias {
	for i := range aliases {
		if aliases[i].Name == name && aliases[i].FuncScope == funcScope {
			return &aliases[i]
		}
	}
	return nil
}

func findConstInScope(consts []RawConst, name, funcScope string) *RawConst {
	for i := range consts {
		if consts[i].Name == name && consts[i].FuncScope == funcScope {
			return &consts[i]
		}
	}
	return nil
}
