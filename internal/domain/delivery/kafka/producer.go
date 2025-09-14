package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"order-service/config"
	"order-service/internal/domain/entities"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Producer struct {
	cfg *config.ConfigModel
	log *zap.SugaredLogger
}

func NewProducer(cfg *config.ConfigModel, l *zap.Logger) (*Producer, error) {
	return &Producer{cfg: cfg, log: l.Named("kafka.producer").Sugar()}, nil
}

func (p *Producer) OnStart() error {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(p.cfg.Kafka.Brokers...),
		Topic:                  p.cfg.Kafka.Topic,
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: true, // удобно для dev, см. примечание ниже
	}

	if err := waitTopicReady(context.Background(), p.cfg.Kafka.Brokers, p.cfg.Kafka.Topic, 30*time.Second, p.log); err != nil {
		p.log.Warnw("kafka not ready yet, will try anyway", "error", err)
	}

	go func() {
		defer w.Close()
		interval := time.Second
		p.log.Infow("emitting", "brokers", p.cfg.Kafka.Brokers, "topic", p.cfg.Kafka.Topic, "interval", interval)

		ctx := context.Background()
		for n := 0; ; n++ {
			ord := makeDummyOrder(n)
			payload, _ := json.Marshal(ord)
			msg := kafka.Message{Key: []byte(ord.OrderId.String()), Value: payload, Time: time.Now()}

			if err := w.WriteMessages(ctx, msg); err != nil {
				p.log.Errorw("send failed", "order_uid", ord.OrderId.String(), "error", err)
			} else {
				p.log.Infow("sent", "order_uid", ord.OrderId.String())
			}
			time.Sleep(interval)
		}
	}()
	return nil
}

func waitTopicReady(ctx context.Context, brokers []string, topic string, timeout time.Duration, log *zap.SugaredLogger) error {
	deadline := time.Now().Add(timeout)
	backoff := 500 * time.Millisecond

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for topic %q", topic)
		}
		conn, err := kafka.DialContext(ctx, "tcp", brokers[0])
		if err == nil {
			parts, perr := conn.ReadPartitions(topic)
			_ = conn.Close()
			if perr == nil && len(parts) > 0 {
				allReady := true
				for _, p := range parts {
					if p.Leader.Host == "" {
						allReady = false
						break
					}
				}
				if allReady {
					log.Infow("kafka topic ready", "topic", topic, "partitions", len(parts))
					return nil
				}
			}
		}
		log.Debugw("waiting kafka...", "topic", topic)
		time.Sleep(backoff)
		if backoff < 5*time.Second {
			backoff *= 2
		}
	}
}

func makeDummyOrder(n int) entities.Order {
	uid := uuid.New()
	return entities.Order{
		OrderId:           uid,
		TrackNumber:       fmt.Sprintf("TESTTRACK-%06d", n),
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "",
		CustomerId:        "emitter",
		DeliveryService:   "emitter-delivery",
		ShardKey:          int64(rand.Intn(1000)),
		SmId:              rand.Intn(1000),
		DateCreated:       time.Now(),

		Delivery: entities.Delivery{
			Name:    "Load Gen",
			Phone:   "+100000000",
			City:    "GoCity",
			Address: "Benchmark str.",
		},
		Payment: entities.Payment{
			TransactionId: uid.String(),
			Amount:        int64(rand.Intn(5000) + 500),
			Currency:      "USD",
			Provider:      "emitter-pay",
			PaymentDt:     time.Now().Unix(),
			Bank:          "DemoBank",
		},
		Items: []entities.Item{{
			ChrtId:      int64(rand.Intn(1e7)),
			TrackNumber: "TESTTRACK",
			Name:        "DemoItem",
			Price:       999,
			TotalPrice:  999,
			Brand:       "EmitterCo",
			Status:      202,
		}},
	}
}
