package zl_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/leyl1ne/UserService/internal/logger"
	"github.com/leyl1ne/UserService/internal/logger/zl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewZeroLogger_DefaultLevel(t *testing.T) {
	var buf bytes.Buffer
	l := zl.NewZerologLogger("invalid-level", &buf)

	l.Info("hello world", logger.Field{Key: "foo", Value: "bar"})

	entry := parseLog(t, &buf)
	assert.Equal(t, "info", entry["level"])
	assert.Equal(t, "hello world", entry["message"])
	assert.Equal(t, "bar", entry["foo"])
}

func parseLog(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)

	return logEntry
}
