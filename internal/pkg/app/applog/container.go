package applog

import (
	"github.com/bufbuild/buf/internal/pkg/app"
	"go.uber.org/zap"
)

type container struct {
	app.Container

	logger *zap.Logger
}

func newContainer(appContainer app.Container, logger *zap.Logger) *container {
	return &container{
		Container: appContainer,
		logger:    logger,
	}
}

func (c *container) Logger() *zap.Logger {
	return c.logger
}
