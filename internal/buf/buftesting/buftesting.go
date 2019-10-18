// Package buftesting provides helpers for testing.
package buftesting

import (
	"context"
	"net/http"
	"time"

	"github.com/bufbuild/buf/internal/buf/bufpb"
	"github.com/bufbuild/buf/internal/pkg/diff"
)

var testHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}

// GetProtocImage gets the Image for the realFilePaths using
// protoc and wkt's on the current PATH.
func GetProtocImage(
	ctx context.Context,
	roots []string,
	realFilePaths []string,
	includeImports bool,
	includeSourceInfo bool,
) (bufpb.Image, error) {
	protocLocation, err := getProtocLocation()
	if err != nil {
		return nil, err
	}
	return getProtocImage(
		ctx,
		protocLocation,
		roots,
		realFilePaths,
		includeImports,
		includeSourceInfo,
	)
}

// GetGithubArchive gets the GitHub archive and untars it to the output directory path.
//
// The root directory within the tarball is stripped.
// If the directory already exists, this is a no-op.
func GetGithubArchive(
	ctx context.Context,
	outputDirPath string,
	owner string,
	repository string,
	ref string,
) error {
	return getGithubArchive(
		ctx,
		testHTTPClient,
		outputDirPath,
		owner,
		repository,
		ref,
	)
}

// ImagesEqual checks if the images are equal.
func ImagesEqual(one bufpb.Image, two bufpb.Image) (bool, error) {
	nativeOne, err := one.ToFileDescriptorSet()
	if err != nil {
		return false, err
	}
	nativeTwo, err := two.ToFileDescriptorSet()
	if err != nil {
		return false, err
	}
	return nativeOne.Equal(nativeTwo), nil
}

// DiffImagesJSON diffs the two Images using jsonpb.
func DiffImagesJSON(one bufpb.Image, two bufpb.Image, name string) (string, error) {
	nativeOne, err := one.ToFileDescriptorSet()
	if err != nil {
		return "", err
	}
	nativeTwo, err := two.ToFileDescriptorSet()
	if err != nil {
		return "", err
	}
	oneData, err := nativeOne.MarshalJSONIndent()
	if err != nil {
		return "", err
	}
	twoData, err := nativeTwo.MarshalJSONIndent()
	if err != nil {
		return "", err
	}
	output, err := diff.Do(oneData, twoData, name)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// DiffImagesText diffs the two Images using proto.MarshalText.
func DiffImagesText(one bufpb.Image, two bufpb.Image, name string) (string, error) {
	nativeOne, err := one.ToFileDescriptorSet()
	if err != nil {
		return "", err
	}
	nativeTwo, err := two.ToFileDescriptorSet()
	if err != nil {
		return "", err
	}
	oneData, err := nativeOne.MarshalText()
	if err != nil {
		return "", err
	}
	twoData, err := nativeTwo.MarshalText()
	if err != nil {
		return "", err
	}
	output, err := diff.Do(oneData, twoData, name)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
