package bufconnect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTokenProviderFromStringForSingleToken(t *testing.T) {
	provider, err := NewTokenProviderFromString("default")

	require.NoError(t, err)
	require.False(t, provider.IsFromEnvVar())
	require.Equal(t, "default", provider.RemoteToken("r1"))
	require.Equal(t, "default", provider.RemoteToken("r2"))
	require.Equal(t, "default", provider.RemoteToken("r3"))
}

func TestNewTokenProviderFromStringForSingleTokenWithRemote(t *testing.T) {
	provider, err := NewTokenProviderFromString("t@r")

	require.NoError(t, err)
	require.False(t, provider.IsFromEnvVar())
	require.Equal(t, "t", provider.RemoteToken("r"))
	require.Equal(t, "", provider.RemoteToken("r2"))
}

func TestNewTokenProviderFromStringForMultipleTokens(t *testing.T) {
	provider, err := NewTokenProviderFromString("t1@r1,t2@r2")

	require.NoError(t, err)
	require.False(t, provider.IsFromEnvVar())
	require.Equal(t, "t1", provider.RemoteToken("r1"))
	require.Equal(t, "t2", provider.RemoteToken("r2"))
	require.Equal(t, "", provider.RemoteToken("r3"))
}

func TestNewTokenProviderFromStringInvalidToken(t *testing.T) {
	invalidTokens := []string{
		"t1@r1,t2@r2,default",
	}

	for _, token := range invalidTokens {
		_, err := NewTokenProviderFromString(token)
		require.Error(t, err, "expected %s to be an invalid token, but it wasn't", token)
	}
}
