package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthChecker manages health checks for peers
type HealthChecker struct {
	nodeID       string
	peers        map[string]string // peerID -> address
	client       *http.Client
	mu           sync.RWMutex
	interval     time.Duration
	stopChan     chan struct{}
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(nodeID string, interval, requestTimeout time.Duration) *HealthChecker {
	return &HealthChecker{
		nodeID:   nodeID,
		peers:    make(map[string]string),
		client:   &http.Client{Timeout: requestTimeout},
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// AddPeer adds a peer to monitor
func (h *HealthChecker) AddPeer(peerID, address string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.peers[peerID] = address
}

// RemovePeer removes a peer from monitoring
func (h *HealthChecker) RemovePeer(peerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.peers, peerID)
}

// Start begins the health check loop
func (h *HealthChecker) Start(callback func(peerID string, isHealthy bool)) {
	ticker := time.NewTicker(h.interval)
	
	go func() {
		for {
			select {
			case <-ticker.C:
				h.checkAllPeers(callback)
			case <-h.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the health checker
func (h *HealthChecker) Stop() {
	close(h.stopChan)
}

// checkAllPeers checks the health of all peers
func (h *HealthChecker) checkAllPeers(callback func(peerID string, isHealthy bool)) {
	h.mu.RLock()
	peersCopy := make(map[string]string)
	for k, v := range h.peers {
		peersCopy[k] = v
	}
	h.mu.RUnlock()
	
	var wg sync.WaitGroup
	for peerID, address := range peersCopy {
		wg.Add(1)
		go func(id, addr string) {
			defer wg.Done()
			healthy := h.checkPeer(addr)
			callback(id, healthy)
		}(peerID, address)
	}
	wg.Wait()
}

// checkPeer performs a health check on a single peer
func (h *HealthChecker) checkPeer(address string) bool {
	url := fmt.Sprintf("http://%s/health", address)
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}
	
	resp, err := h.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

// CheckPeerNow immediately checks a specific peer
func (h *HealthChecker) CheckPeerNow(peerID string) bool {
	h.mu.RLock()
	address, exists := h.peers[peerID]
	h.mu.RUnlock()
	
	if !exists {
		return false
	}
	
	return h.checkPeer(address)
}
