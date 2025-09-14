package kafka

import "go.uber.org/fx"

func Module() fx.Option {
	return fx.Module("kafka",
		fx.Provide(
			NewConsumer,
			NewProducer,
		),
		fx.Invoke(
			func(c *Consumer) error { return c.OnStart() },
			func(p *Producer) error { return p.OnStart() },
		),
	)
}
