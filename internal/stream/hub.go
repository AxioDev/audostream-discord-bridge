package stream

import (
	"sync"
	"sync/atomic"

	"github.com/pion/rtp"
)

type Client struct {
	id      uint64
	packets chan *rtp.Packet
}

func newClient(id uint64, buffer int) *Client {
	if buffer <= 0 {
		buffer = 32
	}
	return &Client{
		id:      id,
		packets: make(chan *rtp.Packet, buffer),
	}
}

func (c *Client) ID() uint64 {
	return c.id
}

func (c *Client) Packets() <-chan *rtp.Packet {
	return c.packets
}

func (c *Client) close() {
	close(c.packets)
}

type Hub struct {
	mu      sync.RWMutex
	clients map[uint64]*Client
	nextID  uint64
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uint64]*Client),
	}
}

func (h *Hub) Register(buffer int) *Client {
	id := atomic.AddUint64(&h.nextID, 1)
	client := newClient(id, buffer)

	h.mu.Lock()
	h.clients[id] = client
	h.mu.Unlock()

	return client
}

func (h *Hub) Unregister(id uint64) {
	h.mu.Lock()
	if client, ok := h.clients[id]; ok {
		delete(h.clients, id)
		client.close()
	}
	h.mu.Unlock()
}

func (h *Hub) Broadcast(packet *rtp.Packet) {
	h.mu.RLock()
	for _, client := range h.clients {
		select {
		case client.packets <- packet:
		default:
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
