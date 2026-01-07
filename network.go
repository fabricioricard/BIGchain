package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type NetworkNode struct {
	blockchain *Blockchain
	p2pNode    *P2PNode
	port       int
	mu         sync.RWMutex
}

func NewNetworkNode(blockchain *Blockchain, p2pNode *P2PNode, port int) *NetworkNode {
	return &NetworkNode{
		blockchain: blockchain,
		p2pNode:    p2pNode,
		port:       port,
	}
}

func (n *NetworkNode) Start() {
	// Add CORS headers middleware
	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Content-Type", "application/json")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next(w, r)
		}
	}
	
	// Register routes with CORS
	http.HandleFunc("/status", corsMiddleware(n.handleStatus))
	http.HandleFunc("/balance", corsMiddleware(n.handleBalance))
	http.HandleFunc("/transaction", corsMiddleware(n.handleTransaction))
	http.HandleFunc("/mine", corsMiddleware(n.handleMine))
	http.HandleFunc("/chain", corsMiddleware(n.handleChain))
	http.HandleFunc("/peers", corsMiddleware(n.handlePeers))
	http.HandleFunc("/connect", corsMiddleware(n.handleConnect))
	http.HandleFunc("/health", corsMiddleware(n.handleHealth))
	
	fmt.Printf("🌐 API server started on http://localhost:%d\n", n.port)
	fmt.Println("📡 Available endpoints:")
	fmt.Println("  GET  /status      - Blockchain status")
	fmt.Println("  GET  /balance     - Get balance (query: address)")
	fmt.Println("  POST /transaction - Create transaction")
	fmt.Println("  GET  /mine        - Mine block (query: miner)")
	fmt.Println("  GET  /chain       - Get full chain")
	fmt.Println("  GET  /peers       - Connected peers count")
	fmt.Println("  POST /connect     - Connect to peer")
	fmt.Println("  GET  /health      - API health check")
	
	if err := http.ListenAndServe(fmt.Sprintf(":%d", n.port), nil); err != nil {
		fmt.Printf("❌ API server error: %v\n", err)
	}
}

func (n *NetworkNode) handleStatus(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	
	chainInfo := n.blockchain.GetChainInfo()
	
	status := map[string]interface{}{
		"blockchain": chainInfo,
		"peers":      n.p2pNode.GetPeerCount(),
		"timestamp":  getCurrentTimestamp(),
	}
	
	json.NewEncoder(w).Encode(status)
}

func (n *NetworkNode) handleBalance(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		address = n.blockchain.NodeID
	}
	
	// Validate address format
	if !isValidAddress(address) {
		http.Error(w, "Invalid address format", http.StatusBadRequest)
		return
	}
	
	n.mu.RLock()
	balance := n.blockchain.GetBalance(address)
	n.mu.RUnlock()
	
	response := map[string]interface{}{
		"address":   address,
		"balance":   balance,
		"timestamp": getCurrentTimestamp(),
	}
	
	json.NewEncoder(w).Encode(response)
}

func (n *NetworkNode) handleTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var tx Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	
	// Validate transaction
	if !isValidTransaction(&tx) {
		http.Error(w, "Invalid transaction data", http.StatusBadRequest)
		return
	}
	
	n.mu.Lock()
	success := n.blockchain.AddTransaction(&tx)
	n.mu.Unlock()
	
	if success {
		n.p2pNode.BroadcastTransaction(&tx)
		fmt.Printf("📝 Transaction received: %s -> %s (%.2f BIG)\n", 
			tx.FromAddress, tx.ToAddress, tx.Amount)
	}
	
	response := map[string]interface{}{
		"success":   success,
		"txHash":    tx.Hash,
		"timestamp": getCurrentTimestamp(),
	}
	
	json.NewEncoder(w).Encode(response)
}

func (n *NetworkNode) handleMine(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	minerAddress := r.URL.Query().Get("miner")
	if minerAddress == "" {
		minerAddress = n.blockchain.NodeID
	}
	
	// Validate miner address
	if !isValidAddress(minerAddress) {
		http.Error(w, "Invalid miner address", http.StatusBadRequest)
		return
	}
	
	n.mu.RLock()
	pendingCount := len(n.blockchain.PendingTransactions)
	n.mu.RUnlock()
	
	if pendingCount == 0 {
		response := map[string]interface{}{
			"success": false,
			"message": "No pending transactions to mine",
			"timestamp": getCurrentTimestamp(),
		}
		w.WriteHeader(http.StatusNoContent)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	n.mu.Lock()
	block := n.blockchain.MinePendingTransactions(minerAddress)
	n.mu.Unlock()
	
	n.p2pNode.BroadcastBlock(block)
	
	fmt.Printf("⛏️  Block mined: %s by %s\n", block.Hash, minerAddress)
	
	json.NewEncoder(w).Encode(block)
}

func (n *NetworkNode) handleChain(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	chainData := map[string]interface{}{
		"length": len(n.blockchain.Chain),
		"chain":  n.blockchain.Chain,
	}
	n.mu.RUnlock()
	
	json.NewEncoder(w).Encode(chainData)
}

func (n *NetworkNode) handlePeers(w http.ResponseWriter, r *http.Request) {
	peerCount := n.p2pNode.GetPeerCount()
	
	response := map[string]interface{}{
		"peer_count": peerCount,
		"timestamp":  getCurrentTimestamp(),
	}
	
	json.NewEncoder(w).Encode(response)
}

func (n *NetworkNode) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		Address string `json:"address"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	
	if request.Address == "" {
		http.Error(w, "Address required", http.StatusBadRequest)
		return
	}
	
	err := n.p2pNode.ConnectToPeer(request.Address)
	success := err == nil
	
	response := map[string]interface{}{
		"success":   success,
		"address":   request.Address,
		"error":     "",
		"timestamp": getCurrentTimestamp(),
	}
	
	if err != nil {
		response["error"] = err.Error()
		response["success"] = false
		w.WriteHeader(http.StatusInternalServerError)
	}
	
	json.NewEncoder(w).Encode(response)
}

func (n *NetworkNode) handleHealth(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	blocks := len(n.blockchain.Chain)
	peers := n.p2pNode.GetPeerCount()
	pending := len(n.blockchain.PendingTransactions)
	n.mu.RUnlock()
	
	health := map[string]interface{}{
		"status":              "healthy",
		"blocks":              blocks,
		"peers":               peers,
		"pending_transactions": pending,
		"timestamp":           getCurrentTimestamp(),
	}
	
	json.NewEncoder(w).Encode(health)
}

// Helper functions

func isValidTransaction(tx *Transaction) bool {
	if tx == nil {
		return false
	}
	
	// Validate addresses
	if !isValidAddress(tx.FromAddress) || !isValidAddress(tx.ToAddress) {
		return false
	}
	
	// Validate amount
	if tx.Amount <= 0 {
		return false
	}
	
	// FromAddress and ToAddress should be different
	if tx.FromAddress == tx.ToAddress {
		return false
	}
	
	return true
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}