package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/IBM/sarama"
	"github.com/sv410/Distributed-Real-Time-Push-Notification-Service/pkg"
)

// Consumer represents a Kafka consumer for notification messages
type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	topics        []string
	handler       *ConsumerGroupHandler
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler
type ConsumerGroupHandler struct {
	messageChan chan *pkg.NotificationMessage
	errorChan   chan error
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(brokers []string, groupID string, topics []string, messageChan chan *pkg.NotificationMessage, errorChan chan error) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	handler := &ConsumerGroupHandler{
		messageChan: messageChan,
		errorChan:   errorChan,
	}

	return &Consumer{
		consumerGroup: consumerGroup,
		topics:        topics,
		handler:       handler,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start starts consuming messages from Kafka
func (c *Consumer) Start() error {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			if err := c.consumerGroup.Consume(c.ctx, c.topics, c.handler); err != nil {
				select {
				case c.handler.errorChan <- fmt.Errorf("consumer error: %w", err):
				case <-c.ctx.Done():
					return
				}
			}

			// Check if context was cancelled
			if c.ctx.Err() != nil {
				return
			}
		}
	}()

	// Monitor consumer group errors
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for err := range c.consumerGroup.Errors() {
			select {
			case c.handler.errorChan <- fmt.Errorf("consumer group error: %w", err):
			case <-c.ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop stops the consumer
func (c *Consumer) Stop() error {
	log.Println("Stopping Kafka consumer...")
	c.cancel()
	c.wg.Wait()
	return c.consumerGroup.Close()
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			// Parse the notification message
			var notification pkg.NotificationMessage
			if err := json.Unmarshal(message.Value, &notification); err != nil {
				select {
				case h.errorChan <- fmt.Errorf("failed to unmarshal message: %w", err):
				case <-session.Context().Done():
					return nil
				}
				continue
			}

			// Send to message channel for processing
			select {
			case h.messageChan <- &notification:
				// Mark message as processed
				session.MarkMessage(message, "")
			case <-session.Context().Done():
				return nil
			}

		case <-session.Context().Done():
			return nil
		}
	}
}

// Producer represents a Kafka producer for testing purposes
type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

// NewProducer creates a new Kafka producer
func NewProducer(brokers []string, topic string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &Producer{
		producer: producer,
		topic:    topic,
	}, nil
}

// Send sends a notification message to Kafka
func (p *Producer) Send(notification *pkg.NotificationMessage) error {
	messageBytes, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	message := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(notification.UserID), // Use UserID as partition key
		Value: sarama.ByteEncoder(messageBytes),
	}

	_, _, err = p.producer.SendMessage(message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.producer.Close()
}

// HealthCheck performs a basic health check by attempting to get metadata
func HealthCheck(brokers []string) error {
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0

	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return fmt.Errorf("failed to create kafka client: %w", err)
	}
	defer client.Close()

	_, err = client.Topics()
	if err != nil {
		return fmt.Errorf("failed to fetch topics: %w", err)
	}

	return nil
}
