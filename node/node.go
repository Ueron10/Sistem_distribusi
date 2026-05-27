package node

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Node represents a single node in the distributed system
type Node struct {
	ID        string
	Address   string // Format: "host:port"
	Peers     map[string]*Peer
	mu        sync.RWMutex
	StartedAt time.Time
}

// Peer represents another node in the cluster
type Peer struct {
	ID          string
	Address     string
	LastSeen    time.Time
	IsActive    bool
	FailureCount int
}

// NewNode creates a new node
func NewNode(id, address string) *Node {
	return &Node{
		ID:        id,
		Address:   address,
		Peers:     make(map[string]*Peer),
		StartedAt: time.Now(),
	}
}

// AddPeer adds a peer to the node's peer list
func (n *Node) AddPeer(id, address string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if peer, exists := n.Peers[id]; exists {
		peer.Address = address
		peer.LastSeen = time.Now()
		peer.IsActive = true
		peer.FailureCount = 0
		return
	}

	n.Peers[id] = &Peer{
		ID:          id,
		Address:     address,
		LastSeen:    time.Now(),
		IsActive:    true,
		FailureCount: 0,
	}
}

// RemovePeer removes a peer from the node's peer list
func (n *Node) RemovePeer(id string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.Peers, id)
}

// GetPeer returns a peer by ID
func (n *Node) GetPeer(id string) (*Peer, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	peer, exists := n.Peers[id]
	return peer, exists
}

// GetAllPeers returns all peers
func (n *Node) GetAllPeers() map[string]*Peer {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	peers := make(map[string]*Peer)
	for k, v := range n.Peers {
		peers[k] = v
	}
	return peers
}

// UpdatePeerLastSeen updates the last seen timestamp for a peer
func (n *Node) UpdatePeerLastSeen(id string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if peer, exists := n.Peers[id]; exists {
		peer.LastSeen = time.Now()
		peer.IsActive = true
		peer.FailureCount = 0
	}
}

// MarkPeerFailed marks a peer as failed
func (n *Node) MarkPeerFailed(id string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	
	if peer, exists := n.Peers[id]; exists {
		peer.FailureCount++
		peer.LastSeen = time.Now()
		if peer.FailureCount >= 3 {
			peer.IsActive = false
		}
	}
}

// GetActivePeers returns only active peers
func (n *Node) GetActivePeers() []*Peer {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	var activePeers []*Peer
	for _, peer := range n.Peers {
		if peer.IsActive {
			activePeers = append(activePeers, peer)
		}
	}
	return activePeers
}

// GetPeerCount returns the number of peers
func (n *Node) GetPeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.Peers)
}

// GetActivePeerCount returns the number of active peers
func (n *Node) GetActivePeerCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	count := 0
	for _, peer := range n.Peers {
		if peer.IsActive {
			count++
		}
	}
	return count
}

// ValidateAddress checks if an address is valid
func ValidateAddress(address string) error {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid address format: %v", err)
	}
	
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	
	if port == "" {
		return fmt.Errorf("port cannot be empty")
	}
	
	return nil
}
