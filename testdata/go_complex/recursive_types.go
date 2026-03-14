package complex

// --- Direct self-reference (tree structure) ---

type TreeNode struct {
	Value    string      `json:"value"`
	Children []*TreeNode `json:"children,omitempty"`
	Parent   *TreeNode   `json:"parent,omitempty"`
}

// --- Mutual recursion (A ↔ B) ---

type Employee struct {
	Name       string      `json:"name"`
	Department *Department `json:"department,omitempty"`
}

type Department struct {
	Name      string      `json:"name"`
	Manager   *Employee   `json:"manager,omitempty"`
	Employees []*Employee `json:"employees,omitempty"`
}

// --- Deeply nested recursion (A → B → C → A) ---

type GraphNode struct {
	ID    string       `json:"id"`
	Edges []*GraphEdge `json:"edges,omitempty"`
}

type GraphEdge struct {
	Weight int          `json:"weight"`
	Target *GraphNode   `json:"target"`
	Labels []*EdgeLabel `json:"labels,omitempty"`
}

type EdgeLabel struct {
	Text  string     `json:"text"`
	Node  *GraphNode `json:"node,omitempty"`
}

// --- Self-referencing map ---

type Config struct {
	Value    string             `json:"value,omitempty"`
	Children map[string]*Config `json:"children,omitempty"`
}

// --- Recursive with multiple self-ref paths ---

type Comment struct {
	ID       int64      `json:"id"`
	Body     string     `json:"body"`
	Replies  []*Comment `json:"replies,omitempty"`
	ReplyTo  *Comment   `json:"reply_to,omitempty"`
	BestReply *Comment  `json:"best_reply,omitempty"`
}

// --- Recursive generic type ---

type LinkedList[T any] struct {
	Value T                `json:"value"`
	Next  *LinkedList[T]   `json:"next,omitempty"`
}
