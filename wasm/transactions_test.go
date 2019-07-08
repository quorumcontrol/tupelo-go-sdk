// +build wasm

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoIt(t *testing.T) {
	require.Nil(t, doIt())
}
