package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// BIGchainSeedNodes já está declarado em p2p.go - não redeclarar aqui

type SeedNodeInfo struct {
	NodeID       string    `json:"node_id"`
	Address      string    `json:"address"`
	Port         int       `json:"port"`
	Version      string    `json:"version"`
	BlockHeight  int       `json:"block_height"`
	TotalSupply  float64   `json:"total_supply"`
	LastSeen     time.Time `json:"last_seen"`
	IsSeed       bool      `json:"is_seed"`
	Uptime       int64     `json:"uptime_seconds"`
}

type NetworkStatus struct {
	TotalNodes    int            `json:"total_nodes"`
	SeedNodes     int            `json:"seed_nodes"`
	MaxHeight     int            `json:"max_block_height"`
	NetworkSupply float64        `json:"network_total_supply"`
	ActiveMiners  int            `json:"active_miners"`
	SeedNodeList  []SeedNodeInfo `json:"seed_node_list"`
}

type SeedNodeMetrics struct {
	mu              sync.RWMutex
	LastBlockHeight int
	TotalSupply     float64
	PeerCount       int
	LastCheck       time.Time
	StartTime       time.Time
	IsHealthy       bool
}

func (p *P2PNode) InitAsSeedNode(config SeedNodeConfig) error {
	if !config.IsSeedNode {
		return nil
	}

	if p.blockchain == nil {
		return fmt.Errorf("blockchain is nil, cannot initialize seed node")
	}

	fmt.Printf("🌱 Initializing as SEED NODE: %s\n", config.SeedNodeName)
	fmt.Printf("📡 Public address: %s\n", config.PublicAddress)
	fmt.Printf("📞 Max connections: %d\n", config.MaxConnections)
	fmt.Printf("📚 History retention: %d days\n", config.HistoryRetention)

	// Start seed node specific services
	go p.seedNodeAnnouncement(config)
	go p.seedNodeHealthCheck()
	go p.seedNodeRESTAPI(config)

	fmt.Println("✅ Seed node services started successfully!")
	return nil
}

func (p *P2PNode) seedNodeAnnouncement(config SeedNodeConfig) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for p.running {
		select {
		case <-ticker.C:
			p.announceSeedNode(config)
		}
	}
}

func (p *P2PNode) announceSeedNode(config SeedNodeConfig) {
	if p.blockchain == nil {
		return
	}

	seedInfo := SeedNodeInfo{
		NodeID:      p.nodeID,
		Address:     config.PublicAddress,
		Port:        p.port,
		Version:     "1.0.0",
		BlockHeight: len(p.blockchain.Chain),
		TotalSupply: p.blockchain.getTotalSupply(),
		LastSeen:    time.Now(),
		IsSeed:      true,
		Uptime:      int64(time.Since(time.Now()).Seconds()),
	}

	message := P2PMessage{
		Type:      "seed_announcement",
		Data:      seedInfo,
		Sender:    p.nodeID,
		Timestamp: time.Now().Unix(),
	}

	p.broadcastMessage(message, "")
	fmt.Printf("🌱 Seed node announced: Height %d, Supply %.2f BIG, Peers %d\n",
		seedInfo.BlockHeight, seedInfo.TotalSupply, p.GetPeerCount())
}

func (p *P2PNode) seedNodeHealthCheck() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for p.running {
		select {
		case <-ticker.C:
			p.performHealthCheck()
		}
	}
}

func (p *P2PNode) performHealthCheck() {
	if p.blockchain == nil {
		return
	}

	peerCount := p.GetPeerCount()
	blockHeight := len(p.blockchain.Chain)
	totalSupply := p.blockchain.getTotalSupply()

	status := "✅ Healthy"
	if peerCount < 1 {
		status = "⚠️  Low peers"
	}
	if blockHeight < 10 {
		status = "⚠️  Low blocks"
	}

	fmt.Printf("💚 Seed Health: %s | %d peers | %d blocks | %.2f BIG supply\n",
		status, peerCount, blockHeight, totalSupply)

	// Auto-save blockchain frequently for seed nodes
	if err := p.blockchain.SaveToFile("blockchain.json"); err != nil {
		fmt.Printf("⚠️  Seed node backup failed: %v\n", err)
	}
}

func (p *P2PNode) seedNodeRESTAPI(config SeedNodeConfig) {
	// REST API specifically for seed nodes
	http.HandleFunc("/api/network/status", p.handleNetworkStatus)
	http.HandleFunc("/api/blockchain/full", p.handleFullBlockchainDownload)
	http.HandleFunc("/api/blockchain/range", p.handleBlockchainRange)
	http.HandleFunc("/api/seed/info", p.handleSeedInfo)

	apiPort := ":8080"
	fmt.Printf("🌐 Seed node API started on port 8080\n")

	if err := http.ListenAndServe(apiPort, nil); err != nil {
		fmt.Printf("⚠️  Seed node API error: %v\n", err)
	}
}

func (p *P2PNode) handleNetworkStatus(w http.ResponseWriter, r *http.Request) {
	if p.blockchain == nil {
		http.Error(w, "Blockchain not initialized", http.StatusInternalServerError)
		return
	}

	peerCount := p.GetPeerCount()

	status := NetworkStatus{
		TotalNodes:    peerCount + 1,
		SeedNodes:     1,
		MaxHeight:     len(p.blockchain.Chain),
		NetworkSupply: p.blockchain.getTotalSupply(),
		ActiveMiners:  len(p.blockchain.GetActiveMiners()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(status); err != nil {
		fmt.Printf("⚠️  Error encoding network status: %v\n", err)
	}

	fmt.Printf("📊 Network status requested by %s\n", r.RemoteAddr)
}

func (p *P2PNode) handleFullBlockchainDownload(w http.ResponseWriter, r *http.Request) {
	if p.blockchain == nil {
		http.Error(w, "Blockchain not initialized", http.StatusInternalServerError)
		return
	}

	fmt.Printf("📥 Full blockchain download requested by %s\n", r.RemoteAddr)

	p.blockchain.mu.RLock()
	chainCopy := make([]Block, len(p.blockchain.Chain))
	copy(chainCopy, p.blockchain.Chain)
	totalSupply := p.blockchain.getTotalSupply()
	p.blockchain.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Disposition", "attachment; filename=bigchain_full.json")

	fullData := struct {
		Chain       []Block `json:"chain"`
		TotalSupply float64 `json:"total_supply"`
		LastUpdated int64   `json:"last_updated"`
		SeedNodeID  string  `json:"seed_node_id"`
	}{
		Chain:       chainCopy,
		TotalSupply: totalSupply,
		LastUpdated: time.Now().Unix(),
		SeedNodeID:  p.nodeID,
	}

	if err := json.NewEncoder(w).Encode(fullData); err != nil {
		fmt.Printf("❌ Error encoding blockchain: %v\n", err)
		return
	}

	fmt.Printf("📤 Sent complete blockchain (%d blocks) to %s\n",
		len(chainCopy), r.RemoteAddr)
}

func (p *P2PNode) handleBlockchainRange(w http.ResponseWriter, r *http.Request) {
	if p.blockchain == nil {
		http.Error(w, "Blockchain not initialized", http.StatusInternalServerError)
		return
	}

	startBlock := 0
	endBlock := len(p.blockchain.Chain)

	if start := r.URL.Query().Get("start"); start != "" {
		if val, err := strconv.Atoi(start); err == nil {
			startBlock = val
		}
	}
	if end := r.URL.Query().Get("end"); end != "" {
		if val, err := strconv.Atoi(end); err == nil {
			endBlock = val
		}
	}

	if startBlock < 0 || endBlock > len(p.blockchain.Chain) || startBlock > endBlock {
		http.Error(w, "Invalid block range", http.StatusBadRequest)
		return
	}

	p.blockchain.mu.RLock()
	blocks := make([]Block, endBlock-startBlock)
	copy(blocks, p.blockchain.Chain[startBlock:endBlock])
	totalBlocks := len(p.blockchain.Chain)
	p.blockchain.mu.RUnlock()

	response := struct {
		Blocks     []Block `json:"blocks"`
		StartBlock int     `json:"start_block"`
		EndBlock   int     `json:"end_block"`
		Total      int     `json:"total_blocks"`
	}{
		Blocks:     blocks,
		StartBlock: startBlock,
		EndBlock:   endBlock,
		Total:      totalBlocks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("❌ Error encoding block range: %v\n", err)
		return
	}

	fmt.Printf("📤 Sent blocks %d-%d to %s\n", startBlock, endBlock-1, r.RemoteAddr)
}

func (p *P2PNode) handleSeedInfo(w http.ResponseWriter, r *http.Request) {
	if p.blockchain == nil {
		http.Error(w, "Blockchain not initialized", http.StatusInternalServerError)
		return
	}

	activePeers := p.GetPeerCount()

	seedInfo := SeedNodeInfo{
		NodeID:      p.nodeID,
		Address:     "seed.bigchain",
		Port:        p.port,
		Version:     "1.0.0",
		BlockHeight: len(p.blockchain.Chain),
		TotalSupply: p.blockchain.getTotalSupply(),
		LastSeen:    time.Now(),
		IsSeed:      true,
		Uptime:      int64(time.Now().Unix()),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(seedInfo); err != nil {
		fmt.Printf("⚠️  Error encoding seed info: %v\n", err)
		return
	}

	fmt.Printf("ℹ️  Seed info requested by %s (Active peers: %d)\n", r.RemoteAddr, activePeers)
}

func (p *P2PNode) downloadFullBlockchain(seedAddr string) error {
	if p.blockchain == nil {
		return fmt.Errorf("blockchain is nil")
	}

	host := seedAddr
	if hostPart, _, err := net.SplitHostPort(seedAddr); err == nil {
		host = hostPart
	}

	url := fmt.Sprintf("http://%s:8080/api/blockchain/full", host)

	fmt.Printf("📥 Attempting to download from: %s\n", url)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to seed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, 500*1024*1024)

	var fullData struct {
		Chain       []Block `json:"chain"`
		TotalSupply float64 `json:"total_supply"`
		LastUpdated int64   `json:"last_updated"`
		SeedNodeID  string  `json:"seed_node_id"`
	}

	if err := json.NewDecoder(limitedReader).Decode(&fullData); err != nil {
		return fmt.Errorf("failed to decode blockchain: %v", err)
	}

	if len(fullData.Chain) == 0 {
		return fmt.Errorf("seed returned empty blockchain")
	}

	if len(fullData.Chain) > len(p.blockchain.Chain) {
		p.blockchain.mu.Lock()
		p.blockchain.Chain = fullData.Chain
		p.blockchain.mu.Unlock()

		if err := p.blockchain.SaveToFile("blockchain.json"); err != nil {
			return fmt.Errorf("failed to save blockchain: %v", err)
		}

		fmt.Printf("📊 Downloaded blockchain: %d blocks, %.2f BIG supply\n",
			len(fullData.Chain), fullData.TotalSupply)
		return nil
	}

	return fmt.Errorf("seed node has older or equal blockchain")
}

func (p *P2PNode) GetBestSeedNodes() []string {
	return BIGchainSeedNodes
}

func ShouldBecomeSeedNode(uptime time.Duration, blockHeight int, peerCount int) bool {
	isSeedCandidate := uptime > 24*time.Hour && blockHeight > 100 && peerCount >= 5

	if isSeedCandidate {
		fmt.Printf("💚 Node is eligible to become seed: uptime=%v, blocks=%d, peers=%d\n",
			uptime, blockHeight, peerCount)
	}

	return isSeedCandidate
}