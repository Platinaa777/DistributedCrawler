package pg

import (
	"github.com/jackc/pgconn"
	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgx/v4"
)

// mergedRows wraps multiple pgx.Rows results into a single pgx.Rows,
// iterating through each set sequentially.
type mergedRows struct {
	sets    []pgx.Rows
	current int
}

func (m *mergedRows) Close() {
	for _, s := range m.sets {
		s.Close()
	}
}

func (m *mergedRows) Err() error {
	for _, s := range m.sets {
		if err := s.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (m *mergedRows) Next() bool {
	for m.current < len(m.sets) {
		if m.sets[m.current].Next() {
			return true
		}
		m.current++
	}
	return false
}

func (m *mergedRows) Scan(dest ...any) error {
	return m.sets[m.current].Scan(dest...)
}

func (m *mergedRows) Values() ([]any, error) {
	return m.sets[m.current].Values()
}

func (m *mergedRows) RawValues() [][]byte {
	return m.sets[m.current].RawValues()
}

func (m *mergedRows) FieldDescriptions() []pgproto3.FieldDescription {
	if len(m.sets) == 0 {
		return nil
	}
	return m.sets[0].FieldDescriptions()
}

func (m *mergedRows) CommandTag() pgconn.CommandTag {
	if m.current < len(m.sets) {
		return m.sets[m.current].CommandTag()
	}
	return pgconn.CommandTag{}
}
