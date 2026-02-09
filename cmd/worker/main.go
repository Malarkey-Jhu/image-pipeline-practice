package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/jackc/pgx/v5/pgxpool"

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

	obs.RegisterAll()
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", obs.MetricsHandler())
		log.Printf("worker metrics listening on :%s", cfg.WorkerMetricsPort)
		_ = http.ListenAndServe(":"+cfg.WorkerMetricsPort, mux)
	}()

	conn, err := amqp.Dial(cfg.RabbitURL())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		cfg.RabbitQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		panic(err)
	}

	msgs, err := ch.Consume(
		cfg.RabbitQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		panic(err)
	}

	workerID, _ := os.Hostname()
	if workerID == "" {
		workerID = "worker-unknown"
	}

	log.Printf("worker started: %s", workerID)

	for msg := range msgs {
		log.Printf("received message: %s", string(msg.Body))
		var task mq.TaskMessage
		if err := json.Unmarshal(msg.Body, &task); err != nil {
			log.Printf("bad message: %v", err)
			_ = msg.Reject(false)
			continue
		}

		log.Printf("decoded task: id=%s media=%s step=%s", task.TaskID, task.MediaID, task.Step)
		claimed, err := db.ClaimTask(ctx, pool, task.TaskID, workerID, cfg.TaskLeaseSeconds)
		if err != nil {
			log.Printf("claim failed: %v", err)
			_ = msg.Nack(false, true)
			continue
		}
		if !claimed {
			log.Printf("task %s not claimed (already locked or not eligible)", task.TaskID)
			_ = msg.Ack(false)
			continue
		}

		row, err := db.GetTask(ctx, pool, task.TaskID)
		if err != nil {
			log.Printf("load task failed: %v", err)
			_ = msg.Nack(false, true)
			continue
		}

		log.Printf("loaded task row: id=%s status=%s retry=%d output=%s", row.ID, row.Status, row.RetryCount, row.OutputKey)
		exists, err := store.ObjectExists(ctx, row.OutputKey)
		if err != nil {
			log.Printf("stat output failed: %v", err)
			_ = handleFailure(ctx, pool, msg, row.ID, row.RetryCount, cfg.TaskMaxRetries, err)
			continue
		}
		if exists {
			log.Printf("output exists, skipping task %s", row.ID)
			_ = db.MarkTaskSucceeded(ctx, pool, row.ID)
			obs.TasksSkipped.Inc()
			_ = msg.Ack(false)
			continue
		}

		log.Printf("processing task %s step=%s", row.ID, row.Step)
		// Simulate work
		time.Sleep(500 * time.Millisecond)

		if err := db.MarkTaskSucceeded(ctx, pool, row.ID); err != nil {
			log.Printf("mark succeeded failed: %v", err)
			_ = handleFailure(ctx, pool, msg, row.ID, row.RetryCount, cfg.TaskMaxRetries, err)
			continue
		}

		obs.TasksProcessed.Inc()
		_ = msg.Ack(false)
		log.Printf("task %s done", row.ID)
	}
}

func handleFailure(ctx context.Context, pool *pgxpool.Pool, msg amqp.Delivery, taskID string, retryCount int, maxRetries int, err error) error {
	if retryCount+1 >= maxRetries {
		_ = db.MarkTaskFailed(ctx, pool, taskID, err.Error())
		obs.TasksFailed.Inc()
		_ = msg.Nack(false, false)
		return nil
	}
	_ = db.MarkTaskRetry(ctx, pool, taskID, err.Error(), 30*time.Second)
	obs.TasksRetried.Inc()
	_ = msg.Nack(false, true)
	return nil
}
