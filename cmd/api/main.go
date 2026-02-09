package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"sys-design/internal/api"
	"sys-design/internal/config"
	"sys-design/internal/db"
	"sys-design/internal/mq"
	"sys-design/internal/obs"
	"sys-design/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.PostgresDSN())
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	store, err := storage.NewMinioStore(cfg)
	if err != nil {
		panic(err)
	}

	publisher, err := mq.NewPublisher(cfg)
	if err != nil {
		panic(err)
	}
	defer publisher.Close()

	obs.RegisterAll()

	r := gin.Default()
	r.Use(api.MetricsMiddleware())
	srv := &api.Server{DB: pool, Store: store, Publisher: publisher}
	srv.RegisterRoutes(r)

	s := &http.Server{
		Addr:           ":" + cfg.APIPort,
		Handler:        r,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {
		panic(err)
	}
}
