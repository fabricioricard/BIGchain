package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type ActiveMiner struct {
	Address  string    `json:"address"`
	LastSeen time.Time `json:"last_seen"`
	IsActive bool      `json:"is_active"`
}

type Blockchain struct {
	Chain               []Block                 `json:"chain"`
	PendingTransactions []Transaction           `json:"pending_transactions"`
	MiningReward        float64                 `json:"mining_reward"`
	Difficulty          int                     `json:"difficulty"`
	NodeID              string                  `json:"node_id"`
	IsMining            bool                    `json:"is_mining"`
	MiningStop          chan bool               `json:"-"`
	ActiveMiners        map[string]*ActiveMiner `json:"active_miners"`
	CooperativeMode     bool                    `json:"cooperative_mode"`
	mu                  sync.RWMutex            `json:"-"`
}

func NewBlockchainWithWallet(walletAddress string) *Blockchain {
	bc := &Blockchain{
		Chain:               make([]Block, 0),
		PendingTransactions: make([]Transaction, 0),
		MiningReward:        1.0,
		Difficulty:          0, // Sustentável: sem necessidade de PoW pesado
		NodeID:              walletAddress,
		ActiveMiners:        make(map[string]*ActiveMiner),
		CooperativeMode:     true,
	}
	bc.createGenesisBlock()
	return bc
}

func (bc *Blockchain) createGenesisBlock() {
	genesis := Block{
		Index:        0,
		Timestamp:    time.Now().Unix(),
		Transactions: []Transaction{},
		PreviousHash: "0",
		MinedBy:      "genesis",
		PoR: ProofOfRelay{
			RelayCount:     0,
			SignatureChain: "genesis_root",
		},
	}
	genesis.Hash = genesis.CalculateHash()
	bc.Chain = append(bc.Chain, genesis)
}

func (bc *Blockchain) GetLatestBlock() Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.Chain[len(bc.Chain)-1]
}

func (bc *Blockchain) AddTransaction(tx *Transaction) bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.PendingTransactions = append(bc.PendingTransactions, *tx)
	return true
}

// MinePendingTransactions agora foca no Proof of Relay
func (bc *Blockchain) MinePendingTransactions(minerAddress string) *Block {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	latestBlock := bc.Chain[len(bc.Chain)-1]
	
	// No Proof of Relay real, esses dados viriam da atividade de rede do nó
	// Aqui simulamos a coleta de provas de retransmissão
	porData := ProofOfRelay{
		RelayCount:     150, // Exemplo de pacotes retransmitidos
		SignatureChain: fmt.Sprintf("sig_%x_%d", latestBlock.Hash, time.Now().UnixNano()),
	}

	newBlock := Block{
		Index:        len(bc.Chain),
		Timestamp:    time.Now().Unix(),
		Transactions: bc.PendingTransactions,
		PreviousHash: latestBlock.Hash,
		Difficulty:   0, // Sustentabilidade máxima
		MinedBy:      minerAddress,
		PoR:          porData,
	}

	// O hash é calculado instantaneamente, sem gasto de energia (PoW = 0)
	newBlock.Hash = newBlock.CalculateHash()

	bc.Chain = append(bc.Chain, newBlock)
	bc.PendingTransactions = []Transaction{}
	
	fmt.Printf("✨ Bloco %d gerado via Proof of Relay (Retransmissões: %d)\n", 
		newBlock.Index, newBlock.PoR.RelayCount)

	return &newBlock
}

func (bc *Blockchain) GetBalance(address string) float64 {
	balance := 0.0
	for _, block := range bc.Chain {
		for _, tx := range block.Transactions {
			if tx.ToAddress == address {
				balance += tx.Amount
			}
			if tx.FromAddress == address {
				balance -= tx.Amount
			}
		}
	}
	return balance
}

func (bc *Blockchain) getTotalSupply() float64 {
	supply := 0.0
	for _, block := range bc.Chain {
		for _, tx := range block.Transactions {
			if tx.FromAddress == "system" || tx.FromAddress == "" {
				supply += tx.Amount
			}
		}
	}
	return supply
}

func (bc *Blockchain) SaveToFile(filename string) error {
	data, _ := json.MarshalIndent(bc, "", "  ")
	return os.WriteFile(filename, data, 0644)
}

func LoadBlockchainFromFile(filename string) (*Blockchain, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var bc Blockchain
	json.Unmarshal(data, &bc)
	return &bc, nil
}

func (bc *Blockchain) Stop() {}
func (bc *Blockchain) RegisterMiner(addr string) {}
func (bc *Blockchain) UpdateMinerHeartbeat(addr string) {}
func (bc *Blockchain) GetChainInfo() ChainInfo {
	return ChainInfo{Blocks: len(bc.Chain), TotalSupply: bc.getTotalSupply()}
}

type ChainInfo struct {
	Blocks      int
	TotalSupply float64
}
