package bufpolicy

import (
	"context"
	"io/fs"

	"github.com/bufbuild/buf/private/bufpkg/bufparse"
	"github.com/bufbuild/buf/private/bufpkg/bufplugin"
)

var (
	// NopPolicyPluginKeyProvider is a no-op PolicyPluginKeyProvider.
	NopPolicyPluginKeyProvider PolicyPluginKeyProvider = nopPolicyPluginKeyProvider{}
)

// PolicyPluginKeyProvider provides PluginKeys for a specific policy.
type PolicyPluginKeyProvider interface {
	// GetPolicyKeysForPolicyRefs gets the PolicyKeys for the given plugin Refs.
	//
	// Returned PolicyKeys will be in the same order as the input Refs.
	//
	// The input Refs are expected to be unique by FullName. The implementation
	// may error if this is not the case.
	//
	// If there is no error, the length of the PolicyKeys returned will match the length of the Refs.
	// If there is an error, no PolicyKeys will be returned.
	// If any Ref is not found, an error with fs.ErrNotExist will be returned.
	GetPolicyPluginKeysForPluginRefs(
		context.Context,
		PolicyKey,
		[]bufparse.Ref,
		bufplugin.DigestType,
	) ([]bufplugin.PluginKey, error)
}

// *** PRIVATE ***

type nopPolicyPluginKeyProvider struct{}

var _ PolicyPluginKeyProvider = nopPolicyPluginKeyProvider{}

func (nopPolicyPluginKeyProvider) GetPolicyPluginKeysForPluginRefs(
	context.Context,
	PolicyKey,
	[]bufparse.Ref,
	bufplugin.DigestType,
) ([]bufplugin.PluginKey, error) {
	return nil, fs.ErrNotExist
}
