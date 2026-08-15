package websocket

import "sync"

// connectionRegistry is intended to track active browser tabs for each user.
type connectionRegistry struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

func newConnectionRegistry() *connectionRegistry {
	return &connectionRegistry{clients: make(map[string]*Client)}
}

func (r *connectionRegistry) Add(client *Client) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, existed := r.clients[client.UserID]
	r.clients[client.UserID] = client
	return !existed
}

func (r *connectionRegistry) Remove(connectionID string) (bool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.clients[connectionID]
	if !ok {
		return false, false
	}
	delete(r.clients, connectionID)
	return true, true
}

func (r *connectionRegistry) All() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	clients := make([]*Client, 0, len(r.clients))
	for _, client := range r.clients {
		clients = append(clients, client)
	}
	return clients
}

func (r *connectionRegistry) ForUser(userID string) []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	client, ok := r.clients[userID]
	if !ok {
		return nil
	}
	return []*Client{client}
}

func (r *connectionRegistry) Usernames() []string {
	clients := r.All()
	users := make([]string, 0, len(clients))
	for _, client := range clients {
		users = append(users, client.Username)
	}
	return users
}
