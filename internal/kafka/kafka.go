// Package kafka provides message queue functionality (simulated with channels for demo)
package kafka

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Message represents a Kafka-like message
type Message struct {
	Key       string
	Value     []byte
	Timestamp time.Time
	Topic     string
	Partition int32
	Offset    int64
}

// Producer simulates a Kafka producer using channels
type Producer struct {
	topic   string
	logger  *logrus.Logger
	queue   chan *Message
	closed  chan bool
	mu      sync.RWMutex
	running bool
}

// Consumer simulates a Kafka consumer using channels
type Consumer struct {
	topic   string
	groupID string
	logger  *logrus.Logger
	queue   chan *Message
	mu      sync.RWMutex
	running bool
}

// Global message queue to simulate Kafka topics
var (
	messageQueues = make(map[string]chan *Message)
	queueMutex    sync.RWMutex
)

// getOrCreateQueue gets or creates a message queue for a topic
func getOrCreateQueue(topic string) chan *Message {
	queueMutex.Lock()
	defer queueMutex.Unlock()
	
	if queue, exists := messageQueues[topic]; exists {
		return queue
	}
	
	// Create a buffered channel to handle high throughput
	queue := make(chan *Message, 10000)
	messageQueues[topic] = queue
	return queue
}

// NewProducer creates a new Kafka-like producer
func NewProducer(bootstrapServers, topic string, logger *logrus.Logger) (*Producer, error) {
	queue := getOrCreateQueue(topic)
	
	producer := &Producer{
		topic:   topic,
		logger:  logger,
		queue:   queue,
		closed:  make(chan bool),
		running: true,
	}

	logger.WithFields(logrus.Fields{
		"topic":             topic,
		"bootstrap_servers": bootstrapServers,
	}).Info("Kafka producer initialized (simulated)")

	return producer, nil
}

// NewConsumer creates a new Kafka-like consumer
func NewConsumer(bootstrapServers, topic, groupID, autoOffsetReset string, logger *logrus.Logger) (*Consumer, error) {
	queue := getOrCreateQueue(topic)
	
	consumer := &Consumer{
		topic:   topic,
		groupID: groupID,
		logger:  logger,
		queue:   queue,
		running: true,
	}

	logger.WithFields(logrus.Fields{
		"topic":             topic,
		"group_id":          groupID,
		"bootstrap_servers": bootstrapServers,
	}).Info("Kafka consumer initialized (simulated)")

	return consumer, nil
}

// Produce sends a message to the queue
func (p *Producer) Produce(key string, value interface{}) error {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if !p.running {
		return fmt.Errorf("producer is closed")
	}

	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	message := &Message{
		Key:       key,
		Value:     valueBytes,
		Timestamp: time.Now(),
		Topic:     p.topic,
		Partition: 0,
		Offset:    time.Now().UnixNano(), // Simulate offset with timestamp
	}

	select {
	case p.queue <- message:
		p.logger.WithFields(logrus.Fields{
			"topic": p.topic,
			"key":   key,
		}).Debug("Message produced successfully")
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("produce timeout")
	case <-p.closed:
		return fmt.Errorf("producer is closed")
	}
}

// Consume reads messages from the queue
func (c *Consumer) Consume() (*Message, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if !c.running {
		return nil, fmt.Errorf("consumer is closed")
	}

	select {
	case msg := <-c.queue:
		return msg, nil
	case <-time.After(1 * time.Second):
		// Return timeout to allow for graceful shutdown checks
		return nil, fmt.Errorf("consume timeout")
	}
}

// Commit simulates committing a message offset
func (c *Consumer) Commit(msg *Message) error {
	// In a real Kafka implementation, this would commit the offset
	// For simulation, we just log it
	c.logger.WithFields(logrus.Fields{
		"topic":     msg.Topic,
		"offset":    msg.Offset,
		"partition": msg.Partition,
	}).Debug("Message committed")
	return nil
}

// Close closes the producer
func (p *Producer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.running {
		p.running = false
		close(p.closed)
		p.logger.Info("Producer closed")
	}
}

// Close closes the consumer
func (c *Consumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.running = false
	c.logger.Info("Consumer closed")
	return nil
}

// GetStats returns producer statistics (simulated)
func (p *Producer) GetStats() (string, error) {
	queueMutex.RLock()
	defer queueMutex.RUnlock()
	
	queue := messageQueues[p.topic]
	return fmt.Sprintf("Queue length: %d, Capacity: %d", len(queue), cap(queue)), nil
}