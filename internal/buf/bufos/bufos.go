// Package bufos provides abstractions to read and write images from an OS context.
//
// This is primarily meant for the CLI tool, and isn't used in a service context.
package bufos

import (
	"context"
	"io"
	"net/http"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufos/internal"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"go.uber.org/zap"
)

// Env is an environment.
type Env struct {
	// Image is the image to use.
	//
	// Validated.
	Image *imagev1beta1.Image
	// Resolver is the resolver to apply before printing paths or annotations.
	// Can be nil.
	Resolver bufbuild.ProtoRealFilePathResolver
	// Config is the config to use.
	Config *bufconfig.Config
}

// EnvReader is an env reader.
type EnvReader interface {
	// ReadEnv reads an environment.
	//
	// If specificFilePaths is empty, this builds all the files under Buf control.
	//
	// Note that includeImports will only be respected for Images if the image was
	// built with buf - if it was built with protoc, we have no way of detecting
	// what is and isn't an import.
	//
	// Note that includeSourceInfo will only be respected for Sources. We make
	// no modifications for Images.
	//
	// Annotations will be fixed per the resolver before returning.
	// If stdin is nil and this tries to read from stdin, returns user error.
	ReadEnv(
		ctx context.Context,
		stdin io.Reader,
		value string,
		configOverride string,
		specificFilePaths []string,
		specificFilePathsAllowNotExist bool,
		includeImports bool,
		includeSourceInfo bool,
	) (*Env, []*analysis.Annotation, error)
	// ReadSourceEnv reads an source environment.
	//
	// This is the same as ReadEnv but disallows image values and always builds.
	ReadSourceEnv(
		ctx context.Context,
		stdin io.Reader,
		value string,
		configOverride string,
		specificFilePaths []string,
		specificFilePathsAllowNotExist bool,
		includeImports bool,
		includeSourceInfo bool,
	) (*Env, []*analysis.Annotation, error)
	// ReadImageEnv reads an image environment.
	//
	// This is the same as ReadEnv but disallows source values and never builds.
	ReadImageEnv(
		ctx context.Context,
		stdin io.Reader,
		value string,
		configOverride string,
		specificFilePaths []string,
		specificFilePathsAllowNotExist bool,
		includeImports bool,
	) (*Env, error)

	// ListFiles lists the files.
	ListFiles(
		ctx context.Context,
		stdin io.Reader,
		value string,
		configOverride string,
	) ([]string, error)

	// GetConfig gets the config.
	GetConfig(
		ctx context.Context,
		configOverride string,
	) (*bufconfig.Config, error)
}

// NewEnvReader returns a new EnvReader.
func NewEnvReader(
	logger *zap.Logger,
	httpClient *http.Client,
	configProvider bufconfig.Provider,
	buildHandler bufbuild.Handler,
	valueFlagName string,
	configOverrideFlagName string,
) EnvReader {
	return newEnvReader(
		logger,
		httpClient,
		configProvider,
		buildHandler,
		valueFlagName,
		configOverrideFlagName,
	)
}

// ImageWriter is an image writer.
type ImageWriter interface {
	// WriteImage writes the image to the value.
	//
	// The file must be an image format.
	// This is a no-np if value is the equivalent of /dev/null.
	//
	// Validates the image before writing.
	WriteImage(
		ctx context.Context,
		stdout io.Writer,
		value string,
		asFileDescriptorSet bool,
		image *imagev1beta1.Image,
	) error
}

// NewImageWriter returns a new ImageWriter.
func NewImageWriter(
	logger *zap.Logger,
	valueFlagName string,
) ImageWriter {
	return newImageWriter(
		logger,
		valueFlagName,
	)
}

// AllFormatsToString returns all format strings.
func AllFormatsToString() string {
	return internal.AllFormatsToString()
}

// SourceFormatsToString returns source format strings.
func SourceFormatsToString() string {
	return internal.SourceFormatsToString()
}

// ImageFormatsToString returns image format strings.
func ImageFormatsToString() string {
	return internal.ImageFormatsToString()
}
