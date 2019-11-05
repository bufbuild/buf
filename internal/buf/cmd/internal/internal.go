package internal

import (
	"net/http"
	"strings"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufcheck/bufbreaking"
	"github.com/bufbuild/buf/internal/buf/bufcheck/buflint"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufos"
	"github.com/bufbuild/buf/internal/pkg/bytepool"
	"github.com/bufbuild/buf/internal/pkg/errs"
	"go.uber.org/zap"
)

var defaultHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
}

// NewBufosEnvReader returns a new bufos.EnvReader.
func NewBufosEnvReader(
	logger *zap.Logger,
	segList *bytepool.SegList,
	inputFlagName string,
	configOverrideFlagName string,
) bufos.EnvReader {
	return bufos.NewEnvReader(
		logger,
		segList,
		defaultHTTPClient,
		bufconfig.NewProvider(logger),
		bufbuild.NewHandler(
			logger,
			segList,
			bufbuild.NewProvider(logger),
			bufbuild.NewRunner(logger),
		),
		inputFlagName,
		configOverrideFlagName,
	)
}

// NewBufosImageWriter returns a new bufos.ImageWriter.
func NewBufosImageWriter(
	logger *zap.Logger,
	outputFlagName string,
) bufos.ImageWriter {
	return bufos.NewImageWriter(
		logger,
		outputFlagName,
	)
}

// NewBuflintHandler returns a new buflint.Handler.
func NewBuflintHandler(
	logger *zap.Logger,
) buflint.Handler {
	return buflint.NewHandler(
		logger,
		buflint.NewRunner(logger),
	)
}

// NewBufbreakingHandler returns a new bufbreaking.Handler.
func NewBufbreakingHandler(
	logger *zap.Logger,
) bufbreaking.Handler {
	return bufbreaking.NewHandler(
		logger,
		bufbreaking.NewRunner(logger),
	)
}

// IsFormatJSON returns true if the format is JSON.
//
// This will probably eventually need to be split between the image/check flags
// and the ls flags as we may have different formats for each.
func IsFormatJSON(flagName string, format string) (bool, error) {
	switch s := strings.TrimSpace(strings.ToLower(format)); s {
	case "text", "":
		return false, nil
	case "json":
		return true, nil
	default:
		return false, errs.NewInvalidArgumentf("--%s: unknown format: %q", flagName, s)
	}
}

// IsLintFormatJSON returns true if the format is JSON for lint.
//
// Also allows config-ignore-yaml.
func IsLintFormatJSON(flagName string, format string) (bool, error) {
	switch s := strings.TrimSpace(strings.ToLower(format)); s {
	case "text", "":
		return false, nil
	case "json":
		return true, nil
	case "config-ignore-yaml":
		return false, nil
	default:
		return false, errs.NewInvalidArgumentf("--%s: unknown format: %q", flagName, s)
	}
}

// IsLintFormatConfigIgnoreYAML returns true if the format is config-ignore-yaml.
func IsLintFormatConfigIgnoreYAML(flagName string, format string) (bool, error) {
	switch s := strings.TrimSpace(strings.ToLower(format)); s {
	case "text", "":
		return false, nil
	case "json":
		return false, nil
	case "config-ignore-yaml":
		return true, nil
	default:
		return false, errs.NewInvalidArgumentf("--%s: unknown format: %q", flagName, s)
	}
}
