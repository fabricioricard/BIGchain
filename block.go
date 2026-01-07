package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// ProofOfRelay armazena os dados necessários para validar a retransmissão de pacotes.
type ProofOfRelay struct {
	RelayCount     int    `json:"relay_count"`     // Quantidade de pacotes retransmitidos
	SignatureChain string `json:"signature_chain"` // Cadeia de assinaturas que prova a rota
}

type Block struct {
	Index          int           `json:"index"`
	Timestamp      int64         `json:"timestamp"`
	Transactions   []Transaction `json:"transactions"`
	PreviousHash   string        `json:"previous_hash"`
	Nonce          int           `json:"nonce"`
	Difficulty     int           `json:"difficulty"`
	MinedBy        string        `json:"mined_by"`
	PoR            ProofOfRelay  `json:"por"` // Mecanismo central de sustentabilidade
	Hash           string        `json:"hash"`
}

func (b *Block) CalculateHash() string {
	if b == nil {
		return ""
	}

	blockData := struct {
		Index        int           `json:"index"`
		Timestamp    int64         `json:"timestamp"`
		Transactions []Transaction `json:"transactions"`
		PreviousHash string        `json:"previous_hash"`
		Nonce        int           `json:"nonce"`
		Difficulty   int           `json:"difficulty"`
		MinedBy      string        `json:"mined_by"`
		PoR          ProofOfRelay  `json:"por"`
	}{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		Transactions: b.Transactions,
		PreviousHash: b.PreviousHash,
		Nonce:        b.Nonce,
		Difficulty:   b.Difficulty,
		MinedBy:      b.MinedBy,
		PoR:          b.PoR,
	}

	jsonData, _ := json.Marshal(blockData)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

func (b *Block) ValidateBlock() bool {
	if b == nil {
		return false
	}

	if b.Hash != b.CalculateHash() {
		return false
	}

	// No Proof of Relay, a validade depende da integridade da SignatureChain
	if b.PoR.RelayCount < 0 {
		return false
	}

	return true
}
