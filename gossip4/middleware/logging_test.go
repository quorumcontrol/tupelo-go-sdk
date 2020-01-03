package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestSetLogLevel(t *testing.T) {
	err := SetLogLevel("error")
	assert.Nil(t, err)
	assert.True(t, Log.Desugar().Core().Enabled(zapcore.ErrorLevel))
	assert.False(t, Log.Desugar().Core().Enabled(zapcore.DebugLevel))

	err = SetLogLevel("debug")
	assert.Nil(t, err)
	assert.True(t, Log.Desugar().Core().Enabled(zapcore.ErrorLevel))
	assert.True(t, Log.Desugar().Core().Enabled(zapcore.DebugLevel))
}
