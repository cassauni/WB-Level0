package usecase

import (
	"context"
	"order-service/internal/domain/entities"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type repo interface {
	Find(ctx context.Context, id string) (*entities.Order, error)
	Save(ctx context.Context, order *entities.Order) error
	CacheRestore(ctx context.Context) ([]*entities.Order, error)
	RecentIDs(ctx context.Context, limit int) ([]uuid.UUID, error)
}

type cache interface {
	Get(id string) (*entities.Order, bool)
	Set(id string, order *entities.Order)
}

type OrderUC struct {
	repo  repo
	cache cache
	log   *zap.SugaredLogger
}

func NewOrderUC(r repo, c cache, l *zap.Logger) (*OrderUC, error) {
	return &OrderUC{repo: r, cache: c, log: l.Named("usecase").Sugar()}, nil
}

func (uc *OrderUC) Get(ctx context.Context, id string) (*entities.Order, error) {
	uc.log.Infow("get order", "order_uid", id)

	if _, err := uuid.Parse(id); err != nil {
		uc.log.Warnw("invalid uuid", "order_uid", id, "error", err)
		return nil, err
	}

	if v, ok := uc.cache.Get(id); ok {
		uc.log.Debugw("served from cache", "order_uid", id)
		return v, nil
	}

	o, err := uc.repo.Find(ctx, id)
	if err != nil {
		uc.log.Errorw("db find error", "order_uid", id, "error", err)
		return nil, err
	}
	if o == nil {
		uc.log.Infow("not found", "order_uid", id)
		return nil, nil
	}

	uc.cache.Set(id, o)
	uc.log.Debugw("cached after db fetch", "order_uid", id)
	return o, nil
}

func (uc *OrderUC) Set(ctx context.Context, o *entities.Order) error {
	id := o.OrderId.String()
	uc.log.Infow("save order", "order_uid", id)

	if err := uc.repo.Save(ctx, o); err != nil {
		uc.log.Errorw("db save error", "order_uid", id, "error", err)
		return err
	}
	uc.cache.Set(id, o)
	uc.log.Infow("saved", "order_uid", id)
	return nil
}

func (uc *OrderUC) WarmCache(ctx context.Context) {
	list, err := uc.repo.CacheRestore(ctx)
	if err != nil {
		uc.log.Errorw("cache restore failed", "error", err)
		return
	}
	for _, o := range list {
		uc.cache.Set(o.OrderId.String(), o)
	}
	uc.log.Infow("cache warmed", "count", len(list))
}

func (uc *OrderUC) RecentIDs(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 10
	}
	ids, err := uc.repo.RecentIDs(ctx, limit)
	if err != nil {
		uc.log.Errorw("recent ids error", "error", err)
		return nil, err
	}
	out := make([]string, 0, len(ids))
	for _, u := range ids {
		out = append(out, u.String())
	}
	uc.log.Debugw("recent ids", "count", len(out))
	return out, nil
}
