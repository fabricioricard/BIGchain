package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Global P2P node pointer for blockchain operations
var globalP2PNode *P2PNode

func main() {
	fmt.Println("=== BIGchain - BIGFOOT Connect ===")
	fmt.Println("Real P2P Blockchain")
	fmt.Println()
	
	wallet := initializeWallet()
	
	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("⚠️  Error loading config: %v, using defaults\n", err)
		config = GetDefaultConfig()
	}
	
	// Tentar carregar blockchain local primeiro
	var bc *Blockchain
	if _, err := os.Stat("blockchain.json"); err == nil {
		fmt.Println("💾 Loading blockchain from local file...")
		bc, err = LoadBlockchainFromFile("blockchain.json")
		if err != nil {
			fmt.Printf("⚠️  Error loading local blockchain: %v\n", err)
		}
	}

	// Se não houver local ou falhar, tentar baixar do seed
	if bc == nil {
		fmt.Println("🔄 Downloading latest blockchain from seed node...")
		maxRetries := 5
		retryDelay := 2 * time.Second
		
		for attempt := 1; attempt <= maxRetries; attempt++ {
			if attempt > 1 {
				fmt.Printf("🔄 Retry attempt %d/%d in %v...\n", attempt, maxRetries, retryDelay)
				time.Sleep(retryDelay)
			}
			
			bc = downloadBlockchainFromSeed(wallet.Address)
			if bc != nil {
				break
			}
		}
	}

	// Se ainda for nil, criar uma nova (genesis)
	if bc == nil {
		fmt.Println("⚠️  Could not sync with network. Starting new blockchain...")
		bc = NewBlockchainWithWallet(wallet.Address)
	}

	bc.NodeID = wallet.Address
	bc.MiningStop = make(chan bool)
	if bc.ActiveMiners == nil {
		bc.ActiveMiners = make(map[string]*ActiveMiner)
	}
	
	fmt.Printf("✅ Blockchain ready: %d blocks, %.2f BIG supply\n", 
		len(bc.Chain), bc.getTotalSupply())
	
	defer bc.Stop()
	
	p2pNode := NewP2PNode(bc, config.P2P.Port)
	globalP2PNode = p2pNode
	
	fmt.Printf("Your Wallet Address: %s\n", wallet.Address)
	fmt.Printf("Private Key: %s\n", wallet.GetPrivateKeyHex())
	fmt.Printf("Latest Block Hash: %s\n", bc.GetLatestBlock().Hash)
	fmt.Printf("Current Balance: %.2f BIG\n", bc.GetBalance(wallet.Address))
	fmt.Println("⚠️  KEEP YOUR PRIVATE KEY SECRET AND SAFE! ⚠️")
	fmt.Println()
	
	// Auto-save blockchain every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := bc.SaveToFile("blockchain.json"); err != nil {
					fmt.Printf("⚠️  Error auto-saving blockchain: %v\n", err)
				} else {
					fmt.Println("💾 Blockchain auto-saved")
				}
			}
		}
	}()
	
	// Start P2P
	if err := p2pNode.Start(); err != nil {
		fmt.Printf("Error starting P2P node: %v\n", err)
		return
	}
	defer p2pNode.Stop()
	
	// Connect to seed nodes
	p2pNode.ConnectToSeeds()
	
	// Register as active miner
	bc.RegisterMiner(wallet.Address)
	
	// Send registration to network
	time.Sleep(2 * time.Second)
	p2pNode.RegisterAsMiner(wallet.Address)
	
	// Heartbeat
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				bc.UpdateMinerHeartbeat(wallet.Address)
				p2pNode.SendMinerHeartbeat(wallet.Address)
			}
		}
	}()
	
	fmt.Println("🚀 BIGchain node is now running!")
	showHelp()
	
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	
	reader := bufio.NewReader(os.Stdin)
	
	go func() {
		for {
			fmt.Print("BIGchain> ")
			input, _ := reader.ReadString('\n')
			command := strings.TrimSpace(input)
			
			switch command {
			case "status":
				showStatusWithMiners(bc)
			case "balance":
				fmt.Printf("Your balance: %.2f BIG\n", bc.GetBalance(wallet.Address))
			case "send":
				handleSendCommandWithWallet(bc, p2pNode, wallet, reader)
				case "peers":
					fmt.Printf("Connected peers: %d\n", p2pNode.GetPeerCount())
				case "connect":
					fmt.Print("Enter peer address (IP:Port): ")
					addr, _ := reader.ReadString('\n')
					addr = strings.TrimSpace(addr)
					if err := p2pNode.ConnectToPeer(addr); err != nil {
						fmt.Printf("❌ Connection failed: %v\n", err)
					} else {
						fmt.Println("✅ Connection attempt started!")
					}
				case "wallet":
				PrintWalletInfo(wallet)
			case "save":
				bc.SaveToFile("blockchain.json")
			case "help":
				showHelp()
			case "quit", "exit":
				shutdownChan <- syscall.SIGTERM
				return
			case "":
				continue
			default:
				fmt.Printf("Unknown command: %s\n", command)
			}
		}
	}()
	
	<-shutdownChan
	fmt.Println("Shutting down...")
	bc.SaveToFile("blockchain.json")
	wallet.SaveToFile(fmt.Sprintf("wallet_%s.json", wallet.Address[3:8]))
}

func downloadBlockchainFromSeed(walletAddress string) *Blockchain {
	seedNodes := []string{
		"45.191.11.128:8080", // IP do usuário para download da API
	}
	
	for _, seedAddr := range seedNodes {
		url := fmt.Sprintf("http://%s/api/blockchain/full", seedAddr)
		fmt.Printf("📥 Trying to download from: %s\n", seedAddr)
		
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			continue
		}
		
		var fullData struct {
			Chain []Block `json:"chain"`
		}
		
		if err := json.NewDecoder(resp.Body).Decode(&fullData); err != nil {
			continue
		}
		
		if len(fullData.Chain) == 0 {
			continue
		}
		
		bc := NewBlockchainWithWallet(walletAddress)
		bc.Chain = fullData.Chain
		return bc
	}
	
	return nil
}

func initializeWallet() *Wallet {
	files, _ := filepath.Glob("wallet_*.json")
	if len(files) > 0 {
		wallet, err := LoadWalletFromFile(files[0])
		if err == nil {
			return wallet
		}
	}
	return createNewWallet()
}

func createNewWallet() *Wallet {
	wallet, _ := NewWallet()
	filename := fmt.Sprintf("wallet_%s.json", wallet.Address[3:8])
	wallet.SaveToFile(filename)
	return wallet
}

func handleSendCommandWithWallet(bc *Blockchain, p2pNode *P2PNode, wallet *Wallet, reader *bufio.Reader) {
	fmt.Print("Enter recipient address: big")
	recipientInput, _ := reader.ReadString('\n')
	recipient := "big" + strings.TrimSpace(recipientInput)
	
	fmt.Print("Enter amount: ")
	amountInput, _ := reader.ReadString('\n')
	amount, _ := strconv.ParseFloat(strings.TrimSpace(amountInput), 64)
	
	tx := &Transaction{
		FromAddress: wallet.Address,
		ToAddress:   recipient,
		Amount:      amount,
		Timestamp:   time.Now().Unix(),
		TxType:      "transfer",
	}
	
	wallet.SignTransaction(tx)
	if bc.AddTransaction(tx) {
		p2pNode.BroadcastTransaction(tx)
		fmt.Println("✅ Transaction sent!")
	}
}

func showStatusWithMiners(bc *Blockchain) {
	info := bc.GetChainInfo()
	fmt.Printf("Blocks: %d | Supply: %.2f BIG | Peers: %d\n", info.Blocks, info.TotalSupply, len(bc.ActiveMiners))
}

func showHelp() {
	fmt.Println("Commands: status, balance, send, peers, connect, wallet, save, help, quit")
}
