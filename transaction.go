package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type Transaction struct {
	FromAddress string  `json:"from_address"`
	ToAddress   string  `json:"to_address"`
	Amount      float64 `json:"amount"`
	Timestamp   int64   `json:"timestamp"`
	TxType      string  `json:"tx_type"`
	Nonce       int64   `json:"nonce"`
	Signature   string  `json:"signature"`
	Hash        string  `json:"hash"`
}

func (tx *Transaction) CalculateHash() string {
	if tx == nil {
		return ""
	}

	txData := struct {
		FromAddress string  `json:"from_address"`
		ToAddress   string  `json:"to_address"`
		Amount      float64 `json:"amount"`
		Timestamp   int64   `json:"timestamp"`
		TxType      string  `json:"tx_type"`
		Nonce       int64   `json:"nonce"`
	}{
		FromAddress: tx.FromAddress,
		ToAddress:   tx.ToAddress,
		Amount:      tx.Amount,
		Timestamp:   tx.Timestamp,
		TxType:      tx.TxType,
		Nonce:       tx.Nonce,
	}

	jsonData, err := json.Marshal(txData)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}

func (tx *Transaction) Validate() bool {
	if tx == nil {
		return false
	}

	if tx.FromAddress == "" && tx.TxType != "mining_reward" {
		return false
	}

	if tx.ToAddress == "" {
		return false
	}

	if tx.Amount <= 0 {
		return false
	}

	if tx.TxType == "" {
		return false
	}

	validTypes := map[string]bool{
				"transfer":      true,
				"mining_reward": true,
				"relay_reward":  true, // Recompensa por retransmissão (PoR)
			}

	if !validTypes[tx.TxType] {
		return false
	}

	return true
}

func (tx *Transaction) GetTransactionValue() float64 {
	if tx == nil {
		return 0
	}
	return tx.Amount
}

func (tx *Transaction) PrintTransaction() {
	if tx == nil {
		fmt.Println("❌ Transaction is nil")
		return
	}

	fmt.Println()
	fmt.Println("=== Transaction ===")
	fmt.Printf("Type: %s\n", tx.TxType)
	fmt.Printf("From: %s\n", tx.FromAddress)
	fmt.Printf("To: %s\n", tx.ToAddress)
	fmt.Printf("Amount: %.8f BIG\n", tx.Amount)
	fmt.Printf("Timestamp: %d\n", tx.Timestamp)
	fmt.Printf("Hash: %s\n", tx.Hash)
	fmt.Println()
}