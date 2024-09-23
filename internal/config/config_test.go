package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFromEnv(t *testing.T) {
	// Given
	// When

	os.Setenv("SUBROUTINES_CREATOR_FGA_GRPC_ADDR", "0.0.0.0")

	_, err := NewFromEnv()
	// Then
	assert.NoError(t, err)
}
