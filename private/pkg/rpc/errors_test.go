package rpc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetErrorMessage(t *testing.T) {
	require.Equal(t, "test", GetErrorMessage(fmt.Errorf("some error: %w", NewInvalidArgumentError("test"))))
}
