package app

import (
	"context"
	"order-service/config"
	"order-service/internal/domain/delivery/http"
	"order-service/internal/domain/delivery/kafka"
	"order-service/internal/domain/repository"
	"order-service/internal/domain/usecase"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func New() *fx.App {
	return fx.New(
		fx.Provide(
			context.Background,
			config.NewConfig,
			zap.NewDevelopment,
		),
		repository.Module(), 
		usecase.Module(),
		http.Module(),
		kafka.Module(), 
		fx.WithLogger(func(l *zap.Logger) fxevent.Logger { return &fxevent.ZapLogger{Logger: l} }),
	)
}
