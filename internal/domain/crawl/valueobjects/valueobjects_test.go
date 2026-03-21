package valueobjects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseID_Behavior(t *testing.T) {
	t.Parallel()

	id := GenerateID()
	require.NotEmpty(t, id.String())
	assert.False(t, id.IsEmpty())
	assert.True(t, id.Equals(id))

	parsed, err := NewID(id.String())
	require.NoError(t, err)
	assert.Equal(t, id.String(), parsed.String())

	_, err = NewID("")
	require.ErrorIs(t, err, ErrEmptyID)
	_, err = NewID("not-a-uuid")
	require.ErrorIs(t, err, ErrInvalidID)
}

func TestTypedIDs_ConstructorsAndGenerators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		new  func(string) (string, error)
		gen  func() string
	}{
		{
			name: "crawl job",
			new: func(raw string) (string, error) {
				id, err := NewCrawlJobID(raw)
				return id.String(), err
			},
			gen: func() string { return GenerateCrawlJobID().String() },
		},
		{
			name: "crawl task",
			new: func(raw string) (string, error) {
				id, err := NewCrawlTaskID(raw)
				return id.String(), err
			},
			gen: func() string { return GenerateCrawlTaskID().String() },
		},
		{
			name: "preview",
			new: func(raw string) (string, error) {
				id, err := NewPreviewID(raw)
				return id.String(), err
			},
			gen: func() string { return GeneratePreviewID().String() },
		},
		{
			name: "page snapshot",
			new: func(raw string) (string, error) {
				id, err := NewPageSnapshotID(raw)
				return id.String(), err
			},
			gen: func() string { return GeneratePageSnapshotID().String() },
		},
		{
			name: "page link",
			new: func(raw string) (string, error) {
				id, err := NewPageLinkID(raw)
				return id.String(), err
			},
			gen: func() string { return GeneratePageLinkID().String() },
		},
		{
			name: "page image",
			new: func(raw string) (string, error) {
				id, err := NewPageImageID(raw)
				return id.String(), err
			},
			gen: func() string { return GeneratePageImageID().String() },
		},
		{
			name: "outbox aggregate",
			new: func(raw string) (string, error) {
				id, err := NewExtractedRecordID(raw)
				return id.String(), err
			},
			gen: func() string { return GenerateExtractedRecordID().String() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw := tt.gen()
			parsed, err := tt.new(raw)
			require.NoError(t, err)
			assert.Equal(t, raw, parsed)
		})
	}

	outboxID := GenerateOutboxEventID()
	require.NotEmpty(t, outboxID.String())
	parsedOutboxID, err := NewOutboxEventID(outboxID.String())
	require.NoError(t, err)
	assert.Equal(t, outboxID.String(), parsedOutboxID.String())
}

