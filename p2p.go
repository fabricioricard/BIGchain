package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type P2PNode struct {
	blockchain     *Blockchain
	address        string
	port           int
	peers          map[string]*PeerConnection
	peersMutex     sync.RWMutex
	listener       net.Listener
	running        bool
	messageQueue   chan P2PMessage
	syncInProgress bool
	seedNodes      []string
	knownPeers     map[string]bool // Cache de endereços conhecidos
	nodeID         string
}

type PeerConnection struct {
	Address    string
	Conn       net.Conn
	LastSeen   int64
	Connected  bool
}

type P2PMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Sender    string      `json:"sender"`
	Timestamp int64       `json:"timestamp"`
}

type BlockchainSyncRequest struct {
	FromBlock int    `json:"from_block"`
	NodeID    string `json:"node_id"`
}

type BlockchainSyncResponse struct {
	Blocks      []Block `json:"blocks"`
	TotalBlocks int     `json:"total_blocks"`
	NodeID      string  `json:"node_id"`
}

type PeerInfo struct {
	Address   string `json:"address"`
	NodeID    string `json:"node_id"`
	LastSeen  int64  `json:"last_seen"`
}

func NewP2PNode(blockchain *Blockchain, port int) *P2PNode {
	seedNodes := []string{"45.191.11.128:8333"} // IP do usuário como Seed Node principal
	
	node := &P2PNode{
		blockchain:     blockchain,
		port:           port,
		address:        fmt.Sprintf(":%d", port),
		peers:          make(map[string]*PeerConnection),
		running:        false,
		messageQueue:   make(chan P2PMessage, 100),
		syncInProgress: false,
		seedNodes:      seedNodes,
		knownPeers:     make(map[string]bool),
		nodeID:         fmt.Sprintf("node_%d_%d", port, time.Now().Unix()),
	}
	node.loadPeerCache()
	return node
}

func (p *P2PNode) loadPeerCache() {
	data, err := os.ReadFile("peer_cache.json")
	if err == nil {
		json.Unmarshal(data, &p.knownPeers)
	}
}

func (p *P2PNode) savePeerCache() {
	data, _ := json.MarshalIndent(p.knownPeers, "", "  ")
	os.WriteFile("peer_cache.json", data, 0644)
}

func (p *P2PNode) Start() error {
	listener, err := net.Listen("tcp", p.address)
	if err != nil {
		return err
	}
	p.listener = listener
	p.running = true

	go p.processMessages()
	go p.acceptConnections()
	go p.peerDiscovery()
	
	fmt.Printf("🌐 P2P Node started on port %d\n", p.port)
	return nil
}

func (p *P2PNode) Stop() {
	p.running = false
	if p.listener != nil {
		p.listener.Close()
	}
}

func (p *P2PNode) acceptConnections() {
	for p.running {
		conn, err := p.listener.Accept()
		if err != nil {
			continue
		}
		go p.handleConnection(conn)
	}
}

func (p *P2PNode) handleConnection(conn net.Conn) {
	defer conn.Close()
	remoteAddr := conn.RemoteAddr().String()
	
	p.peersMutex.Lock()
	p.peers[remoteAddr] = &PeerConnection{
		Address:   remoteAddr,
		Conn:      conn,
		LastSeen:  time.Now().Unix(),
		Connected: true,
	}
	// Adicionar ao cache de pares conhecidos
	p.knownPeers[remoteAddr] = true
	p.savePeerCache()
	p.peersMutex.Unlock()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() && p.running {
		var msg P2PMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err == nil {
			msg.Sender = remoteAddr
			p.messageQueue <- msg
		}
	}

	p.peersMutex.Lock()
	delete(p.peers, remoteAddr)
	p.peersMutex.Unlock()
}

func (p *P2PNode) processMessages() {
	for msg := range p.messageQueue {
		switch msg.Type {
		case "peer_discovery_req":
			p.handlePeerDiscoveryRequest(msg)
		case "peer_discovery_res":
			p.handlePeerDiscoveryResponse(msg)
		case "new_block":
			// Lógica de bloco
		case "new_transaction":
			// Lógica de transação
		}
	}
}

// Peer Exchange (PEX) - Qualquer nó responde com sua lista de pares
func (p *P2PNode) handlePeerDiscoveryRequest(msg P2PMessage) {
	p.peersMutex.RLock()
	peerList := make([]string, 0, len(p.peers))
	for addr := range p.peers {
		peerList = append(peerList, addr)
	}
	p.peersMutex.RUnlock()

	response := P2PMessage{
		Type: "peer_discovery_res",
		Data: peerList,
		Sender: p.nodeID,
	}
	p.sendMessageToPeer(response, msg.Sender)
}

func (p *P2PNode) handlePeerDiscoveryResponse(msg P2PMessage) {
	peerList, ok := msg.Data.([]interface{})
	if !ok {
		return
	}
	for _, peerAddr := range peerList {
		addr := peerAddr.(string)
		p.peersMutex.RLock()
		_, exists := p.peers[addr]
		p.peersMutex.RUnlock()
		
		if !exists && addr != p.address {
			go p.ConnectToPeer(addr)
		}
	}
}

func (p *P2PNode) peerDiscovery() {
	ticker := time.NewTicker(1 * time.Minute)
	for p.running {
		select {
		case <-ticker.C:
			msg := P2PMessage{Type: "peer_discovery_req", Sender: p.nodeID}
			p.broadcastMessage(msg, "")
		}
	}
}

func (p *P2PNode) ConnectToPeer(address string) error {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return err
	}
	go p.handleConnection(conn)
	return nil
}

func (p *P2PNode) ConnectToSeeds() {
	// Tentar sementes oficiais
	for _, seed := range p.seedNodes {
		go p.ConnectToPeer(seed)
	}
	// Tentar cache de pares conhecidos de sessões anteriores
	p.peersMutex.RLock()
	for addr := range p.knownPeers {
		go p.ConnectToPeer(addr)
	}
	p.peersMutex.RUnlock()
}

func (p *P2PNode) broadcastMessage(msg P2PMessage, exclude string) {
	data, _ := json.Marshal(msg)
	p.peersMutex.RLock()
	defer p.peersMutex.RUnlock()
	for addr, peer := range p.peers {
		if addr != exclude {
			peer.Conn.Write(append(data, '\n'))
		}
	}
}

func (p *P2PNode) sendMessageToPeer(msg P2PMessage, addr string) {
	data, _ := json.Marshal(msg)
	p.peersMutex.RLock()
	peer, exists := p.peers[addr]
	p.peersMutex.RUnlock()
	if exists {
		peer.Conn.Write(append(data, '\n'))
	}
}

func (p *P2PNode) GetPeerCount() int {
	p.peersMutex.RLock()
	defer p.peersMutex.RUnlock()
	return len(p.peers)
}

func (p *P2PNode) BroadcastTransaction(tx *Transaction) {
	msg := P2PMessage{Type: "new_transaction", Data: tx, Sender: p.nodeID}
	p.broadcastMessage(msg, "")
}

func (p *P2PNode) BroadcastBlock(block *Block) {
	msg := P2PMessage{Type: "new_block", Data: block, Sender: p.nodeID}
	p.broadcastMessage(msg, "")
}

func (p *P2PNode) RegisterAsMiner(addr string) {}
func (p *P2PNode) SendMinerHeartbeat(addr string) {}
