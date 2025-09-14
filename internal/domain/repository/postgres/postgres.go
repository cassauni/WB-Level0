package postgres

import (
	"context"
	"fmt"

	"order-service/config"
	"order-service/internal/domain/entities"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Repository struct {
	pool *pgxpool.Pool
	cfg  *config.ConfigModel
	log  *zap.SugaredLogger
}

func NewRepository(ctx context.Context, cfg *config.ConfigModel, l *zap.Logger) (*Repository, error) {
	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		return nil, err
	}
	r := &Repository{pool: pool, cfg: cfg, log: l.Named("postgres").Sugar()}

	if err := r.runMigrations(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}
	r.log.Infow("migrations ok")
	return r, nil
}

func (r *Repository) runMigrations() error {
	m, err := migrate.New("file://migrations", r.cfg.Postgres.DSN)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func (r *Repository) Close() { r.pool.Close() }

func (r *Repository) Find(ctx context.Context, id string) (*entities.Order, error) {
	var ord entities.Order

	u, err := uuid.Parse(id)
	if err != nil {
		r.log.Warnw("invalid uuid", "order_uid", id, "error", err)
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}
	r.log.Debugw("db find", "order_uid", u)

	const orderSQL = `SELECT order_uid, track_number, entry, locale, internal_signature, customer_id,
	                  delivery_service, shardkey, sm_id, date_created
	                  FROM orders WHERE order_uid=$1`
	err = r.pool.QueryRow(ctx, orderSQL, u).Scan(
		&ord.OrderId, &ord.TrackNumber, &ord.Entry, &ord.Locale, &ord.InternalSignature,
		&ord.CustomerId, &ord.DeliveryService, &ord.ShardKey, &ord.SmId, &ord.DateCreated,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			r.log.Infow("not found", "order_uid", u)
			return nil, nil
		}
		r.log.Errorw("query order failed", "order_uid", u, "error", err)
		return nil, err
	}

	const delSQL = `SELECT del_name, phone, zip, city, address, region, email
	                FROM deliveries WHERE order_uid=$1`
	if err = r.pool.QueryRow(ctx, delSQL, u).Scan(
		&ord.Delivery.Name, &ord.Delivery.Phone, &ord.Delivery.Zip, &ord.Delivery.City,
		&ord.Delivery.Address, &ord.Delivery.Region, &ord.Delivery.Email,
	); err != nil {
		r.log.Errorw("query delivery failed", "order_uid", u, "error", err)
		return nil, err
	}

	const paySQL = `SELECT transaction_id, request_id, currency, provider,
	                      amount::BIGINT, payment_dt, bank, delivery_cost::BIGINT, goods_total::BIGINT, custom_fee::BIGINT
	                 FROM payments WHERE order_uid=$1`
	if err = r.pool.QueryRow(ctx, paySQL, u).Scan(
		&ord.Payment.TransactionId, &ord.Payment.RequestId, &ord.Payment.Currency, &ord.Payment.Provider,
		&ord.Payment.Amount, &ord.Payment.PaymentDt, &ord.Payment.Bank, &ord.Payment.DeliveryCost,
		&ord.Payment.GoodsTotal, &ord.Payment.CustomFee,
	); err != nil {
		r.log.Errorw("query payment failed", "order_uid", u, "error", err)
		return nil, err
	}

	const itemSQL = `SELECT chrt_id, track_number, price::BIGINT, rid, item_name, sale, item_size, total_price::BIGINT, nm_id, brand, status
	                 FROM items WHERE order_uid=$1`
	rows, err := r.pool.Query(ctx, itemSQL, u)
	if err != nil {
		r.log.Errorw("query items failed", "order_uid", u, "error", err)
		return nil, err
	}
	defer rows.Close()

	ord.Items = make([]entities.Item, 0, 4)
	for rows.Next() {
		var it entities.Item
		if err := rows.Scan(&it.ChrtId, &it.TrackNumber, &it.Price, &it.RID, &it.Name, &it.Sale, &it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status); err != nil {
			r.log.Errorw("scan item failed", "order_uid", u, "error", err)
			return nil, err
		}
		ord.Items = append(ord.Items, it)
	}
	r.log.Infow("order loaded", "order_uid", u, "items", len(ord.Items))
	return &ord, nil
}

func (r *Repository) Save(ctx context.Context, o *entities.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	const q1 = `INSERT INTO orders(
	               order_uid, track_number, entry, locale, internal_signature,
	               customer_id, delivery_service, shardkey, sm_id, date_created
	           )
	           VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	           ON CONFLICT (order_uid) DO UPDATE SET
	               track_number       = EXCLUDED.track_number,
	               entry              = EXCLUDED.entry,
	               locale             = EXCLUDED.locale,
	               internal_signature = EXCLUDED.internal_signature,
	               customer_id        = EXCLUDED.customer_id,
	               delivery_service   = EXCLUDED.delivery_service,
	               shardkey           = EXCLUDED.shardkey,
	               sm_id              = EXCLUDED.sm_id`
	if _, err := tx.Exec(ctx, q1,
		o.OrderId, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature,
		o.CustomerId, o.DeliveryService, o.ShardKey, o.SmId, o.DateCreated,
	); err != nil {
		return err
	}

	const q2 = `INSERT INTO deliveries(
	               order_uid, del_name, phone, zip, city, address, region, email
	           )
	           VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	           ON CONFLICT (order_uid) DO UPDATE SET
	               del_name = EXCLUDED.del_name,
	               phone    = EXCLUDED.phone,
	               zip      = EXCLUDED.zip,
	               city     = EXCLUDED.city,
	               address  = EXCLUDED.address,
	               region   = EXCLUDED.region,
	               email    = EXCLUDED.email`
	if _, err := tx.Exec(ctx, q2,
		o.OrderId, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email,
	); err != nil {
		return err
	}

	const q3 = `INSERT INTO payments(
	               order_uid, transaction_id, request_id, currency, provider,
	               amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
	           )
	           VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	           ON CONFLICT (order_uid) DO UPDATE SET
	               transaction_id = EXCLUDED.transaction_id,
	               request_id     = EXCLUDED.request_id,
	               currency       = EXCLUDED.currency,
	               provider       = EXCLUDED.provider,
	               amount         = EXCLUDED.amount,
	               payment_dt     = EXCLUDED.payment_dt,
	               bank           = EXCLUDED.bank,
	               delivery_cost  = EXCLUDED.delivery_cost,
	               goods_total    = EXCLUDED.goods_total,
	               custom_fee     = EXCLUDED.custom_fee`
	if _, err := tx.Exec(ctx, q3,
		o.OrderId, o.Payment.TransactionId, o.Payment.RequestId, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank, o.Payment.DeliveryCost, o.Payment.GoodsTotal, o.Payment.CustomFee,
	); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM items WHERE order_uid=$1`, o.OrderId); err != nil {
		return err
	}
	if len(o.Items) > 0 {
		const q4 = `INSERT INTO items(
		    order_uid, chrt_id, track_number, price, rid, item_name, sale, item_size, total_price, nm_id, brand, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
		b := &pgx.Batch{}
		for _, it := range o.Items {
			b.Queue(q4, o.OrderId, it.ChrtId, it.TrackNumber, it.Price, it.RID, it.Name, it.Sale, it.Size, it.TotalPrice, it.NmID, it.Brand, it.Status)
		}
		if err := tx.SendBatch(ctx, b).Close(); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	r.log.Infow("order saved", "order_uid", o.OrderId)
	return nil
}

func (r *Repository) CacheRestore(ctx context.Context) ([]*entities.Order, error) {
	const q = `SELECT order_uid FROM orders ORDER BY date_created DESC LIMIT 10`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	out := make([]*entities.Order, 0, len(ids))
	for _, id := range ids {
		o, err := r.Find(ctx, id.String())
		if err != nil {
			return nil, err
		}
		if o != nil {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *Repository) RecentIDs(ctx context.Context, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 10
	}
	const q = `SELECT order_uid FROM orders ORDER BY date_created DESC LIMIT $1`
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
