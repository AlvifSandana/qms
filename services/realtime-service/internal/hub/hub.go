package hub

import (
	"encoding/json"
	"log"
	"sync"
)

type Subscription struct {
	TenantID  string
	BranchID  string
	ServiceID string
}

type Client struct {
	ID           string
	Send         chan []byte
	Subscription Subscription
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

type SubscribeMessage struct {
	Action    string `json:"action"`
	TenantID  string `json:"tenant_id"`
	BranchID  string `json:"branch_id"`
	ServiceID string `json:"service_id"`
}

func New() *Hub {
	return &Hub{clients: make(map[string]*Client)}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client.ID] = client
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client.ID)
	close(client.Send)
}

func (h *Hub) UpdateSubscription(client *Client, sub Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()
	client.Subscription = sub
}

func (h *Hub) Broadcast(payload []byte, meta Subscription) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, client := range h.clients {
		if !match(client.Subscription, meta) {
			continue
		}
		select {
		case client.Send <- payload:
		default:
			log.Printf("drop message for client %s", client.ID)
		}
	}
}

func match(sub Subscription, meta Subscription) bool {
	if sub.TenantID != "" && meta.TenantID != sub.TenantID {
		return false
	}
	if sub.BranchID != "" && meta.BranchID != sub.BranchID {
		return false
	}
	if sub.ServiceID != "" && meta.ServiceID != sub.ServiceID {
		return false
	}
	return true
}

func ParseSubscribe(data []byte) (SubscribeMessage, bool) {
	var msg SubscribeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return SubscribeMessage{}, false
	}
	if msg.Action != "subscribe" && msg.Action != "unsubscribe" {
		return SubscribeMessage{}, false
	}
	return msg, true
}
