package http

import "go.uber.org/fx"

func Module() fx.Option {
	return fx.Module("http",
		fx.Provide(NewServer), 
		fx.Invoke(func(s *Server) error { return s.OnStart() }),
	)
}
