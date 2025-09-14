package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func NewConfig() (*ConfigModel, error) {
	_ = godotenv.Load(".env")

	c := &ConfigModel{}

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8081"
	}
	c.HTTP.Addr = addr

	c.Postgres.DSN = os.Getenv("PG_DSN")
	if c.Postgres.DSN == "" {
		c.Postgres.DSN = "postgres://wb_user:wb@localhost:5432/wb_orders?sslmode=disable"
	}

	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:29092"
	}
	c.Kafka.Brokers = splitAndTrim(brokers)
	if c.Kafka.Topic = os.Getenv("KAFKA_TOPIC"); c.Kafka.Topic == "" {
		c.Kafka.Topic = "orders-topic"
	}
	if c.Kafka.GroupID = os.Getenv("KAFKA_GROUP_ID"); c.Kafka.GroupID == "" {
		c.Kafka.GroupID = "orders-group"
	}

	c.Redis.Addr = getenvDefault("REDIS_ADDR", "localhost:6379")
	c.Redis.Password = os.Getenv("REDIS_PASSWORD")
	c.Redis.DB = atoiDefault("REDIS_DB", 0)
	c.Redis.TTLSeconds = atoiDefault("REDIS_TTL_SECONDS", 0)
	c.Redis.KeyPrefix = getenvDefault("REDIS_KEY_PREFIX", "orders:")

	return c, nil
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenvDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func atoiDefault(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
