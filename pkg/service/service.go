package service

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"

	"github.com/myntra/cortex/pkg/config"
	"github.com/myntra/cortex/pkg/store"
)

// Service encapsulates the http server and the raft store
type Service struct {
	srv      *http.Server
	node     *store.Node
	listener net.Listener
}

// Shutdown the service
func (s *Service) Shutdown(ctx context.Context) error {
	s.srv.Shutdown(ctx)
	if err := s.node.Shutdown(); err != nil {
		return err
	}
	return nil
}

// Start the service
func (s *Service) Start() error {

	// start the raft node
	if err := s.node.Start(); err != nil {
		return err
	}

	// start the http service
	if err := s.srv.Serve(s.listener); err != nil {
		return err
	}

	return nil
}

// New returns the http service wrapper for the store.
func New(cfg *config.Config) (*Service, error) {

	node, err := store.NewNode(cfg)
	if err != nil {
		return nil, err
	}

	svc := &Service{
		node: node,
	}

	router := chi.NewRouter()
	router.Use(middleware.Recoverer)

	router.Post("/event", svc.leaderProxy(svc.eventHandler))

	router.Get("/rules", svc.getRulesHandler)
	router.Get("/rules/{id}", svc.getRuleHandler)
	router.Get("/rules/{id}/executions", svc.getRulesExecutions)
	router.Post("/rules", svc.leaderProxy(svc.addRuleHandler))
	router.Put("/rules", svc.leaderProxy(svc.updateRuleHandler))
	router.Delete("/rules/{id}", svc.leaderProxy(svc.removeRuleHandler))

	router.Get("/scripts", svc.getScriptListHandler)
	router.Get("/scripts/{id}", svc.getScriptHandler)
	router.Post("/scripts", svc.leaderProxy(svc.addScriptHandler))
	router.Put("/scripts", svc.leaderProxy(svc.updateScriptHandler))
	router.Delete("/scripts/{id}", svc.leaderProxy(svc.removeScriptHandler))

	router.Get("/leave/{id}", svc.leaveHandler)
	router.Post("/join", svc.joinHandler)

	srv := &http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      router,
	}

	svc.srv = srv
	svc.listener = cfg.HTTPListener

	return svc, nil
}
