package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
)

type opensearchCore struct {
	zapcore.LevelEnabler
	endpoint      string
	index         string
	client        *http.Client
	buf           []map[string]any
	mu            sync.Mutex
	batchSize     int
	flushInterval time.Duration
	stopCh        chan struct{}
	fields        []zapcore.Field
}

// NewOpenSearchCore creates a zap core that sends logs to OpenSearch via bulk API.
func NewOpenSearchCore(level zapcore.Level, endpoint, index string, batchSize int, flushIntervalSec int) zapcore.Core {
	c := &opensearchCore{
		LevelEnabler:  level,
		endpoint:      endpoint,
		index:         index,
		client:        &http.Client{Timeout: 10 * time.Second},
		buf:           make([]map[string]any, 0, batchSize),
		batchSize:     batchSize,
		flushInterval: time.Duration(flushIntervalSec) * time.Second,
		stopCh:        make(chan struct{}),
	}

	go c.runFlusher()

	return c
}

func (c *opensearchCore) With(fields []zapcore.Field) zapcore.Core {
	clone := &opensearchCore{
		LevelEnabler:  c.LevelEnabler,
		endpoint:      c.endpoint,
		index:         c.index,
		client:        c.client,
		buf:           c.buf,
		mu:            sync.Mutex{},
		batchSize:     c.batchSize,
		flushInterval: c.flushInterval,
		stopCh:        c.stopCh,
		fields:        append(c.fields[:len(c.fields):len(c.fields)], fields...),
	}
	return clone
}

func (c *opensearchCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.After(entry, c)
	}
	return ce
}

func (c *opensearchCore) OnWrite(_ *zapcore.CheckedEntry, _ []zapcore.Field) {}

func (c *opensearchCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	doc := map[string]any{
		"@timestamp": entry.Time.UTC().Format(time.RFC3339Nano),
		"level":      entry.Level.String(),
		"message":    entry.Message,
		"logger":     entry.LoggerName,
	}

	if entry.Caller.Defined {
		doc["caller"] = entry.Caller.TrimmedPath()
	}
	if entry.Stack != "" {
		doc["stacktrace"] = entry.Stack
	}

	enc := zapcore.NewMapObjectEncoder()
	for _, f := range c.fields {
		f.AddTo(enc)
	}
	for _, f := range fields {
		f.AddTo(enc)
	}
	for k, v := range enc.Fields {
		doc[k] = v
	}

	c.mu.Lock()
	c.buf = append(c.buf, doc)
	shouldFlush := len(c.buf) >= c.batchSize
	c.mu.Unlock()

	if shouldFlush {
		c.flush()
	}

	return nil
}

func (c *opensearchCore) Sync() error {
	c.flush()
	return nil
}

// Stop stops the background flusher and flushes remaining logs.
func (c *opensearchCore) Stop() {
	close(c.stopCh)
	c.flush()
}

func (c *opensearchCore) runFlusher() {
	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.flush()
		case <-c.stopCh:
			return
		}
	}
}

func (c *opensearchCore) flush() {
	c.mu.Lock()
	if len(c.buf) == 0 {
		c.mu.Unlock()
		return
	}
	batch := c.buf
	c.buf = make([]map[string]any, 0, c.batchSize)
	c.mu.Unlock()

	c.sendBulk(batch)
}

func (c *opensearchCore) sendBulk(docs []map[string]any) {
	var body bytes.Buffer

	index := fmt.Sprintf("%s-%s", c.index, time.Now().UTC().Format("2006.01.02"))

	for _, doc := range docs {
		meta := map[string]any{
			"index": map[string]any{
				"_index": index,
			},
		}
		metaLine, _ := json.Marshal(meta)
		docLine, _ := json.Marshal(doc)

		body.Write(metaLine)
		body.WriteByte('\n')
		body.Write(docLine)
		body.WriteByte('\n')
	}

	req, err := http.NewRequest(http.MethodPost, c.endpoint+"/_bulk", &body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := c.client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
