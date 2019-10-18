package bufconfig

import (
	"context"
	"io/ioutil"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/pkg/encodingutil"
	"github.com/bufbuild/buf/internal/pkg/logutil"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type provider struct {
	logger                 *zap.Logger
	externalConfigModifier func(*ExternalConfig) error
}

func newProvider(logger *zap.Logger, options ...ProviderOption) *provider {
	provider := &provider{
		logger: logger.Named("config"),
	}
	for _, option := range options {
		option(provider)
	}
	return provider
}

func (p *provider) GetConfigForBucket(ctx context.Context, bucket storage.ReadBucket) (_ *Config, retErr error) {
	defer logutil.Defer(p.logger, "get_config_for_bucket")()

	externalConfig := &ExternalConfig{}
	readObject, err := bucket.Get(ctx, ConfigFilePath)
	if err != nil {
		if storage.IsNotExist(err) {
			return p.newConfig(externalConfig)
		}
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readObject.Close())
	}()
	data, err := ioutil.ReadAll(readObject)
	if err != nil {
		return nil, err
	}
	if err := encodingutil.UnmarshalYAMLStrict(data, externalConfig); err != nil {
		return nil, err
	}
	return p.newConfig(externalConfig)
}

func (p *provider) GetConfigForData(data []byte) (*Config, error) {
	defer logutil.Defer(p.logger, "get_config_for_data")()

	externalConfig := &ExternalConfig{}
	if err := encodingutil.UnmarshalJSONOrYAMLStrict(data, externalConfig); err != nil {
		return nil, err
	}
	return p.newConfig(externalConfig)
}

func (p *provider) newConfig(externalConfig *ExternalConfig) (*Config, error) {
	if p.externalConfigModifier != nil {
		if err := p.externalConfigModifier(externalConfig); err != nil {
			return nil, err
		}
	}
	buildConfig, err := bufbuild.ConfigBuilder{
		Roots:    externalConfig.Build.Roots,
		Excludes: externalConfig.Build.Excludes,
	}.NewConfig()
	if err != nil {
		return nil, err
	}
	breakingConfig, err := bufbreaking.ConfigBuilder{
		Use:                           externalConfig.Breaking.Use,
		Except:                        externalConfig.Breaking.Except,
		IgnoreRootPaths:               externalConfig.Breaking.Ignore,
		IgnoreIDOrCategoryToRootPaths: externalConfig.Breaking.IgnoreOnly,
	}.NewConfig()
	if err != nil {
		return nil, err
	}
	lintConfig, err := buflint.ConfigBuilder{
		Use:                                  externalConfig.Lint.Use,
		Except:                               externalConfig.Lint.Except,
		IgnoreRootPaths:                      externalConfig.Lint.Ignore,
		IgnoreIDOrCategoryToRootPaths:        externalConfig.Lint.IgnoreOnly,
		EnumZeroValueSuffix:                  externalConfig.Lint.EnumZeroValueSuffix,
		RPCAllowSameRequestResponse:          externalConfig.Lint.RPCAllowSameRequestResponse,
		RPCAllowGoogleProtobufEmptyRequests:  externalConfig.Lint.RPCAllowGoogleProtobufEmptyRequests,
		RPCAllowGoogleProtobufEmptyResponses: externalConfig.Lint.RPCAllowGoogleProtobufEmptyResponses,
		ServiceSuffix:                        externalConfig.Lint.ServiceSuffix,
	}.NewConfig()
	if err != nil {
		return nil, err
	}
	return &Config{
		Build:    buildConfig,
		Breaking: breakingConfig,
		Lint:     lintConfig,
	}, nil
}
