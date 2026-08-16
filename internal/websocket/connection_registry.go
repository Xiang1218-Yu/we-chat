package websocket

import "sync"

// connectionRegistry keeps connection identity separate from user identity. A user
// may have multiple browser tabs, and every tab must remain a delivery target until
// that specific connection goes away.
type connectionRegistry struct {
	mu       sync.RWMutex
	byID     map[string]*Client
	byUserID map[string]map[string]*Client
}

func newConnectionRegistry() *connectionRegistry {
	return &connectionRegistry{
		byID:     make(map[string]*Client),
		byUserID: make(map[string]map[string]*Client),
	}
}

// Add reports whether this is the user's first active connection.
func (r *connectionRegistry) Add(client *Client) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	connections := r.byUserID[client.UserID]
	if connections == nil {
		connections = make(map[string]*Client)
		r.byUserID[client.UserID] = connections
	}
	_, existed := r.byID[client.ID]
	r.byID[client.ID] = client
	connections[client.ID] = client
	return !existed && len(connections) == 1
}

// Remove reports whether a connection was removed and whether it was the user's
// final active connection.
func (r *connectionRegistry) Remove(connectionID string) (bool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	client, ok := r.byID[connectionID]
	if !ok {
		return false, false
	}
	delete(r.byID, connectionID)
	connections := r.byUserID[client.UserID]
	delete(connections, connectionID)
	if len(connections) != 0 {
		return true, false
	}
	delete(r.byUserID, client.UserID)
	return true, true
}

func (r *connectionRegistry) All() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	clients := make([]*Client, 0, len(r.byID))
	for _, client := range r.byID {
		clients = append(clients, client)
	}
	return clients
}

func (r *connectionRegistry) ForUser(userID string) []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	connections := r.byUserID[userID]
	clients := make([]*Client, 0, len(connections))
	for _, client := range connections {
		clients = append(clients, client)
	}
	return clients
}

func (r *connectionRegistry) Usernames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	users := make([]string, 0, len(r.byUserID))
	for _, connections := range r.byUserID {
		for _, client := range connections {
			users = append(users, client.Username)
			break
		}
	}
	return users
}

// TryEnqueue hands a frame to a specific websocket writer without letting a
// reconnect worker block behind a saturated client buffer.
func (r *connectionRegistry) TryEnqueue(client *Client, frame []byte) bool {
	// A reconnect worker waits for the writer even when its bounded buffer is full.
	// Callers therefore never receive a failed handoff signal.
	client.Send <- frame
	return true
}
