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
	"github.com/bufbuild/buf/internal/buf/ext/extimage"
	"github.com/bufbuild/buf/internal/buf/ext/extio"
	filev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/file/v1beta1"
	imagev1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/image/v1beta1"
	iov1beta1 "github.com/bufbuild/buf/internal/gen/proto/go/v1/bufbuild/buf/io/v1beta1"
	"github.com/bufbuild/buf/internal/pkg/cli/clienv"
	"github.com/bufbuild/buf/internal/pkg/cli/clios"
	"github.com/bufbuild/buf/internal/pkg/storage"
	"github.com/bufbuild/buf/internal/pkg/storage/storagegit"
	"github.com/bufbuild/buf/internal/pkg/storage/storagegit/storagegitplumbing"
	"github.com/bufbuild/buf/internal/pkg/storage/storagemem"
	"github.com/bufbuild/buf/internal/pkg/storage/storageos"
	"github.com/bufbuild/buf/internal/pkg/storage/storagepath"
	"github.com/bufbuild/buf/internal/pkg/storage/storageutil"
	"github.com/bufbuild/buf/internal/pkg/util/utilproto"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type envReader struct {
	logger                   *zap.Logger
	httpClient               *http.Client
	configProvider           bufconfig.Provider
	buildHandler             bufbuild.Handler
	valueFlagName            string
	configOverrideFlagName   string
	httpsUsernameEnvKey      string
	httpsPasswordEnvKey      string
	sshKeyFileEnvKey         string
	sshKeyPassphraseEnvKey   string
	sshKnownHostsFilesEnvKey string
	experimentalGitClone     bool
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
	experimentalGitClone bool,
) *envReader {
	return &envReader{
		logger:                   logger.Named("bufos"),
		httpClient:               httpClient,
		configProvider:           configProvider,
		buildHandler:             buildHandler,
		valueFlagName:            valueFlagName,
		configOverrideFlagName:   configOverrideFlagName,
		httpsUsernameEnvKey:      httpsUsernameEnvKey,
		httpsPasswordEnvKey:      httpsPasswordEnvKey,
		sshKeyFileEnvKey:         sshKeyFileEnvKey,
		sshKeyPassphraseEnvKey:   sshKeyPassphraseEnvKey,
		sshKnownHostsFilesEnvKey: sshKnownHostsFilesEnvKey,
		experimentalGitClone:     experimentalGitClone,
	}
}

func (e *envReader) ReadEnv(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
) (*Env, []*filev1beta1.FileAnnotation, error) {
	inputRef, err := e.parseInputRef(value)
	if err != nil {
		return nil, nil, err
	}
	if imageRef := inputRef.GetImageRef(); imageRef != nil {
		env, err := e.readEnvFromImage(
			ctx,
			stdin,
			environ,
			configOverride,
			specificFilePaths,
			specificFilePathsAllowNotExist,
			includeImports,
			imageRef,
		)
		return env, nil, err
	}
	if sourceRef := inputRef.GetSourceRef(); sourceRef != nil {
		return e.readEnvFromSource(
			ctx,
			stdin,
			environ,
			configOverride,
			specificFilePaths,
			specificFilePathsAllowNotExist,
			includeImports,
			includeSourceInfo,
			sourceRef,
		)
	}
	return nil, nil, errors.New("invalid InputRef")
}

func (e *envReader) ReadSourceEnv(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
) (*Env, []*filev1beta1.FileAnnotation, error) {
	sourceRef, err := e.parseSourceRef(value)
	if err != nil {
		return nil, nil, err
	}
	return e.readEnvFromSource(
		ctx,
		stdin,
		environ,
		configOverride,
		specificFilePaths,
		specificFilePathsAllowNotExist,
		includeImports,
		includeSourceInfo,
		sourceRef,
	)
}

func (e *envReader) ReadImageEnv(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	value string,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
) (*Env, error) {
	imageRef, err := e.parseImageRef(value)
	if err != nil {
		return nil, err
	}
	return e.readEnvFromImage(
		ctx,
		stdin,
		environ,
		configOverride,
		specificFilePaths,
		specificFilePathsAllowNotExist,
		includeImports,
		imageRef,
	)
}

func (e *envReader) ListFiles(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	value string,
	configOverride string,
) (_ []string, retErr error) {
	inputRef, err := e.parseInputRef(value)
	if err != nil {
		return nil, err
	}

	if imageRef := inputRef.GetImageRef(); imageRef != nil {
		// if we have an image, list the files in the image
		image, err := e.getImage(ctx, stdin, environ, imageRef)
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
	sourceRef := inputRef.GetSourceRef()
	if sourceRef == nil {
		return nil, errors.New("invalid InputRef")
	}

	// we have a source, we need to get everything
	readBucketCloser, err := e.getReadBucketCloser(ctx, stdin, environ, sourceRef)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()
	var config *bufconfig.Config
	if configOverride != "" {
		config, err = e.parseConfigOverride(configOverride)
		if err != nil {
			return nil, err
		}
	} else {
		// if there is no config override, we read the config from the bucket
		// if there was no file, this just returns default config
		config, err = e.configProvider.GetConfigForReadBucket(ctx, readBucketCloser)
		if err != nil {
			return nil, err
		}
	}

	protoFileSet, err := e.buildHandler.GetProtoFileSet(
		ctx,
		readBucketCloser,
		bufbuild.GetProtoFileSetOptions{
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
	bucketDirPath := getBucketDirPath(sourceRef)
	if bucketDirPath == "" {
		// if format is not a directory, just output the file paths
		return filePaths, nil
	}

	// if we built a directory, we need to resolve file paths
	resolver, err := newRelRealProtoFilePathResolver(bucketDirPath, nil)
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
		return e.parseConfigOverride(configOverride)
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

func (e *envReader) readEnvFromSource(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	includeSourceInfo bool,
	sourceRef *iov1beta1.SourceRef,
) (_ *Env, _ []*filev1beta1.FileAnnotation, retErr error) {
	readBucketCloser, err := e.getReadBucketCloser(ctx, stdin, environ, sourceRef)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, readBucketCloser.Close())
	}()

	var config *bufconfig.Config
	if configOverride != "" {
		config, err = e.parseConfigOverride(configOverride)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// if there is no config override, we read the config from the bucket
		// if there was no file, this just returns default config
		config, err = e.configProvider.GetConfigForReadBucket(ctx, readBucketCloser)
		if err != nil {
			return nil, nil, err
		}
	}
	bucketDirPath := getBucketDirPath(sourceRef)
	var specificRealFilePaths []string
	if len(specificFilePaths) > 0 {
		// since we are doing a build, we filter before doing the build
		// via bufbuild.Provider
		// this will include imports if necessary
		specificRealFilePaths = make([]string, len(specificFilePaths))
		if bucketDirPath != "" {
			// if we had a directory input, then we need to make everything relative to that directory
			absDirPath, err := filepath.Abs(bucketDirPath)
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
	var protoFileSet bufbuild.ProtoFileSet
	if len(specificRealFilePaths) > 0 {
		protoFileSet, err = e.buildHandler.GetProtoFileSetForFiles(
			ctx,
			readBucketCloser,
			specificRealFilePaths,
			bufbuild.GetProtoFileSetForFilesOptions{
				Roots:         config.Build.Roots,
				AllowNotExist: specificFilePathsAllowNotExist,
			},
		)
		if err != nil {
			return nil, nil, err
		}
	} else {
		protoFileSet, err = e.buildHandler.GetProtoFileSet(
			ctx,
			readBucketCloser,
			bufbuild.GetProtoFileSetOptions{
				Roots:    config.Build.Roots,
				Excludes: config.Build.Excludes,
			},
		)
		if err != nil {
			return nil, nil, err
		}
	}
	var resolver bufbuild.ProtoRealFilePathResolver = protoFileSet
	if bucketDirPath != "" {
		resolver, err = newRelRealProtoFilePathResolver(bucketDirPath, resolver)
		if err != nil {
			return nil, nil, err
		}
	}
	image, fileAnnotations, err := e.buildHandler.Build(
		ctx,
		readBucketCloser,
		protoFileSet,
		bufbuild.BuildOptions{
			IncludeImports:    includeImports,
			IncludeSourceInfo: includeSourceInfo,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	if len(fileAnnotations) > 0 {
		// the documentation for EnvReader says we will resolve before returning
		if err := bufbuild.FixFileAnnotationPaths(resolver, fileAnnotations...); err != nil {
			return nil, nil, err
		}
		return nil, fileAnnotations, nil
	}
	return &Env{Image: image, Resolver: resolver, Config: config}, nil, nil
}

func (e *envReader) readEnvFromImage(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	configOverride string,
	specificFilePaths []string,
	specificFilePathsAllowNotExist bool,
	includeImports bool,
	imageRef *iov1beta1.ImageRef,
) (_ *Env, retErr error) {
	image, err := e.getImage(ctx, stdin, environ, imageRef)
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

func (e *envReader) getReadBucketCloser(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	sourceRef *iov1beta1.SourceRef,
) (storage.ReadBucketCloser, error) {
	if archiveRef := sourceRef.GetArchiveRef(); archiveRef != nil {
		return e.getReadBucketCloserFromArchive(
			ctx,
			stdin,
			environ,
			archiveRef,
		)
	}
	if gitRef := sourceRef.GetGitRef(); gitRef != nil {
		return e.getReadBucketCloserFromGit(
			ctx,
			stdin,
			environ,
			gitRef,
		)
	}
	if bucketRef := sourceRef.GetBucketRef(); bucketRef != nil {
		return e.getReadBucketCloserFromBucket(
			ctx,
			stdin,
			environ,
			bucketRef,
		)
	}
	return nil, errors.New("invalid SourceRef")
}

func (e *envReader) getReadBucketCloserFromBucket(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	bucketRef *iov1beta1.BucketRef,
) (storage.ReadBucketCloser, error) {
	switch bucketRef.BucketScheme {
	case iov1beta1.BucketScheme_BUCKET_SCHEME_DIR:
		readBucketCloser, err := storageos.NewReadWriteBucketCloser(bucketRef.Path)
		if err != nil {
			if storage.IsNotExist(err) || storageos.IsNotDir(err) {
				return nil, err
			}
			return nil, err
		}
		return readBucketCloser, nil
	default:
		return nil, fmt.Errorf("unknown BucketScheme: %v", bucketRef.BucketScheme)
	}
}

func (e *envReader) getReadBucketCloserFromArchive(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	archiveRef *iov1beta1.ArchiveRef,
) (_ storage.ReadBucketCloser, retErr error) {
	data, err := e.getFileData(ctx, stdin, environ, archiveRef.FileScheme, archiveRef.Path)
	if err != nil {
		return nil, err
	}
	transformerOptions := []storagepath.TransformerOption{
		storagepath.WithExt(".proto"),
		storagepath.WithExactPath(bufconfig.ConfigFilePath),
	}
	if archiveRef.StripComponents > 0 {
		transformerOptions = append(
			transformerOptions,
			storagepath.WithStripComponents(archiveRef.StripComponents),
		)
	}
	readWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	switch archiveRef.ArchiveFormat {
	case iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TAR:
		err = storageutil.Untar(ctx, bytes.NewReader(data), readWriteBucketCloser, transformerOptions...)
	case iov1beta1.ArchiveFormat_ARCHIVE_FORMAT_TARGZ:
		err = storageutil.Untargz(ctx, bytes.NewReader(data), readWriteBucketCloser, transformerOptions...)
	default:
		return nil, fmt.Errorf("unknown ArchiveFormat: %v", archiveRef.ArchiveFormat)
	}
	if err != nil {
		return nil, multierr.Append(fmt.Errorf("untar error: %v", err), readWriteBucketCloser.Close())
	}
	return readWriteBucketCloser, nil
}

func (e *envReader) getReadBucketCloserFromGit(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	gitRef *iov1beta1.GitRef,
) (_ storage.ReadBucketCloser, retErr error) {
	homeDirPath, err := clios.Home(environ.Getenv)
	if err != nil {
		return nil, err
	}
	gitURL, err := getGitURL(gitRef)
	if err != nil {
		return nil, err
	}
	gitRecurseSubmodules, err := getGitRecurseSubmodules(gitRef)
	if err != nil {
		return nil, err
	}
	readWriteBucketCloser := storagemem.NewReadWriteBucketCloser()
	transformerOptions := []storagepath.TransformerOption{
		storagepath.WithExt(".proto"),
		storagepath.WithExactPath(bufconfig.ConfigFilePath),
	}
	if e.experimentalGitClone {
		err = storagegit.ExperimentalClone(
			ctx,
			e.logger,
			environ.Environ(),
			gitURL,
			gitRef.GetBranch(),
			gitRef.GetTag(),
			gitRecurseSubmodules,
			readWriteBucketCloser,
			transformerOptions...,
		)
	} else {
		var gitRefName storagegitplumbing.RefName
		gitRefName, err = getGitRefName(gitRef)
		if err != nil {
			return nil, err
		}
		err = storagegit.Clone(
			ctx,
			e.logger,
			environ.Getenv,
			homeDirPath,
			gitURL,
			gitRefName,
			gitRecurseSubmodules,
			e.httpsUsernameEnvKey,
			e.httpsPasswordEnvKey,
			e.sshKeyFileEnvKey,
			e.sshKeyPassphraseEnvKey,
			e.sshKnownHostsFilesEnvKey,
			readWriteBucketCloser,
			transformerOptions...,
		)
	}
	if err != nil {
		return nil, multierr.Append(
			fmt.Errorf("could not clone %s: %v", gitURL, err),
			readWriteBucketCloser.Close(),
		)
	}
	return readWriteBucketCloser, nil
}

func (e *envReader) getImage(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	imageRef *iov1beta1.ImageRef,
) (_ *imagev1beta1.Image, retErr error) {
	data, err := e.getFileData(ctx, stdin, environ, imageRef.FileScheme, imageRef.Path)
	if err != nil {
		return nil, err
	}
	return e.getImageFromData(imageRef.ImageFormat, data)
}

func (e *envReader) getFileData(
	ctx context.Context,
	stdin io.Reader,
	environ clienv.Environ,
	fileScheme iov1beta1.FileScheme,
	path string,
) ([]byte, error) {
	switch fileScheme {
	case iov1beta1.FileScheme_FILE_SCHEME_HTTP:
		return e.getFileDataFromHTTP(ctx, environ, "http://"+path)
	case iov1beta1.FileScheme_FILE_SCHEME_HTTPS:
		return e.getFileDataFromHTTP(ctx, environ, "https://"+path)
	case iov1beta1.FileScheme_FILE_SCHEME_STDIO:
		return ioutil.ReadAll(stdin)
	case iov1beta1.FileScheme_FILE_SCHEME_NULL:
		return nil, errors.New("cannot read file data from /dev/null equivalent")
	case iov1beta1.FileScheme_FILE_SCHEME_FILE:
		return ioutil.ReadFile(path)
	default:
		return nil, fmt.Errorf("uknown FileScheme: %v", fileScheme)
	}
}

func (e *envReader) getFileDataFromHTTP(
	ctx context.Context,
	environ clienv.Environ,
	path string,
) (_ []byte, retErr error) {
	request, err := http.NewRequestWithContext(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	if environ != nil && strings.HasPrefix(path, "https://") && e.httpsUsernameEnvKey != "" && e.httpsPasswordEnvKey != "" {
		httpsUsername := environ.Getenv(e.httpsUsernameEnvKey)
		httpsPassword := environ.Getenv(e.httpsPasswordEnvKey)
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

func (e *envReader) getImageFromData(
	imageFormat iov1beta1.ImageFormat,
	data []byte,
) (_ *imagev1beta1.Image, retErr error) {
	if imageFormat == iov1beta1.ImageFormat_IMAGE_FORMAT_BINGZ || imageFormat == iov1beta1.ImageFormat_IMAGE_FORMAT_JSONGZ {
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
	switch imageFormat {
	case iov1beta1.ImageFormat_IMAGE_FORMAT_BIN, iov1beta1.ImageFormat_IMAGE_FORMAT_BINGZ:
		err = utilproto.UnmarshalWire(data, image)
	case iov1beta1.ImageFormat_IMAGE_FORMAT_JSON, iov1beta1.ImageFormat_IMAGE_FORMAT_JSONGZ:
		err = utilproto.UnmarshalJSON(data, image)
	default:
		return nil, fmt.Errorf("unknown ImageFormat: %v", imageFormat)
	}
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal Image: %v", err)
	}
	if err := extimage.ValidateImage(image); err != nil {
		return nil, err
	}
	return image, nil
}

func (e *envReader) parseInputRef(value string) (*iov1beta1.InputRef, error) {
	inputRef, err := extio.ParseInputRef(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", e.valueFlagName, err)
	}
	e.logger.Debug("read", zap.Any("input_ref", inputRef))
	return inputRef, nil
}

func (e *envReader) parseImageRef(value string) (*iov1beta1.ImageRef, error) {
	imageRef, err := extio.ParseImageRef(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", e.valueFlagName, err)
	}
	e.logger.Debug("read", zap.Any("image_ref", imageRef))
	return imageRef, nil
}

func (e *envReader) parseSourceRef(value string) (*iov1beta1.SourceRef, error) {
	sourceRef, err := extio.ParseSourceRef(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", e.valueFlagName, err)
	}
	e.logger.Debug("read", zap.Any("source_ref", sourceRef))
	return sourceRef, nil
}

func (e *envReader) parseConfigOverride(value string) (*bufconfig.Config, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("config override value is empty")
	}
	var data []byte
	var err error
	switch filepath.Ext(value) {
	case ".json", ".yaml":
		data, err = ioutil.ReadFile(value)
		if err != nil {
			return nil, fmt.Errorf("%s: could not read file: %v", e.configOverrideFlagName, err)
		}
	default:
		data = []byte(value)
	}
	config, err := e.configProvider.GetConfigForData(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", e.configOverrideFlagName, err)
	}
	return config, nil
}

func getBucketDirPath(sourceRef *iov1beta1.SourceRef) string {
	bucketRef := sourceRef.GetBucketRef()
	if bucketRef == nil {
		return ""
	}
	if bucketRef.BucketScheme != iov1beta1.BucketScheme_BUCKET_SCHEME_DIR {
		return ""
	}
	return bucketRef.Path
}

func getGitURL(gitRef *iov1beta1.GitRef) (string, error) {
	switch gitRef.GitScheme {
	case iov1beta1.GitScheme_GIT_SCHEME_HTTP:
		return "http://" + gitRef.Path, nil
	case iov1beta1.GitScheme_GIT_SCHEME_HTTPS:
		return "https://" + gitRef.Path, nil
	case iov1beta1.GitScheme_GIT_SCHEME_SSH:
		return "ssh://" + gitRef.Path, nil
	case iov1beta1.GitScheme_GIT_SCHEME_FILE:
		absPath, err := filepath.Abs(gitRef.Path)
		if err != nil {
			return "", err
		}
		return "file://" + absPath, nil
	default:
		return "", fmt.Errorf("unknown GitScheme: %v", gitRef.GitScheme)
	}
}

func getGitRefName(gitRef *iov1beta1.GitRef) (storagegitplumbing.RefName, error) {
	if branch := gitRef.GetBranch(); branch != "" {
		return storagegitplumbing.NewBranchRefName(branch), nil
	}
	if tag := gitRef.GetTag(); tag != "" {
		return storagegitplumbing.NewTagRefName(tag), nil
	}
	return nil, errors.New("invalid GitRef")
}

func getGitRecurseSubmodules(gitRef *iov1beta1.GitRef) (bool, error) {
	switch gitRef.GitSubmoduleBehavior {
	case iov1beta1.GitSubmoduleBehavior_GIT_SUBMODULE_BEHAVIOR_NONE:
		return false, nil
	case iov1beta1.GitSubmoduleBehavior_GIT_SUBMODULE_BEHAVIOR_RECURSIVE:
		return true, nil
	default:
		return false, fmt.Errorf("unknown GitSubmoduleBehavior: %v", gitRef.GitSubmoduleBehavior)
	}
}
