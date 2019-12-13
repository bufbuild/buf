package bufos

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/buferrs"
	"github.com/bufbuild/buf/internal/buf/bufos/internal"
	"github.com/bufbuild/buf/internal/buf/bufpb"
	"github.com/bufbuild/buf/internal/pkg/analysis"
	"github.com/bufbuild/buf/internal/pkg/logutil"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagegit"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"github.com/bufbuild/cli/clios"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type envReader struct {
	logger               *zap.Logger
	httpClient           *http.Client
	configProvider       bufconfig.Provider
	buildHandler         bufbuild.Handler
	inputRefParser       internal.InputRefParser
	configOverrideParser internal.ConfigOverrideParser
}

func newEnvReader(
	logger *zap.Logger,
	httpClient *http.Client,
	configProvider bufconfig.Provider,
	buildHandler bufbuild.Handler,
	valueFlagName string,
	configOverrideFlagName string,
) *envReader {
	return &envReader{
		logger:         logger.Named("bufos"),
		httpClient:     httpClient,
		configProvider: configProvider,
		buildHandler:   buildHandler,
		inputRefParser: internal.NewInputRefParser(
			valueFlagName,
		),
		configOverrideParser: internal.NewConfigOverrideParser(
			configProvider,
			configOverrideFlagName,
		),
	}
}

func (e *envReader) ReadEnv(
	ctx context.Context,
	stdin io.Reader,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
) (*Env, []*analysis.Annotation, error) {
	return e.readEnv(
		ctx,
		stdin,
		value,
		configOverride,
		specificFilePaths,
		specificFilePathsAllowNotExist,
		includeImports,
		includeSourceInfo,
		false,
		false,
	)
}

func (e *envReader) ReadSourceEnv(
	ctx context.Context,
	stdin io.Reader,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
) (*Env, []*analysis.Annotation, error) {
	return e.readEnv(
		ctx,
		stdin,
		value,
		configOverride,
		specificFilePaths,
		specificFilePathsAllowNotExist,
		includeImports,
		includeSourceInfo,
		true,
		false,
	)
}

func (e *envReader) ReadImageEnv(
	ctx context.Context,
	stdin io.Reader,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
) (*Env, error) {
	env, annotations, err := e.readEnv(
		ctx,
		stdin,
		value,
		configOverride,
		specificFilePaths,
		specificFilePathsAllowNotExist,
		includeImports,
		false,
		false,
		true,
	)
	if err != nil {
		return nil, err
	}
	if len(annotations) > 0 {
		// TODO: need to refactor this
		return nil, buferrs.NewSystemError("got annotations for ReadImageEnv which should be impossible")
	}
	return env, nil
}

func (e *envReader) ListFiles(
	ctx context.Context,
	stdin io.Reader,
	value string,
	configOverride string,
) (_ []string, retErr error) {
	inputRef, err := e.inputRefParser.ParseInputRef(value, false, false)
	if err != nil {
		return nil, err
	}
	e.logger.Debug("parse", zap.Any("input_ref", inputRef), zap.Stringer("format", inputRef.Format))

	if inputRef.Format.IsImage() {
		// if we have an image, list the files in the image
		image, err := e.getImage(ctx, stdin, inputRef)
		if err != nil {
			return nil, err
		}
		files := image.GetFile()
		filePaths := make([]string, len(files))
		for i, file := range image.GetFile() {
			filePaths[i] = file.GetName()
		}
		sort.Strings(filePaths)
		return filePaths, nil
	}

	// we have a source, we need to get everything
	bucket, err := e.getBucket(ctx, stdin, inputRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, bucket.Close())
	}()
	var config *bufconfig.Config
	if configOverride != "" {
		config, err = e.configOverrideParser.ParseConfigOverride(configOverride)
		if err != nil {
			return nil, err
		}
	} else {
		// if there is no config override, we read the config from the bucket
		// if there was no file, this just returns default config
		config, err = e.configProvider.GetConfigForBucket(ctx, bucket)
		if err != nil {
			return nil, err
		}
	}

	filePaths, err := e.buildHandler.ListFiles(ctx, bucket, config.Build.Roots, config.Build.Excludes)
	if err != nil {
		return nil, err
	}
	if inputRef.Format != internal.FormatDir {
		// if format is not a directory, just output the file paths
		return filePaths, nil
	}

	// if we built a directory, we need to resolve file paths
	resolver, err := internal.NewRelProtoFilePathResolver(inputRef.Path, nil)
	if err != nil {
		return nil, err
	}
	for i, filePath := range filePaths {
		resolvedFilePath, err := resolver.GetFilePath(filePath)
		if err != nil {
			// This is an internal error if we cannot resolve this file path.
			return nil, buferrs.NewSystemError(err.Error())
		}
		filePaths[i] = resolvedFilePath
	}
	return filePaths, nil
}

func (e *envReader) GetConfig(
	ctx context.Context,
	configOverride string,
) (*bufconfig.Config, error) {
	if configOverride != "" {
		return e.configOverrideParser.ParseConfigOverride(configOverride)
	}
	// if there is no config override, we read the config from the current directory
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadFile(filepath.Join(pwd, bufconfig.ConfigFilePath))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// just in case
		data = nil
	}
	// if there was no file, this just returns default config
	return e.configProvider.GetConfigForData(data)
}

func (e *envReader) readEnv(
	ctx context.Context,
	stdin io.Reader,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
	onlySources bool,
	onlyImages bool,
) (_ *Env, _ []*analysis.Annotation, retErr error) {
	inputRef, err := e.inputRefParser.ParseInputRef(value, onlySources, onlyImages)
	if err != nil {
		return nil, nil, err
	}
	e.logger.Debug("parse", zap.Any("input_ref", inputRef), zap.Stringer("format", inputRef.Format))

	if inputRef.Format.IsImage() {
		env, err := e.readEnvFromImage(
			ctx,
			stdin,
			configOverride,
			specificFilePaths,
			specificFilePathsAllowNotExist,
			includeImports,
			inputRef,
		)
		return env, nil, err
	}
	return e.readEnvFromBucket(
		ctx,
		stdin,
		configOverride,
		specificFilePaths,
		specificFilePathsAllowNotExist,
		includeImports,
		includeSourceInfo,
		inputRef,
	)
}

func (e *envReader) readEnvFromBucket(
	ctx context.Context,
	stdin io.Reader,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
	inputRef *internal.InputRef,
) (_ *Env, _ []*analysis.Annotation, retErr error) {
	bucket, err := e.getBucket(ctx, stdin, inputRef)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, bucket.Close())
	}()

	var config *bufconfig.Config
	if configOverride != "" {
		config, err = e.configOverrideParser.ParseConfigOverride(configOverride)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// if there is no config override, we read the config from the bucket
		// if there was no file, this just returns default config
		config, err = e.configProvider.GetConfigForBucket(ctx, bucket)
		if err != nil {
			return nil, nil, err
		}
	}
	var specificRealFilePaths []string
	if len(specificFilePaths) > 0 {
		// since we are doing a build, we filter before doing the build
		// via bufbuild.Provider
		// this will include imports if necessary
		specificRealFilePaths = make([]string, len(specificFilePaths))
		if inputRef.Format == internal.FormatDir {
			// if we had a directory input, then we need to make everything relative to that directory
			absDirPath, err := filepath.Abs(inputRef.Path)
			if err != nil {
				return nil, nil, err
			}
			for i, specificFilePath := range specificFilePaths {
				absSpecificFilePath, err := filepath.Abs(specificFilePath)
				if err != nil {
					return nil, nil, err
				}
				rel, err := filepath.Rel(absDirPath, absSpecificFilePath)
				if err != nil {
					return nil, nil, err
				}
				specificRealFilePath, err := storagepath.NormalizeAndValidate(rel)
				if err != nil {
					return nil, nil, err
				}
				specificRealFilePaths[i] = specificRealFilePath
			}
		} else {
			// if we did not have a directory input, then we need to make sure all paths are normalized
			// and relative
			for i, specificFilePath := range specificFilePaths {
				specificRealFilePath, err := storagepath.NormalizeAndValidate(specificFilePath)
				if err != nil {
					return nil, nil, err
				}
				specificRealFilePaths[i] = specificRealFilePath
			}
		}
	}
	// we now have everything we need, actually build the image
	image, rootResolver, annotations, err := e.buildHandler.BuildImage(
		ctx,
		bucket,
		config.Build.Roots,
		config.Build.Excludes,
		specificRealFilePaths,
		specificFilePathsAllowNotExist,
		includeImports,
		includeSourceInfo,
	)
	if err != nil {
		return nil, nil, err
	}
	resolver := rootResolver
	if inputRef.Format == internal.FormatDir {
		resolver, err = internal.NewRelProtoFilePathResolver(inputRef.Path, rootResolver)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(annotations) > 0 {
		// the documentation for EnvReader says we will resolve before returning
		if err := bufbuild.FixAnnotationFilenames(resolver, annotations); err != nil {
			return nil, nil, err
		}
		return nil, annotations, nil
	}
	return &Env{Image: image, Resolver: resolver, Config: config}, nil, nil
}

func (e *envReader) readEnvFromImage(
	ctx context.Context,
	stdin io.Reader,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	inputRef *internal.InputRef,
) (_ *Env, retErr error) {
	image, err := e.getImage(ctx, stdin, inputRef)
	if err != nil {
		return nil, err
	}
	config, err := e.GetConfig(ctx, configOverride)
	if err != nil {
		return nil, err
	}
	if len(specificFilePaths) > 0 {
		// note this must include imports if these are required for whatever operation
		// you are doing
		image, err = image.WithSpecificNames(specificFilePathsAllowNotExist, specificFilePaths...)
		if err != nil {
			return nil, err
		}
	}
	if !includeImports {
		image, err = image.WithoutImports()
		if err != nil {
			return nil, err
		}
	}
	return &Env{
		Image:  image,
		Config: config,
	}, nil
}

func (e *envReader) getBucket(
	ctx context.Context,
	stdin io.Reader,
	inputRef *internal.InputRef,
) (storage.ReadBucket, error) {
	switch inputRef.Format {
	case internal.FormatDir:
		return e.getBucketFromLocalDir(inputRef.Path)
	case internal.FormatTar, internal.FormatTarGz:
		return e.getBucketFromLocalTarball(
			ctx,
			stdin,
			inputRef.Format,
			inputRef.Path,
			inputRef.StripComponents,
		)
	case internal.FormatGit:
		return e.getBucketFromGitRepo(
			ctx,
			inputRef.Path,
			inputRef.GitBranch,
		)
	default:
		return nil, buferrs.NewSystemErrorf("unknown format outside of parse: %v", inputRef.Format)
	}
}

func (e *envReader) getImage(
	ctx context.Context,
	stdin io.Reader,
	inputRef *internal.InputRef,
) (bufpb.Image, error) {
	switch inputRef.Format {
	case internal.FormatBin, internal.FormatBinGz, internal.FormatJSON, internal.FormatJSONGz:
		return e.getImageFromLocalFile(ctx, stdin, inputRef.Format, inputRef.Path)
	default:
		return nil, buferrs.NewSystemErrorf("unknown format outside of parse: %v", inputRef.Format)
	}
}

// Can handle formats FormatDir
func (e *envReader) getBucketFromLocalDir(
	path string,
) (storage.ReadBucket, error) {
	bucket, err := storageos.NewBucket(path)
	if err != nil {
		if storage.IsNotExist(err) || storageos.IsNotDir(err) {
			return nil, buferrs.NewUserError(err.Error())
		}
		return nil, err
	}
	return bucket, nil
}

// Can handle formats FormatTar, FormatTarGz
func (e *envReader) getBucketFromLocalTarball(
	ctx context.Context,
	stdin io.Reader,
	format internal.Format,
	path string,
	stripComponents uint32,
) (_ storage.ReadBucket, retErr error) {
	data, err := e.getFileData(ctx, stdin, path)
	if err != nil {
		return nil, err
	}
	transformerOptions := []storagepath.TransformerOption{
		storagepath.WithExt(".proto"),
		storagepath.WithExactPath(bufconfig.ConfigFilePath),
	}
	if stripComponents > 0 {
		transformerOptions = append(
			transformerOptions,
			storagepath.WithStripComponents(stripComponents),
		)
	}
	bucket := storagemem.NewBucket()
	switch format {
	case internal.FormatTar:
		err = storageutil.Untar(ctx, bytes.NewReader(data), bucket, transformerOptions...)
	case internal.FormatTarGz:
		err = storageutil.Untargz(ctx, bytes.NewReader(data), bucket, transformerOptions...)
	default:
		return nil, buferrs.NewSystemErrorf("got image format %v outside of parse", format)
	}
	if err != nil {
		// TODO: this isn't really an invalid argument
		return nil, multierr.Append(buferrs.NewUserErrorf("untar error: %v", err), bucket.Close())
	}
	return bucket, nil
}

// For FormatGit
func (e *envReader) getBucketFromGitRepo(
	ctx context.Context,
	gitRepo string,
	gitBranch string,
) (_ storage.ReadBucket, retErr error) {
	defer logutil.Defer(e.logger, "get_git_bucket_memory")()

	if !strings.Contains(gitRepo, "://") {
		absGitRepo, err := filepath.Abs(gitRepo)
		if err != nil {
			return nil, err
		}
		gitRepo = "file://" + absGitRepo
	}
	bucket := storagemem.NewBucket()
	if err := storagegit.Clone(
		ctx,
		e.logger,
		gitRepo,
		gitBranch,
		bucket,
		storagepath.WithExt(".proto"),
		storagepath.WithExactPath(bufconfig.ConfigFilePath),
	); err != nil {
		return nil, multierr.Append(
			// TODO: not really an invalid argument
			buferrs.NewUserErrorf("could not clone %s: %v", gitRepo, err),
			bucket.Close(),
		)
	}
	return bucket, nil
}

// Can handle formats FormatBin, FormatBinGz, FormatJSON, FormatJSONGz
func (e *envReader) getImageFromLocalFile(
	ctx context.Context,
	stdin io.Reader,
	format internal.Format,
	path string,
) (_ bufpb.Image, retErr error) {
	data, err := e.getFileData(ctx, stdin, path)
	if err != nil {
		return nil, err
	}
	return e.getImageFromData(format, data)
}

func (e *envReader) getFileData(
	ctx context.Context,
	stdin io.Reader,
	path string,
) ([]byte, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return e.getFileDataFromHTTP(ctx, path)
	}
	return e.getFileDataFromOS(stdin, path)
}

func (e *envReader) getFileDataFromHTTP(
	ctx context.Context,
	path string,
) (_ []byte, retErr error) {
	request, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	response, err := e.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		// TODO: not really an invalid argument
		return nil, buferrs.NewUserErrorf("got HTTP status code %d for %s", response.StatusCode, path)
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		// TODO: not really an invalid argument
		return nil, buferrs.NewUserErrorf("could not read %s: %v", path, err)
	}
	return data, nil
}

func (e *envReader) getFileDataFromOS(
	stdin io.Reader,
	path string,
) (_ []byte, retErr error) {
	if strings.HasPrefix(path, "file://") {
		path = strings.TrimPrefix(path, "file://")
	}
	readCloser, err := clios.ReadCloserForFilePath(stdin, path)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readCloser.Close())
	}()
	data, err := ioutil.ReadAll(readCloser)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, buferrs.NewUserError(err.Error())
		}
		return nil, err
	}
	return data, nil
}

// Can handle formats FormatBin, FormatBinGz, FormatJSON, FormatJSONGz
func (e *envReader) getImageFromData(
	format internal.Format,
	data []byte,
) (_ bufpb.Image, retErr error) {
	if format == internal.FormatBinGz || format == internal.FormatJSONGz {
		// TODO: this has to be woefully inefficient
		// we can prob do a non-copy
		gzipReader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			// TODO: not really an invalid argument
			return nil, buferrs.NewUserErrorf("gzip error: %v", err)
		}
		defer func() {
			retErr = multierr.Append(retErr, gzipReader.Close())
		}()
		uncompressedData, err := ioutil.ReadAll(gzipReader)
		if err != nil {
			// TODO: not really an invalid argument
			return nil, buferrs.NewUserErrorf("gzip error: %v", err)
		}
		data = uncompressedData
	}

	var image bufpb.Image
	var err error
	switch format {
	case internal.FormatBin, internal.FormatBinGz:
		image, err = bufpb.UnmarshalWireDataImage(data)
	case internal.FormatJSON, internal.FormatJSONGz:
		image, err = bufpb.UnmarshalJSONDataImage(data)
	default:
		return nil, buferrs.NewSystemErrorf("got image format %v outside of parse", format)
	}
	if err != nil {
		// TODO: not really an invalid argument
		return nil, buferrs.NewUserErrorf("could not unmarshal Image: %v", err)
	}
	return image, nil
}
