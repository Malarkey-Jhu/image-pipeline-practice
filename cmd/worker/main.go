package main

import (
	"context"
	"time"

	"sys-design/internal/config"
	"sys-design/internal/db"
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

	// Placeholder loop
	for {
		_ = pool
		time.Sleep(2 * time.Second)
	}
}
