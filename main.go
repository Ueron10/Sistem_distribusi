package main

import (
	"bytes"
	"context"
	"distributed-counter/api"
	"distributed-counter/config"
	"distributed-counter/counter"
	"distributed-counter/gossip"
	"distributed-counter/health"
	"distributed-counter/node"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Command line flags
	configFile := flag.String("config", "", "Path to configuration file")
	nodeID := flag.String("id", "", "Node ID (required if no config file)")
	address := flag.String("addr", "", "Listen address (e.g., localhost:8080)")
	peer := flag.String("peer", "", "Initial peer address to join (e.g., localhost:8081)")
	flag.Parse()
	
	// Load configuration
	var cfg *config.Config
	var err error
	
	if *configFile != "" {
		cfg, err = config.LoadConfig(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
	} else {
		if *nodeID == "" || *address == "" {
			fmt.Println("Usage:")
			fmt.Println("  With config file:")
			fmt.Println("    go run main.go -config config.json")
			fmt.Println("  Without config file:")
			fmt.Println("    go run main.go -id node1 -addr localhost:8080 [-peer localhost:8081]")
			os.Exit(1)
		}
		
		cfg = config.DefaultConfig(*nodeID, *address)
		
		if *peer != "" {
			cfg.Peers = []string{*peer}
		}
	}
	
	// Validate address
	if err := node.ValidateAddress(cfg.ListenAddress); err != nil {
		log.Fatalf("Invalid listen address: %v", err)
	}
	
	fmt.Printf("Starting Distributed Counter Node\n")
	fmt.Printf("Node ID: %s\n", cfg.NodeID)
	fmt.Printf("Listen Address: %s\n", cfg.ListenAddress)
	fmt.Printf("Initial Peers: %v\n", cfg.Peers)
	
	// Create node
	n := node.NewNode(cfg.NodeID, cfg.ListenAddress)
	
	// Add initial peers
	for _, peerAddr := range cfg.Peers {
		peerID := fmt.Sprintf("peer-%s", peerAddr)
		n.AddPeer(peerID, peerAddr)
		fmt.Printf("Added initial peer: %s at %s\n", peerID, peerAddr)
	}
	
	// Create CRDT counter
	c := counter.NewPNCounter()
	
	// Create gossip protocol
	gossipInterval := time.Duration(cfg.GossipInterval) * time.Millisecond
	requestTimeout := time.Duration(cfg.RequestTimeout) * time.Millisecond
	g := gossip.NewGossipProtocol(cfg.NodeID, cfg.ListenAddress, gossipInterval, requestTimeout)
	g.SetCounterCallbacks(
		func() ([]byte, error) {
			return c.ToJSON()
		},
		func(data []byte) error {
			tempCounter := counter.NewPNCounter()
			if err := tempCounter.FromJSON(data); err != nil {
				return err
			}
			c.Merge(tempCounter)
			return nil
		},
	)
	
	// Add peers to gossip
	for peerID, peer := range n.GetAllPeers() {
		g.AddPeer(peerID, peer.Address)
	}
	
	// Create health checker
	healthInterval := time.Duration(cfg.HealthCheckInterval) * time.Millisecond
	h := health.NewHealthChecker(cfg.NodeID, healthInterval, requestTimeout)
	
	// Add peers to health checker
	for peerID, peer := range n.GetAllPeers() {
		h.AddPeer(peerID, peer.Address)
	}
	
	// Set up health check callback
	h.Start(func(peerID string, isHealthy bool) {
		if isHealthy {
			n.UpdatePeerLastSeen(peerID)
		} else {
			n.MarkPeerFailed(peerID)
		}
	})
	
	// Create API
	hc := health.NewHealthChecker(cfg.NodeID, healthInterval, requestTimeout)
	api := api.NewAPI(n, c, g, hc)

	// Start gossip protocol
	if err := g.Start(); err != nil {
		log.Fatalf("Failed to start gossip protocol: %v", err)
	}
	fmt.Println("Gossip protocol started")

	// Start HTTP server
	mux := http.NewServeMux()
	api.RegisterHandlers(mux)
	g.RegisterHandlers(mux)
	server := &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: mux,
	}
	
	go func() {
		fmt.Printf("HTTP server listening on %s\n", cfg.ListenAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()
	
	// If we have initial peers, try to join them
	if len(cfg.Peers) > 0 {
		go func() {
			time.Sleep(2 * time.Second) // Wait for server to start
			for _, peerAddr := range cfg.Peers {
				joinCluster(cfg.NodeID, cfg.ListenAddress, peerAddr)
			}
		}()
	}
	
	fmt.Println("Node started successfully")
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("  POST /counter/increment - Increment the counter")
	fmt.Println("  GET  /counter/value    - Get current counter value")
	fmt.Println("  POST /cluster/join     - Join a new node to the cluster")
	fmt.Println("  GET  /health           - Health check endpoint")
	fmt.Println("  GET  /peers            - List all peers")
	
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	fmt.Println("\nShutting down gracefully...")
	
	// Stop health checker
	h.Stop()
	
	// Stop gossip protocol
	g.Stop()
	
	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	
	fmt.Println("Node stopped")
}

// joinCluster attempts to join a cluster via a peer
func joinCluster(nodeID, address, peerAddress string) {
	url := fmt.Sprintf("http://%s/cluster/join", peerAddress)
	
	payload := map[string]string{
		"node_id": nodeID,
		"address": address,
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Failed to marshal join payload: %v\n", err)
		return
	}
	
	fmt.Printf("Attempting to join cluster via %s...\n", peerAddress)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("Failed to join cluster: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Successfully joined cluster via %s\n", peerAddress)
	} else {
		fmt.Printf("Failed to join cluster (status: %d)\n", resp.StatusCode)
	}
}
