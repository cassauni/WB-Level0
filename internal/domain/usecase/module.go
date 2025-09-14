package usecase

import "go.uber.org/fx"

import (
	cachepkg "order-service/internal/domain/repository/cache"
	pgrepo "order-service/internal/domain/repository/postgres"
)

func asRepo(r *pgrepo.Repository) repo     { return r }
func asCache(c *cachepkg.RedisCache) cache { return c }

func Module() fx.Option {
	return fx.Module("usecase",
		fx.Provide(
			asRepo,
			asCache,
			NewOrderUC,
		),
	)
}
