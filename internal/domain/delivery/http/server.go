package http

import (
	"context"
	"encoding/json"
	"net/http"

	"order-service/config"
	"order-service/internal/domain/usecase"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Server struct {
	cfg *config.ConfigModel
	uc  *usecase.OrderUC
	log *zap.SugaredLogger
}

func NewServer(cfg *config.ConfigModel, uc *usecase.OrderUC, l *zap.Logger) (*Server, error) {
	return &Server{cfg: cfg, uc: uc, log: l.Named("http").Sugar()}, nil
}

func (s *Server) OnStart() error {
	go s.uc.WarmCache(context.Background())

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/index.html")
	})

	r.Get("/recent", func(w http.ResponseWriter, r *http.Request) {
		s.log.Infow("request", "method", "GET", "path", "/recent")
		ids, err := s.uc.RecentIDs(r.Context(), 20)
		if err != nil {
			s.log.Errorw("recent failed", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ids)
	})

	r.Get("/order/{uid}", func(w http.ResponseWriter, r *http.Request) {
		uid := chi.URLParam(r, "uid")
		s.log.Infow("request", "method", "GET", "path", "/order/{uid}", "order_uid", uid)

		if uid == "" {
			s.log.Warnw("missing uid")
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		obj, err := s.uc.Get(r.Context(), uid)
		if err != nil {
			s.log.Warnw("bad id", "order_uid", uid, "error", err)
			http.Error(w, "bad id: "+err.Error(), http.StatusBadRequest)
			return
		}
		if obj == nil {
			s.log.Infow("not found", "order_uid", uid)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		b, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			s.log.Errorw("encode error", "order_uid", uid, "error", err)
			http.Error(w, "encode error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		s.log.Infow("response ok", "order_uid", uid)
		_, _ = w.Write(b)
	})

	go func() {
		s.log.Infow("http listen", "addr", s.cfg.HTTP.Addr)
		if err := http.ListenAndServe(s.cfg.HTTP.Addr, r); err != nil && err != http.ErrServerClosed {
			s.log.Errorw("http serve error", "error", err)
		}
	}()
	return nil
}
