package api

import (
	"bytes"
	"context"
	"distributed-counter/counter"
	"distributed-counter/health"
	"distributed-counter/node"
	"distributed-counter/gossip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// API represents the HTTP API
type API struct {
	node      *node.Node
	counter   *counter.PNCounter
	gossip    *gossip.GossipProtocol
	health    *health.HealthChecker
	mu        sync.RWMutex
}

// NewAPI creates a new API instance
func NewAPI(n *node.Node, c *counter.PNCounter, g *gossip.GossipProtocol, h *health.HealthChecker) *API {
	return &API{
		node:    n,
		counter: c,
		gossip:  g,
		health:  h,
	}
}

// IncrementRequest represents an increment request
type IncrementRequest struct {
	Delta int64 `json:"delta"`
}

// IncrementResponse represents an increment response
type IncrementResponse struct {
	Value    int64  `json:"value"`
	NodeID   string `json:"node_id"`
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
}

// ValueResponse represents a counter value response
type ValueResponse struct {
	Value      int64  `json:"value"`
	NodeID     string `json:"node_id"`
	PeerCount  int    `json:"peer_count"`
	ActivePeers int   `json:"active_peers"`
}

// JoinRequest represents a join cluster request
type JoinRequest struct {
	NodeID    string `json:"node_id"`
	Address   string `json:"address"`
	Propagate *bool  `json:"propagate,omitempty"`
}

// JoinResponse represents a join cluster response
type JoinResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	NodeID   string `json:"node_id"`
	PeerCount int   `json:"peer_count"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status      string `json:"status"`
	NodeID      string `json:"node_id"`
	Uptime      string `json:"uptime"`
	PeerCount   int    `json:"peer_count"`
	ActivePeers int    `json:"active_peers"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
}

// RegisterHandlers registers all API handlers
func (a *API) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/counter/increment", a.handleIncrement)
	mux.HandleFunc("/counter/value", a.handleGetValue)
	mux.HandleFunc("/counter/state", a.handleGetState)
	mux.HandleFunc("/cluster/join", a.handleJoin)
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc("/peers", a.handleListPeers)
}

// handleIncrement handles POST /counter/increment
func (a *API) handleIncrement(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req IncrementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	
	// Default delta to 1 if not specified
	if req.Delta == 0 {
		req.Delta = 1
	}
	
	a.mu.Lock()
	// Increment local counter
	a.counter.Increment(a.node.ID, req.Delta)
	localValue := a.counter.Value()
	a.mu.Unlock()
	
	// Trigger gossip to propagate changes
	go a.gossip.GossipToAllPeers()
	
	resp := IncrementResponse{
		Value:   localValue,
		NodeID:  a.node.ID,
		Success: true,
		Message: "Counter incremented successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleGetValue handles GET /counter/value
func (a *API) handleGetValue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.RLock()
	peerCount := a.node.GetPeerCount()
	activePeers := a.node.GetActivePeerCount()
	peers := a.node.GetActivePeers()
	a.mu.RUnlock()

	for _, peer := range peers {
		if peer.Address == a.node.Address {
			continue
		}

		stateData, err := a.fetchPeerCounterState(peer.Address)
		if err != nil {
			continue
		}

		tempCounter := counter.NewPNCounter()
		if err := tempCounter.FromJSON(stateData); err != nil {
			continue
		}

		a.mu.Lock()
		a.counter.Merge(tempCounter)
		a.mu.Unlock()
	}

	a.mu.RLock()
	value := a.counter.Value()
	a.mu.RUnlock()

	resp := ValueResponse{
		Value:      value,
		NodeID:     a.node.ID,
		PeerCount:  peerCount,
		ActivePeers: activePeers,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleGetState handles GET /counter/state
func (a *API) handleGetState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	a.mu.RLock()
	counterData, err := a.counter.ToJSON()
	peerCount := a.node.GetPeerCount()
	activePeers := a.node.GetActivePeerCount()
	a.mu.RUnlock()

	if err != nil {
		a.sendError(w, "Failed to serialize state", http.StatusInternalServerError)
		return
	}

	var state map[string]interface{}
	if err := json.Unmarshal(counterData, &state); err != nil {
		a.sendError(w, "Failed to parse counter state", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_id":      a.node.ID,
		"peer_count":   peerCount,
		"active_peers": activePeers,
		"state":        state,
	})
}

// handleJoin handles POST /cluster/join
func (a *API) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req JoinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		a.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.NodeID == "" || req.Address == "" {
		a.sendError(w, "node_id and address are required", http.StatusBadRequest)
		return
	}

	// Validate address format
	if err := node.ValidateAddress(req.Address); err != nil {
		a.sendError(w, fmt.Sprintf("Invalid address: %v", err), http.StatusBadRequest)
		return
	}

	propagate := true
	if req.Propagate != nil {
		propagate = *req.Propagate
	}

	a.mu.Lock()
	// Add peer to local node
	a.node.AddPeer(req.NodeID, req.Address)

	// Add peer to gossip protocol
	a.gossip.AddPeer(req.NodeID, req.Address)

	if a.health != nil {
		a.health.AddPeer(req.NodeID, req.Address)
	}

	peerCount := a.node.GetPeerCount()
	a.mu.Unlock()

	if propagate {
		a.forwardJoin(req)
	}

	resp := JoinResponse{
		Success:   true,
		Message:   "Node joined cluster successfully",
		NodeID:    req.NodeID,
		PeerCount: peerCount,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (a *API) forwardJoin(req JoinRequest) {
	falseValue := false
	req.Propagate = &falseValue

	data, err := json.Marshal(req)
	if err != nil {
		return
	}

	a.mu.RLock()
	peers := a.node.GetAllPeers()
	a.mu.RUnlock()

	for _, peer := range peers {
		if peer.ID == req.NodeID || peer.Address == a.node.Address {
			continue
		}

		go func(address string) {
			url := fmt.Sprintf("http://%s/cluster/join", address)
			client := &http.Client{Timeout: 3 * time.Second}
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(data))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
		}(peer.Address)
	}
}

func (a *API) fetchPeerCounterState(address string) ([]byte, error) {
	url := fmt.Sprintf("http://%s/counter/state", address)
	client := &http.Client{Timeout: 3 * time.Second}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// handleHealth handles GET /health
func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	a.mu.RLock()
	uptime := time.Since(a.node.StartedAt).String()
	peerCount := a.node.GetPeerCount()
	activePeers := a.node.GetActivePeerCount()
	a.mu.RUnlock()
	
	resp := HealthResponse{
		Status:      "healthy",
		NodeID:      a.node.ID,
		Uptime:      uptime,
		PeerCount:   peerCount,
		ActivePeers: activePeers,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleListPeers handles GET /peers
func (a *API) handleListPeers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		a.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	a.mu.RLock()
	peers := a.node.GetAllPeers()
	peerList := make([]map[string]interface{}, 0, len(peers))
	
	for _, peer := range peers {
		peerInfo := map[string]interface{}{
			"id":       peer.ID,
			"address":  peer.Address,
			"active":   peer.IsActive,
			"last_seen": peer.LastSeen.Format(time.RFC3339),
		}
		peerList = append(peerList, peerInfo)
	}
	a.mu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"peers": peerList,
		"count": len(peerList),
	})
}

// sendError sends an error response
func (a *API) sendError(w http.ResponseWriter, message string, code int) {
	resp := ErrorResponse{
		Error: message,
		Code:  code,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp)
}
