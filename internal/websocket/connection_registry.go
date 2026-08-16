package websocket

import "sync"

// connectionRegistry tracks active browser tabs for each user. A single user
// may hold several tabs open at once, so connections are keyed by the per-tab
// connection ID (Client.ID, unique per tab) while a secondary index maps each
// UserID to every live tab for that user. This keeps every tab routable until
// it closes, and lets callers tell when the user's last tab goes away.
type connectionRegistry struct {
	mu        sync.RWMutex
	clients   map[string]*Client // connection ID -> client
	byUser    map[string]map[string]*Client // user ID -> connection ID -> client
}

func newConnectionRegistry() *connectionRegistry {
	return &connectionRegistry{
		clients: make(map[string]*Client),
		byUser:  make(map[string]map[string]*Client),
	}
}

// Add registers a connection. It returns true when this is the user's first
// live tab (an online transition) and false otherwise.
func (r *connectionRegistry) Add(client *Client) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	userTabs, existed := r.byUser[client.UserID]
	firstConnection := !existed || len(userTabs) == 0
	if firstConnection {
		userTabs = make(map[string]*Client)
		r.byUser[client.UserID] = userTabs
	}
	userTabs[client.ID] = client
	r.clients[client.ID] = client
	return firstConnection
}

// Remove drops the connection with the given ID. It returns
// (removed, lastConnection): whether a tab was actually removed, and whether
// the user no longer has any live tabs left afterwards.
func (r *connectionRegistry) Remove(connectionID string) (bool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	client, ok := r.clients[connectionID]
	if !ok {
		return false, false
	}
	delete(r.clients, connectionID)

	userTabs := r.byUser[client.UserID]
	delete(userTabs, connectionID)
	lastConnection := len(userTabs) == 0
	if lastConnection {
		delete(r.byUser, client.UserID)
	}
	return true, lastConnection
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

// ForUser returns every live tab for the given user, so messages fan out to
// all of their open browser tabs.
func (r *connectionRegistry) ForUser(userID string) []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	userTabs, ok := r.byUser[userID]
	if !ok {
		return nil
	}
	clients := make([]*Client, 0, len(userTabs))
	for _, client := range userTabs {
		clients = append(clients, client)
	}
	return clients
}

func (r *connectionRegistry) Usernames() []string {
	clients := r.All()
	users := make([]string, 0, len(clients))
	for _, client := range clients {
		users = append(users, client.Username)
	}
	return users
}
