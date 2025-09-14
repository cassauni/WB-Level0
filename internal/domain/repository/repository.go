package repository

import (
	"order-service/internal/domain/repository/cache"
	"order-service/internal/domain/repository/postgres"

	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module("repository",
		fx.Provide(
			cache.NewRedisCache,    
			postgres.NewRepository, 
		),
	)
}
