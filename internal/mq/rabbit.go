package mq

import (
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"

	"sys-design/internal/config"
)

type Publisher struct {
	Conn  *amqp.Connection
	Ch    *amqp.Channel
	Queue string
}

type TaskMessage struct {
	TaskID  string `json:"task_id"`
	MediaID string `json:"media_id"`
	Step    string `json:"step"`
}

func NewPublisher(cfg *config.Config) (*Publisher, error) {
	conn, err := amqp.Dial(cfg.RabbitURL())
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	_, err = ch.QueueDeclare(
		cfg.RabbitQueue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &Publisher{Conn: conn, Ch: ch, Queue: cfg.RabbitQueue}, nil
}

func (p *Publisher) Close() {
	if p.Ch != nil {
		_ = p.Ch.Close()
	}
	if p.Conn != nil {
		_ = p.Conn.Close()
	}
}

func (p *Publisher) PublishTask(msg TaskMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.Ch.Publish(
		"",
		p.Queue,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
