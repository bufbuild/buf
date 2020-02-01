package bufos

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bufbuild/buf/internal/buf/bufbuild"
	"github.com/bufbuild/buf/internal/buf/bufconfig"
	"github.com/bufbuild/buf/internal/buf/bufos/internal"
	"github.com/bufbuild/buf/internal/buf/ext/extimage"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagegit"
	"github.com/bufbuild/buf/internal/pkg/storage/storagegit/storagegitplumbing"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"github.com/bufbuild/buf/internal/pkg/util/utillog"
	"github.com/bufbuild/cli/clios"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

var jsonUnmarshaler = &jsonpb.Unmarshaler{
	AllowUnknownFields: true,
}

type envReader struct {
	logger                   *zap.Logger
	httpClient               *http.Client
	configProvider           bufconfig.Provider
	buildHandler             bufbuild.Handler
	inputRefParser           internal.InputRefParser
	configOverrideParser     internal.ConfigOverrideParser
	httpsUsernameEnvKey      string
	httpsPasswordEnvKey      string
	sshKeyFileEnvKey         string
	sshKeyPassphraseEnvKey   string
	sshKnownHostsFilesEnvKey string
}

func newEnvReader(
	logger *zap.Logger,
	httpClient *http.Client,
	configProvider bufconfig.Provider,
	buildHandler bufbuild.Handler,
	valueFlagName string,
	configOverrideFlagName string,
	httpsUsernameEnvKey string,
	httpsPasswordEnvKey string,
	sshKeyFileEnvKey string,
	sshKeyPassphraseEnvKey string,
	sshKnownHostsFilesEnvKey string,
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
		httpsUsernameEnvKey:      httpsUsernameEnvKey,
		httpsPasswordEnvKey:      httpsPasswordEnvKey,
		sshKeyFileEnvKey:         sshKeyFileEnvKey,
		sshKeyPassphraseEnvKey:   sshKeyPassphraseEnvKey,
		sshKnownHostsFilesEnvKey: sshKnownHostsFilesEnvKey,
	}
}

func (e *envReader) ReadEnv(
	ctx context.Context,
	stdin io.Reader,
	getenv func(string) string,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
) (*Env, []*filev1beta1.FileAnnotation, error) {
	return e.readEnv(
		ctx,
		stdin,
		getenv,
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
	getenv func(string) string,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
) (*Env, []*filev1beta1.FileAnnotation, error) {
	return e.readEnv(
		ctx,
		stdin,
		getenv,
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
	getenv func(string) string,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
) (*Env, error) {
	env, fileAnnotations, err := e.readEnv(
		ctx,
		stdin,
		getenv,
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
	if len(fileAnnotations) > 0 {
		// TODO: need to refactor this
		return nil, errors.New("got fileAnnotations for ReadImageEnv which should be impossible")
	}
	return env, nil
}

func (e *envReader) ListFiles(
	ctx context.Context,
	stdin io.Reader,
	getenv func(string) string,
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
		image, err := e.getImage(ctx, stdin, getenv, inputRef)
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
	bucket, err := e.getBucket(ctx, stdin, getenv, inputRef)
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

	protoFileSet, err := e.buildHandler.Files(
		ctx,
		bucket,
		bufbuild.FilesOptions{
			Roots:    config.Build.Roots,
			Excludes: config.Build.Excludes,
		},
	)
	if err != nil {
		return nil, err
	}
	filePaths := protoFileSet.RealFilePaths()
	//// The files are in the order of the root file paths, we want to sort them for output.
	sort.Strings(filePaths)
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
		resolvedFilePath, err := resolver.GetRealFilePath(filePath)
		if err != nil {
			// This is an internal error if we cannot resolve this file path.
			return nil, err
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
	getenv func(string) string,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
	onlySources bool,
	onlyImages bool,
) (_ *Env, _ []*filev1beta1.FileAnnotation, retErr error) {
	inputRef, err := e.inputRefParser.ParseInputRef(value, onlySources, onlyImages)
	if err != nil {
		return nil, nil, err
	}
	e.logger.Debug("parse", zap.Any("input_ref", inputRef), zap.Stringer("format", inputRef.Format))

	if inputRef.Format.IsImage() {
		env, err := e.readEnvFromImage(
			ctx,
			stdin,
			getenv,
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
		getenv,
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
	getenv func(string) string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
	inputRef *internal.InputRef,
) (_ *Env, _ []*filev1beta1.FileAnnotation, retErr error) {
	bucket, err := e.getBucket(ctx, stdin, getenv, inputRef)
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
	protoFileSet, err := e.buildHandler.Files(
		ctx,
		bucket,
		bufbuild.FilesOptions{
			Roots:                              config.Build.Roots,
			Excludes:                           config.Build.Excludes,
			SpecificRealFilePaths:              specificRealFilePaths,
			SpecificRealFilePathsAllowNotExist: specificFilePathsAllowNotExist,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	var resolver bufbuild.ProtoRealFilePathResolver = protoFileSet
	if inputRef.Format == internal.FormatDir {
		resolver, err = internal.NewRelProtoFilePathResolver(inputRef.Path, resolver)
		if err != nil {
			return nil, nil, err
		}
	}
	image, fileAnnotations, err := e.buildHandler.Build(
		ctx,
		bucket,
		protoFileSet,
		bufbuild.BuildOptions{
			IncludeImports:    includeImports,
			IncludeSourceInfo: includeSourceInfo,
			// If we specified specific file paths, do not copy to memory
			CopyToMemory: len(specificRealFilePaths) == 0,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	if len(fileAnnotations) > 0 {
		// the documentation for EnvReader says we will resolve before returning
		if err := bufbuild.FixFileAnnotationPaths(resolver, fileAnnotations); err != nil {
			return nil, nil, err
		}
		return nil, fileAnnotations, nil
	}
	return &Env{Image: image, Resolver: resolver, Config: config}, nil, nil
}

func (e *envReader) readEnvFromImage(
	ctx context.Context,
	stdin io.Reader,
	getenv func(string) string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	inputRef *internal.InputRef,
) (_ *Env, retErr error) {
	image, err := e.getImage(ctx, stdin, getenv, inputRef)
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
		image, err = extimage.ImageWithSpecificNames(image, specificFilePathsAllowNotExist, specificFilePaths...)
		if err != nil {
			return nil, err
		}
	}
	if !includeImports {
		image, err = extimage.ImageWithoutImports(image)
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
	getenv func(string) string,
	inputRef *internal.InputRef,
) (storage.ReadBucket, error) {
	switch inputRef.Format {
	case internal.FormatDir:
		return e.getBucketFromLocalDir(inputRef.Path)
	case internal.FormatTar, internal.FormatTarGz:
		return e.getBucketFromLocalTarball(
			ctx,
			stdin,
			getenv,
			inputRef.Format,
			inputRef.Path,
			inputRef.StripComponents,
		)
	case internal.FormatGit:
		return e.getBucketFromGitRepo(
			ctx,
			getenv,
			inputRef.Path,
			inputRef.GitRefName,
		)
	default:
		return nil, fmt.Errorf("unknown format outside of parse: %v", inputRef.Format)
	}
}

func (e *envReader) getImage(
	ctx context.Context,
	stdin io.Reader,
	getenv func(string) string,
	inputRef *internal.InputRef,
) (*imagev1beta1.Image, error) {
	switch inputRef.Format {
	case internal.FormatBin, internal.FormatBinGz, internal.FormatJSON, internal.FormatJSONGz:
		return e.getImageFromLocalFile(ctx, stdin, getenv, inputRef.Format, inputRef.Path)
	default:
		return nil, fmt.Errorf("unknown format outside of parse: %v", inputRef.Format)
	}
}

// Can handle formats FormatDir
func (e *envReader) getBucketFromLocalDir(
	path string,
) (storage.ReadBucket, error) {
	bucket, err := storageos.NewBucket(path)
	if err != nil {
		if storage.IsNotExist(err) || storageos.IsNotDir(err) {
			return nil, err
		}
		return nil, err
	}
	return bucket, nil
}

// Can handle formats FormatTar, FormatTarGz
func (e *envReader) getBucketFromLocalTarball(
	ctx context.Context,
	stdin io.Reader,
	getenv func(string) string,
	format internal.Format,
	path string,
	stripComponents uint32,
) (_ storage.ReadBucket, retErr error) {
	data, err := e.getFileData(ctx, stdin, getenv, path)
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
		return nil, fmt.Errorf("got image format %v outside of parse", format)
	}
	if err != nil {
		// TODO: this isn't really an invalid argument
		return nil, multierr.Append(fmt.Errorf("untar error: %v", err), bucket.Close())
	}
	return bucket, nil
}

// For FormatGit
func (e *envReader) getBucketFromGitRepo(
	ctx context.Context,
	getenv func(string) string,
	gitRepo string,
	gitRefName storagegitplumbing.RefName,
) (_ storage.ReadBucket, retErr error) {
	defer utillog.Defer(e.logger, "get_git_bucket_memory")()

	homeDirPath, err := clios.Home(getenv)
	if err != nil {
		return nil, err
	}
	bucket := storagemem.NewBucket()
	if err := storagegit.Clone(
		ctx,
		e.logger,
		getenv,
		homeDirPath,
		gitRepo,
		gitRefName,
		e.httpsUsernameEnvKey,
		e.httpsPasswordEnvKey,
		e.sshKeyFileEnvKey,
		e.sshKeyPassphraseEnvKey,
		e.sshKnownHostsFilesEnvKey,
		bucket,
		storagepath.WithExt(".proto"),
		storagepath.WithExactPath(bufconfig.ConfigFilePath),
	); err != nil {
		return nil, multierr.Append(
			fmt.Errorf("could not clone %s: %v", gitRepo, err),
			bucket.Close(),
		)
	}
	return bucket, nil
}

// Can handle formats FormatBin, FormatBinGz, FormatJSON, FormatJSONGz
func (e *envReader) getImageFromLocalFile(
	ctx context.Context,
	stdin io.Reader,
	getenv func(string) string,
	format internal.Format,
	path string,
) (_ *imagev1beta1.Image, retErr error) {
	data, err := e.getFileData(ctx, stdin, getenv, path)
	if err != nil {
		return nil, err
	}
	return e.getImageFromData(format, data)
}

func (e *envReader) getFileData(
	ctx context.Context,
	stdin io.Reader,
	getenv func(string) string,
	path string,
) ([]byte, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return e.getFileDataFromHTTP(ctx, getenv, path)
	}
	return e.getFileDataFromOS(stdin, path)
}

func (e *envReader) getFileDataFromHTTP(
	ctx context.Context,
	getenv func(string) string,
	path string,
) (_ []byte, retErr error) {
	request, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	if getenv != nil && strings.HasPrefix(path, "https://") && e.httpsUsernameEnvKey != "" && e.httpsPasswordEnvKey != "" {
		httpsUsername := getenv(e.httpsUsernameEnvKey)
		httpsPassword := getenv(e.httpsPasswordEnvKey)
		if httpsUsername != "" && httpsPassword != "" {
			request.SetBasicAuth(httpsUsername, httpsPassword)
		}
	}
	response, err := e.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, response.Body.Close())
	}()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got HTTP status code %d for %s", response.StatusCode, path)
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %v", path, err)
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
			return nil, err
		}
		return nil, err
	}
	return data, nil
}

// Can handle formats FormatBin, FormatBinGz, FormatJSON, FormatJSONGz
func (e *envReader) getImageFromData(
	format internal.Format,
	data []byte,
) (_ *imagev1beta1.Image, retErr error) {
	if format == internal.FormatBinGz || format == internal.FormatJSONGz {
		// TODO: this has to be woefully inefficient
		// we can prob do a non-copy
		gzipReader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("gzip error: %v", err)
		}
		defer func() {
			retErr = multierr.Append(retErr, gzipReader.Close())
		}()
		uncompressedData, err := ioutil.ReadAll(gzipReader)
		if err != nil {
			return nil, fmt.Errorf("gzip error: %v", err)
		}
		data = uncompressedData
	}

	image := &imagev1beta1.Image{}
	var err error
	switch format {
	case internal.FormatBin, internal.FormatBinGz:
		err = proto.Unmarshal(data, image)
	case internal.FormatJSON, internal.FormatJSONGz:
		err = unmarshalJSON(data, image)
	default:
		return nil, fmt.Errorf("got image format %v outside of parse", format)
	}
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal Image: %v", err)
	}
	if err := extimage.ValidateImage(image); err != nil {
		return nil, err
	}
	return image, nil
}

func unmarshalJSON(data []byte, message proto.Message) error {
	return jsonUnmarshaler.Unmarshal(bytes.NewReader(data), message)
}
