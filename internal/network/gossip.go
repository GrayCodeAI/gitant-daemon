package network

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// GossipSub wraps the GossipSub protocol for pub/sub messaging
type GossipSub struct {
	mu     sync.RWMutex
	ps     *pubsub.PubSub
	host   *Host
	ctx    context.Context
	topics map[string]*pubsub.Topic
}

// NewGossipSub creates a new GossipSub instance
func NewGossipSub(ctx context.Context, host *Host) (*GossipSub, error) {
	ps, err := pubsub.NewGossipSub(ctx, host.host)
	if err != nil {
		return nil, fmt.Errorf("creating GossipSub: %w", err)
	}

	return &GossipSub{
		ps:     ps,
		host:   host,
		ctx:    ctx,
		topics: make(map[string]*pubsub.Topic),
	}, nil
}

func (g *GossipSub) joinTopic(topic string) (*pubsub.Topic, error) {
	g.mu.RLock()
	if t, ok := g.topics[topic]; ok {
		g.mu.RUnlock()
		return t, nil
	}
	g.mu.RUnlock()

	g.mu.Lock()
	defer g.mu.Unlock()

	if t, ok := g.topics[topic]; ok {
		return t, nil
	}

	t, err := g.ps.Join(topic)
	if err != nil {
		return nil, fmt.Errorf("joining topic: %w", err)
	}
	g.topics[topic] = t
	return t, nil
}

// Subscribe subscribes to a topic
func (g *GossipSub) Subscribe(topic string) (*Subscription, error) {
	t, err := g.joinTopic(topic)
	if err != nil {
		return nil, err
	}

	sub, err := t.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("subscribing to topic: %w", err)
	}

	return &Subscription{
		topic: t,
		sub:   sub,
	}, nil
}

// Publish publishes a message to a topic
func (g *GossipSub) Publish(topic string, data []byte) error {
	t, err := g.joinTopic(topic)
	if err != nil {
		return err
	}

	return t.Publish(g.ctx, data)
}

// Topics returns the list of topics
func (g *GossipSub) Topics() []string {
	return g.ps.GetTopics()
}

// Close closes the GossipSub
func (g *GossipSub) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for name, topic := range g.topics {
		_ = topic.Close()
		delete(g.topics, name)
	}
	return nil
}

// Subscription represents a topic subscription
type Subscription struct {
	topic *pubsub.Topic
	sub   *pubsub.Subscription
}

// Next returns the next message
func (s *Subscription) Next(ctx context.Context) (*Message, error) {
	msg, err := s.sub.Next(ctx)
	if err != nil {
		return nil, fmt.Errorf("receiving message: %w", err)
	}

	return &Message{
		From: msg.ReceivedFrom,
		Data: msg.Data,
	}, nil
}

// Cancel cancels the subscription
func (s *Subscription) Cancel() {
	s.sub.Cancel()
}

// Message represents a pub/sub message
type Message struct {
	From peer.ID
	Data []byte
}
