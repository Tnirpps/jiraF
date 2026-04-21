package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringSlice stores string arrays as JSON in PostgreSQL JSON/JSONB columns.
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return []byte("[]"), nil
	}

	data, err := json.Marshal([]string(s))
	if err != nil {
		return nil, fmt.Errorf("marshal string slice: %w", err)
	}

	return data, nil
}

func (s *StringSlice) Scan(src any) error {
	if src == nil {
		*s = StringSlice{}
		return nil
	}

	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported StringSlice source type %T", src)
	}

	if len(data) == 0 {
		*s = StringSlice{}
		return nil
	}

	var parsed []string
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("unmarshal string slice: %w", err)
	}

	*s = StringSlice(parsed)
	return nil
}
