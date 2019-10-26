package internal

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/pkg/errs"
)

type configOverrideParser struct {
	configProvider         bufconfig.Provider
	configOverrideFlagName string
}

func newConfigOverrideParser(
	configProvider bufconfig.Provider,
	configOverrideFlagName string,
) *configOverrideParser {
	return &configOverrideParser{
		configProvider:         configProvider,
		configOverrideFlagName: configOverrideFlagName,
	}
}

func (c *configOverrideParser) ParseConfigOverride(value string) (*bufconfig.Config, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errs.NewInternal("config override value is empty")
	}
	var data []byte
	var err error
	switch filepath.Ext(value) {
	case ".json", ".yaml":
		data, err = ioutil.ReadFile(value)
		if err != nil {
			return nil, newConfigOverrideCouldNotReadFileError(c.configOverrideFlagName, err)
		}
	default:
		data = []byte(value)
	}
	config, err := c.configProvider.GetConfigForData(data)
	if err != nil {
		return nil, newConfigOverrideCouldNotParseError(c.configOverrideFlagName, err)
	}
	return config, nil
}

func newConfigOverrideCouldNotReadFileError(configOverrideFlagName string, err error) error {
	return errs.NewInvalidArgumentf("%s: could not read file: %v", configOverrideFlagName, err)
}

func newConfigOverrideCouldNotParseError(configOverrideFlagName string, err error) error {
	return errs.NewInvalidArgumentf("%s: %v", configOverrideFlagName, err)
}
