package kafka

import (
	"context"
	"encoding/json"
	"time"

	"order-service/config"
	"order-service/internal/domain/entities"
	"order-service/internal/domain/usecase"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Consumer struct {
	cfg *config.ConfigModel
	uc  *usecase.OrderUC
	log *zap.SugaredLogger
}

func NewConsumer(cfg *config.ConfigModel, uc *usecase.OrderUC, l *zap.Logger) (*Consumer, error) {
	return &Consumer{cfg: cfg, uc: uc, log: l.Named("kafka.consumer").Sugar()}, nil
}

func (c *Consumer) OnStart() error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.cfg.Kafka.Brokers,
		GroupID:        c.cfg.Kafka.GroupID,
		Topic:          c.cfg.Kafka.Topic,
		StartOffset:    kafka.LastOffset,
		CommitInterval: 0,
		MinBytes:       1_000,
		MaxBytes:       1_000_000,
		MaxWait:        500 * time.Millisecond,
	})

	go func() {
		defer r.Close()
		c.log.Infow("listening",
			"brokers", c.cfg.Kafka.Brokers,
			"topic", c.cfg.Kafka.Topic,
			"group", c.cfg.Kafka.GroupID,
		)
		ctx := context.Background()

		for {
			msg, err := r.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				c.log.Errorw("read error", "error", err)
				return
			}
			var ord entities.Order
			if err := json.Unmarshal(msg.Value, &ord); err != nil {
				c.log.Warnw("bad json, skip", "partition", msg.Partition, "offset", msg.Offset, "error", err)
				_ = r.CommitMessages(ctx, msg)
				continue
			}

			c.log.Infow("received",
				"order_uid", ord.OrderId.String(),
				"key", string(msg.Key),
				"partition", msg.Partition,
				"offset", msg.Offset,
			)

			if err := c.uc.Set(ctx, &ord); err != nil {
				c.log.Errorw("save failed", "order_uid", ord.OrderId.String(), "error", err)
				continue
			}
			if err := r.CommitMessages(ctx, msg); err != nil {
				c.log.Errorw("commit failed", "partition", msg.Partition, "offset", msg.Offset, "error", err)
			}
		}
	}()
	return nil
}
