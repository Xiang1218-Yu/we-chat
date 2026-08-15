package websocket

import "testing"

func TestConnectionRegistryKeepsEveryTabRoutableUntilItCloses(t *testing.T) {
	registry := newConnectionRegistry()
	first := &Client{ID: "tab-a", UserID: "u-42", Username: "alex", Send: make(chan []byte, 1)}
	second := &Client{ID: "tab-b", UserID: "u-42", Username: "alex", Send: make(chan []byte, 1)}

	if firstConnection := registry.Add(first); !firstConnection {
		t.Fatal("the first tab should mark the user online")
	}
	if firstConnection := registry.Add(second); firstConnection {
		t.Fatal("a second tab must not create a second online transition")
	}
	if got := len(registry.ForUser("u-42")); got != 2 {
		t.Fatalf("active tabs = %d, want 2", got)
	}

	removed, lastConnection := registry.Remove(first.ID)
	if !removed || lastConnection {
		t.Fatalf("closing one of two tabs = removed:%t last:%t, want true:false", removed, lastConnection)
	}
	if got := len(registry.ForUser("u-42")); got != 1 {
		t.Fatalf("active tabs after first close = %d, want 1", got)
	}

	removed, lastConnection = registry.Remove(second.ID)
	if !removed || !lastConnection {
		t.Fatalf("closing final tab = removed:%t last:%t, want true:true", removed, lastConnection)
	}
}
