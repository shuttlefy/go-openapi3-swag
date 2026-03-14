package complex

import "fmt"

// --- Top-level type ---

type TopLevelResult struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// FuncWithLocalTypes defines types inside its body.
// @Summary Do something
// @Router  /local [post]
func FuncWithLocalTypes(input string) TopLevelResult {
	type localRequest struct {
		Field1 string
		Field2 int
	}

	type localResponse struct {
		OK      bool
		Message string
	}

	req := localRequest{Field1: input, Field2: 42}
	_ = req

	return TopLevelResult{Code: 0, Msg: "ok"}
}

// FuncWithLocalTypeAlias defines a type alias inside its body.
func FuncWithLocalTypeAlias() {
	type localEnum string

	const (
		localA localEnum = "a"
		localB localEnum = "b"
	)

	fmt.Println(localA, localB)
}

// FuncWithAnonymousStruct uses an anonymous struct inline.
func FuncWithAnonymousStruct() interface{} {
	result := struct {
		Name  string
		Value int
	}{
		Name:  "test",
		Value: 1,
	}
	return result
}

// --- Top-level type with same name as a function-local type ---

type TopLevelRequest struct {
	Input string `json:"input"`
}

// FuncSameNameAsTopLevel has a local type whose name collides with a top-level type.
func FuncSameNameAsTopLevel() {
	type TopLevelRequest struct {
		LocalOnly string
	}
	r := TopLevelRequest{LocalOnly: "hidden"}
	_ = r
}

// --- Method with local types ---

type LocalTypeService struct{}

func (s *LocalTypeService) Process(data string) TopLevelResult {
	type intermediate struct {
		parsed bool
		data   string
	}
	im := intermediate{parsed: true, data: data}
	_ = im
	return TopLevelResult{Code: 0, Msg: data}
}
