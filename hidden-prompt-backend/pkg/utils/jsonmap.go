package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONMap is a map[string]any that knows how to read/write itself as
// a SQL JSON column. Unlike gox.StringObjectMap (a plain map alias with no
// driver.Valuer/sql.Scanner implementation), this type can actually be
// passed to database/sql as a query argument or Scan destination.
type JSONMap map[string]any

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(map[string]any(m))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (m *JSONMap) Scan(src any) error {
	if src == nil {
		*m = JSONMap{}
		return nil
	}

	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unsupported Scan source type %T for JSONMap", src)
	}

	if len(b) == 0 {
		*m = JSONMap{}
		return nil
	}

	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return err
	}
	*m = out
	return nil
}
