package counter

import (
	"encoding/json"
	"sync"
)

// PNCounter is a Positive-Negative Counter CRDT
// It handles conflict resolution by tracking increments and decrements separately
type PNCounter struct {
	mu         sync.RWMutex
	Increments map[string]int64 // nodeID -> increment count
	Decrements map[string]int64 // nodeID -> decrement count
}

// NewPNCounter creates a new PN Counter
func NewPNCounter() *PNCounter {
	return &PNCounter{
		Increments: make(map[string]int64),
		Decrements: make(map[string]int64),
	}
}

// Increment adds delta to the counter for a specific node
func (c *PNCounter) Increment(nodeID string, delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if delta > 0 {
		c.Increments[nodeID] += delta
	}
}

// Decrement subtracts delta from the counter for a specific node
func (c *PNCounter) Decrement(nodeID string, delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if delta > 0 {
		c.Decrements[nodeID] += delta
	}
}

// Value returns the current counter value
func (c *PNCounter) Value() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var sum int64
	for _, v := range c.Increments {
		sum += v
	}
	for _, v := range c.Decrements {
		sum -= v
	}
	return sum
}

// Merge combines two PN Counters (conflict resolution)
func (c *PNCounter) Merge(other *PNCounter) {
	c.mu.Lock()
	other.mu.RLock()
	defer c.mu.Unlock()
	defer other.mu.RUnlock()
	
	// Merge increments (take max for each node)
	for nodeID, val := range other.Increments {
		if current, exists := c.Increments[nodeID]; exists {
			if val > current {
				c.Increments[nodeID] = val
			}
		} else {
			c.Increments[nodeID] = val
		}
	}
	
	// Merge decrements (take max for each node)
	for nodeID, val := range other.Decrements {
		if current, exists := c.Decrements[nodeID]; exists {
			if val > current {
				c.Decrements[nodeID] = val
			}
		} else {
			c.Decrements[nodeID] = val
		}
	}
}

// ToJSON converts the counter to JSON for network transmission
func (c *PNCounter) ToJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	data := struct {
		Increments map[string]int64 `json:"increments"`
		Decrements map[string]int64 `json:"decrements"`
	}{
		Increments: c.Increments,
		Decrements: c.Decrements,
	}
	
	return json.Marshal(data)
}

// FromJSON creates a counter from JSON data
func (c *PNCounter) FromJSON(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	var parsed struct {
		Increments map[string]int64 `json:"increments"`
		Decrements map[string]int64 `json:"decrements"`
	}
	
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	
	c.Increments = parsed.Increments
	c.Decrements = parsed.Decrements
	
	return nil
}

// Clone creates a deep copy of the counter
func (c *PNCounter) Clone() *PNCounter {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	clone := NewPNCounter()
	
	for k, v := range c.Increments {
		clone.Increments[k] = v
	}
	
	for k, v := range c.Decrements {
		clone.Decrements[k] = v
	}
	
	return clone
}
