package complex

import "net/http"

// --- Named function type ---

type HandlerFunc func(http.ResponseWriter, *http.Request)

type Middleware func(HandlerFunc) HandlerFunc

// --- Generic types ---

type Pagination[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
}

type Result[T any, E error] struct {
	Data  *T     `json:"data,omitempty"`
	Error E      `json:"error,omitempty"`
	OK    bool   `json:"ok"`
}

// --- Generic instantiation in fields ---

type UserList struct {
	Pagination[User]
	Filter string `json:"filter"`
}

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// --- Complex nested types ---

type NestedTypes struct {
	SliceOfMaps      []map[string]int            `json:"slice_of_maps"`
	MapOfSlices      map[string][]int64           `json:"map_of_slices"`
	MapOfMaps        map[string]map[string]bool   `json:"map_of_maps"`
	MapOfPtrSlice    map[int][]*User              `json:"map_of_ptr_slice"`
	DeepNested       *[]map[string][]*User        `json:"deep_nested"`
}

// --- Channel and func fields ---

type RuntimeTypes struct {
	Callback     func(string) error          `json:"-"`
	Transform    func(int, string) (bool, error) `json:"-"`
	Events       chan string
	SendOnly     chan<- int
	RecvOnly     <-chan bool
	InlineStruct struct{}
}

// --- Type alias with = syntax ---

type StringAlias = string

type AliasedStruct struct {
	Label StringAlias `json:"label"`
}

// --- Function using generic types ---

func ListUsers(page, size int) *Pagination[User] {
	return nil
}

func ProcessResult(r Result[User, error]) bool {
	return r.OK
}
