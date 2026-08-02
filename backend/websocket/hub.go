package websocket

import (
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	mu        sync.RWMutex
	documents map[uuid.UUID]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{documents: make(map[uuid.UUID]map[*Client]struct{})}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := h.documents[client.DocumentID]
	if clients == nil {
		clients = make(map[*Client]struct{})
		h.documents[client.DocumentID] = clients
	}
	clients[client] = struct{}{}
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := h.documents[client.DocumentID]
	if clients == nil {
		return
	}
	delete(clients, client)
	if len(clients) == 0 {
		delete(h.documents, client.DocumentID)
	}
}

func (h *Hub) BroadcastDocument(documentID uuid.UUID, payload []byte) {
	h.mu.RLock()
	clients := h.documents[documentID]
	targets := make([]*Client, 0, len(clients))
	for client := range clients {
		if !client.Ready.Load() {
			continue
		}
		targets = append(targets, client)
	}
	h.mu.RUnlock()

	for _, client := range targets {
		client.TrySend(payload)
	}
}

func (h *Hub) BroadcastDocumentExcept(documentID uuid.UUID, exclude *Client, payload []byte) {
	h.mu.RLock()
	clients := h.documents[documentID]
	targets := make([]*Client, 0, len(clients))
	for client := range clients {
		if client == exclude {
			continue
		}
		if !client.Ready.Load() {
			continue
		}
		targets = append(targets, client)
	}
	h.mu.RUnlock()

	for _, client := range targets {
		client.TrySend(payload)
	}
}

func (h *Hub) DisconnectDocument(documentID uuid.UUID, payload []byte) {
	h.mu.Lock()
	clients := h.documents[documentID]
	targets := make([]*Client, 0, len(clients))
	for client := range clients {
		if !client.Ready.Load() {
			continue
		}
		targets = append(targets, client)
	}
	delete(h.documents, documentID)
	h.mu.Unlock()

	for _, client := range targets {
		if payload != nil {
			client.TrySend(payload)
		}
		client.Close()
	}
}

func (h *Hub) DisconnectWorkspace(workspaceID uuid.UUID, payload []byte) {
	h.mu.Lock()
	targets := make([]*Client, 0)
	for documentID, clients := range h.documents {
		for client := range clients {
			if client.WorkspaceID != workspaceID || !client.Ready.Load() {
				continue
			}
			targets = append(targets, client)
			delete(clients, client)
		}
		if len(clients) == 0 {
			delete(h.documents, documentID)
		}
	}
	h.mu.Unlock()

	for _, client := range targets {
		if payload != nil {
			client.TrySend(payload)
		}
		client.Close()
	}
}
