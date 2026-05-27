package gossip

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// GossipMessage represents a gossip message containing counter state
type GossipMessage struct {
	NodeID    string          `json:"node_id"`
	Counter   json.RawMessage `json:"counter"`
	Timestamp time.Time       `json:"timestamp"`
}

// GossipProtocol manages gossip communication between nodes
type GossipProtocol struct {
	nodeID       string
	address      string
	peers        map[string]string // peerID -> address
	client       *http.Client
	mu           sync.RWMutex
	interval     time.Duration
	stopChan     chan struct{}
	counterMu    *sync.RWMutex
	getCounter   func() ([]byte, error)
	mergeCounter func([]byte) error
}

// NewGossipProtocol creates a new gossip protocol instance
func NewGossipProtocol(nodeID, address string, interval, requestTimeout time.Duration) *GossipProtocol {
	return &GossipProtocol{
		nodeID:   nodeID,
		address:  address,
		peers:    make(map[string]string),
		client:   &http.Client{Timeout: requestTimeout},
		interval: interval,
		stopChan: make(chan struct{}),
		counterMu: &sync.RWMutex{},
	}
}

// SetCounterCallbacks sets the callbacks for getting and merging counter state
func (g *GossipProtocol) SetCounterCallbacks(getFunc func() ([]byte, error), mergeFunc func([]byte) error) {
	g.counterMu.Lock()
	defer g.counterMu.Unlock()
	g.getCounter = getFunc
	g.mergeCounter = mergeFunc
}

// RegisterHandlers registers gossip HTTP handlers on an existing server mux
func (g *GossipProtocol) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/gossip", g.handleGossip)
}

// AddPeer adds a peer to the gossip protocol
func (g *GossipProtocol) AddPeer(peerID, address string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.peers[peerID] = address
}

// RemovePeer removes a peer from the gossip protocol
func (g *GossipProtocol) RemovePeer(peerID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.peers, peerID)
}

// Start begins the gossip protocol
func (g *GossipProtocol) Start() error {
	go g.gossipLoop()
	return nil
}

// Stop stops the gossip protocol
func (g *GossipProtocol) Stop() {
	select {
	case <-g.stopChan:
		return
	default:
		close(g.stopChan)
	}
}

// gossipLoop periodically sends gossip messages to random peers
func (g *GossipProtocol) gossipLoop() {
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			g.sendGossip()
		case <-g.stopChan:
			return
		}
	}
}

// sendGossip sends gossip messages to a subset of peers
func (g *GossipProtocol) sendGossip() {
	g.mu.RLock()
	g.counterMu.RLock()
	
	// Get current counter state
	if g.getCounter == nil {
		g.mu.RUnlock()
		g.counterMu.RUnlock()
		return
	}
	
	counterData, err := g.getCounter()
	if err != nil {
		g.mu.RUnlock()
		g.counterMu.RUnlock()
		fmt.Printf("Error getting counter for gossip: %v\n", err)
		return
	}
	
	// Copy peers map
	peersCopy := make(map[string]string)
	for k, v := range g.peers {
		peersCopy[k] = v
	}
	
	g.mu.RUnlock()
	g.counterMu.RUnlock()
	
	// Create gossip message
	msg := GossipMessage{
		NodeID:    g.nodeID,
		Counter:   counterData,
		Timestamp: time.Now(),
	}
	
	msgData, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("Error marshaling gossip message: %v\n", err)
		return
	}
	
	// Send to all peers (could be optimized to random subset)
	var wg sync.WaitGroup
	for peerID, address := range peersCopy {
		wg.Add(1)
		go func(id, addr string) {
			defer wg.Done()
			g.sendToPeer(addr, msgData)
		}(peerID, address)
	}
	wg.Wait()
}

// sendToPeer sends a gossip message to a specific peer
func (g *GossipProtocol) sendToPeer(address string, data []byte) {
	url := fmt.Sprintf("http://%s/gossip", address)
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := g.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// handleGossip handles incoming gossip messages
func (g *GossipProtocol) handleGossip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var msg GossipMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	
	// Merge the counter state
	g.counterMu.Lock()
	defer g.counterMu.Unlock()
	
	if g.mergeCounter != nil {
		if err := g.mergeCounter(msg.Counter); err != nil {
			fmt.Printf("Error merging counter from gossip: %v\n", err)
			http.Error(w, "Failed to merge counter", http.StatusInternalServerError)
			return
		}
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GossipToAllPeers immediately sends gossip to all peers
func (g *GossipProtocol) GossipToAllPeers() {
	g.sendGossip()
}
