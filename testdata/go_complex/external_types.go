package complex

import (
	"database/sql"
	"encoding/json"
	"math/big"
	"net"
	"net/url"
	"sync"
	"time"
)

// --- Struct with standard library types ---

type StdLibFields struct {
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   *time.Time     `json:"deleted_at,omitempty"`
	Duration    time.Duration  `json:"duration"`
	NullString  sql.NullString `json:"null_string"`
	NullInt64   sql.NullInt64  `json:"null_int64"`
	RawMessage  json.RawMessage `json:"raw_message"`
	IP          net.IP         `json:"ip"`
	URL         *url.URL       `json:"url"`
	BigInt      *big.Int       `json:"big_int"`
	Mutex       sync.Mutex
}

// --- Struct with slices/maps of external types ---

type ExternalCollections struct {
	Timestamps   []time.Time                `json:"timestamps"`
	NullStrings  []*sql.NullString          `json:"null_strings"`
	IPMap        map[string]net.IP          `json:"ip_map"`
	DurationMap  map[string]time.Duration   `json:"duration_map"`
	URLSlice     []*url.URL                 `json:"urls"`
	NestedExtMap map[string][]*sql.NullInt64 `json:"nested_ext_map"`
}

// --- Struct embedding standard library types ---

type EmbeddedExternal struct {
	sync.Mutex
	sync.Once
	Name string `json:"name"`
}

// --- Nested external type references ---

type TimeRange struct {
	Start time.Time  `json:"start"`
	End   *time.Time `json:"end,omitempty"`
}

type ScheduleConfig struct {
	Ranges   []TimeRange          `json:"ranges"`
	Interval time.Duration        `json:"interval"`
	Timezone *time.Location       `json:"timezone"`
	Backoff  map[int]time.Duration `json:"backoff"`
}

// --- Functions with external types ---

func ParseTime(layout string, value string) (time.Time, error) {
	return time.Parse(layout, value)
}

func ResolveAddr(host string, port int) (*net.TCPAddr, error) {
	return nil, nil
}

func BatchInsert(tx *sql.Tx, items []json.RawMessage) (sql.Result, error) {
	return nil, nil
}
