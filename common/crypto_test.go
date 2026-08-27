package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptSecretRoundTrip(t *testing.T) {
	CryptoSecret = "test-crypto-secret"
	enc, err := EncryptSecret("sk-test-key")
	require.NoError(t, err)
	require.NotEmpty(t, enc)
	require.NotEqual(t, "sk-test-key", enc)
	plain, err := DecryptSecret(enc)
	require.NoError(t, err)
	require.Equal(t, "sk-test-key", plain)
}

func TestEncryptSecretEmpty(t *testing.T) {
	enc, err := EncryptSecret("")
	require.NoError(t, err)
	require.Empty(t, enc)
}
