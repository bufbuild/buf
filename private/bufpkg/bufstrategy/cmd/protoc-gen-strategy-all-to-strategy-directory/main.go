package main

import (
	"context"
	"fmt"

	"github.com/bufbuild/buf/private/bufpkg/bufimage"
	"github.com/bufbuild/buf/private/pkg/app"
	"github.com/bufbuild/buf/private/pkg/app/appproto"
	"github.com/bufbuild/buf/private/pkg/app/appproto/appprotoexec"
	"github.com/bufbuild/buf/private/pkg/command"
	"github.com/bufbuild/buf/private/pkg/thread"
	"google.golang.org/protobuf/types/pluginpb"
)

const pluginPathEnvKey = "PLUGIN_PATH"

// Main is the main.
func Main() {
	appproto.Main(
		context.Background(),
		appproto.HandlerFunc(
			func(
				ctx context.Context,
				container app.EnvStderrContainer,
				responseWriter appproto.ResponseBuilder,
				request *pluginpb.CodeGeneratorRequest,
			) error {
				return handle(
					ctx,
					container,
					responseWriter,
					request,
				)
			},
		),
	)
}

func handle(
	ctx context.Context,
	container app.EnvStderrContainer,
	responseWriter appproto.ResponseBuilder,
	request *pluginpb.CodeGeneratorRequest,
) error {
	pluginPath := container.Env(pluginPathEnvKey)
	if pluginPath == "" {
		return fmt.Errorf("must set %s", pluginPathEnvKey)
	}
	image, err := bufimage.NewImageForCodeGeneratorRequest(request)
	if err != nil {
		return err
	}
	imagesByDir, err := bufimage.ImageByDir(image)
	if err != nil {
		return err
	}
	requestsByDir := bufimage.ImagesToCodeGeneratorRequests(
		imagesByDir,
		request.GetParameter(),
		request.GetCompilerVersion(),
		false,
		false,
	)
	if err != nil {
		return err
	}
	handler, err := appprotoexec.NewBinaryHandler(command.NewRunner(), pluginPath)
	if err != nil {
		return err
	}
	jobs := make([]func(context.Context) error, 0, len(requestsByDir))
	for _, requestByDir := range requestsByDir {
		requestByDir := requestByDir
		jobs = append(
			jobs,
			func(ctx context.Context) error {
				return handler.Handle(ctx, container, responseWriter, requestByDir)
			},
		)
	}
	if err := thread.Parallelize(ctx, jobs); err != nil {
		return err
	}
	return nil
}
