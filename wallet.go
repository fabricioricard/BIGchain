package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"
)

type Wallet struct {
	PrivateKey *ecdsa.PrivateKey `json:"-"` // Never serialize private key
	PublicKey  []byte            `json:"-"` // Never serialize
	Address    string            `json:"address"`
	CreatedAt  int64             `json:"created_at"`
	Label      string            `json:"label,omitempty"`
}

type WalletFile struct {
	PrivateKeyHex string `json:"private_key_hex"`
	PublicKeyHex  string `json:"public_key_hex"`
	Address       string `json:"address"`
	CreatedAt     int64  `json:"created_at"`
	Label         string `json:"label,omitempty"`
}

type WalletManager struct {
	wallets map[string]*Wallet
	mu      sync.RWMutex
}

// NewWallet creates a new wallet with ECDSA cryptographic keys
func NewWallet() (*Wallet, error) {
	// Generate ECDSA private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %v", err)
	}

	// Extract public key
	publicKey := append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...)

	// Generate address
	address := generateAddress(publicKey)

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
		CreatedAt:  time.Now().Unix(),
	}, nil
}

// generateAddress generates a BIG address from public key
// Uses SHA256 hashing for deterministic address generation
func generateAddress(publicKey []byte) string {
	if len(publicKey) == 0 {
		return ""
	}

	// Hash public key twice for security
	hash1 := sha256.Sum256(publicKey)
	hash2 := sha256.Sum256(hash1[:])

	// Take first 20 bytes and convert to hex
	addressBytes := hash2[:20]
	addressHex := hex.EncodeToString(addressBytes)

	// Add "big" prefix
	return "big" + addressHex
}

// GetPrivateKeyHex returns private key in hexadecimal format
// ⚠️ WARNING: Returns sensitive data!
func (w *Wallet) GetPrivateKeyHex() string {
	if w.PrivateKey == nil {
		return ""
	}

	// Pad to 32 bytes (256 bits)
	privateKeyBytes := make([]byte, 32)
	d := w.PrivateKey.D.Bytes()
	copy(privateKeyBytes[len(privateKeyBytes)-len(d):], d)

	return hex.EncodeToString(privateKeyBytes)
}

// GetPublicKeyHex returns public key in hexadecimal format
func (w *Wallet) GetPublicKeyHex() string {
	return hex.EncodeToString(w.PublicKey)
}

// SignTransaction signs a transaction with the private key
func (w *Wallet) SignTransaction(tx *Transaction) error {
	if w.PrivateKey == nil {
		return fmt.Errorf("private key is nil, cannot sign")
	}

	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	// Calculate transaction hash
	txHash := tx.CalculateHash()
	if txHash == "" {
		return fmt.Errorf("failed to calculate transaction hash")
	}

	hashBytes, err := hex.DecodeString(txHash)
	if err != nil {
		return fmt.Errorf("failed to decode hash: %v", err)
	}

	// Sign hash with ECDSA
	r, s, err := ecdsa.Sign(rand.Reader, w.PrivateKey, hashBytes)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Combine r and s into signature (64 bytes)
	signature := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Pad to 32 bytes each
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)

	tx.Signature = hex.EncodeToString(signature)
	tx.Hash = txHash

	return nil
}

// VerifyTransactionSignature verifies if a transaction signature is valid
func VerifyTransactionSignature(tx *Transaction, publicKey []byte) bool {
	if tx == nil || tx.Signature == "" || len(publicKey) == 0 {
		return false
	}

	// Decode signature
	signatureBytes, err := hex.DecodeString(tx.Signature)
	if err != nil || len(signatureBytes) != 64 {
		return false
	}

	// Extract r and s
	r := new(big.Int).SetBytes(signatureBytes[:32])
	s := new(big.Int).SetBytes(signatureBytes[32:64])

	// Reconstruct public key
	curve := elliptic.P256()
	if len(publicKey) != 64 {
		return false
	}

	x := new(big.Int).SetBytes(publicKey[:32])
	y := new(big.Int).SetBytes(publicKey[32:64])

	pubKey := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}

	// Verify signature
	if tx.Hash == "" {
		tx.Hash = tx.CalculateHash()
	}

	hashBytes, err := hex.DecodeString(tx.Hash)
	if err != nil {
		return false
	}

	return ecdsa.Verify(pubKey, hashBytes, r, s)
}

// SaveToFile saves wallet to JSON file with restricted permissions
func (w *Wallet) SaveToFile(filename string) error {
	if w == nil {
		return fmt.Errorf("wallet is nil")
	}

	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	// Sanitize filename
	if len(filename) > 255 {
		return fmt.Errorf("filename too long")
	}

	walletFile := WalletFile{
		PrivateKeyHex: w.GetPrivateKeyHex(),
		PublicKeyHex:  w.GetPublicKeyHex(),
		Address:       w.Address,
		CreatedAt:     w.CreatedAt,
		Label:         w.Label,
	}

	jsonData, err := json.MarshalIndent(walletFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize wallet: %v", err)
	}

	// Write with restricted permissions (0600 = rw-------)
	if err := os.WriteFile(filename, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write wallet file: %v", err)
	}

	fmt.Printf("💾 Wallet saved securely to: %s\n", filename)
	return nil
}

// LoadWalletFromFile loads wallet from JSON file
func LoadWalletFromFile(filename string) (*Wallet, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("wallet file not found: %s", filename)
	}

	// Read file
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet file: %v", err)
	}

	// Decode JSON
	var walletFile WalletFile
	if err := json.Unmarshal(jsonData, &walletFile); err != nil {
		return nil, fmt.Errorf("failed to decode wallet: %v", err)
	}

	// Validate address
	if !isValidAddress(walletFile.Address) {
		return nil, fmt.Errorf("invalid address in wallet file")
	}

	// Reconstruct private key
	privateKeyBytes, err := hex.DecodeString(walletFile.PrivateKeyHex)
	if err != nil || len(privateKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid private key format")
	}

	privateKey := new(ecdsa.PrivateKey)
	privateKey.Curve = elliptic.P256()
	privateKey.D = new(big.Int).SetBytes(privateKeyBytes)
	privateKey.PublicKey.X, privateKey.PublicKey.Y = privateKey.Curve.ScalarBaseMult(privateKeyBytes)

	// Reconstruct public key
	publicKey, err := hex.DecodeString(walletFile.PublicKeyHex)
	if err != nil || len(publicKey) != 64 {
		return nil, fmt.Errorf("invalid public key format")
	}

	// Verify address matches
	calculatedAddress := generateAddress(publicKey)
	if calculatedAddress != walletFile.Address {
		return nil, fmt.Errorf("wallet address mismatch - possible corruption")
	}

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    walletFile.Address,
		CreatedAt:  walletFile.CreatedAt,
		Label:      walletFile.Label,
	}, nil
}

// ImportWalletFromPrivateKey imports wallet from private key hex
func ImportWalletFromPrivateKey(privateKeyHex string) (*Wallet, error) {
	if privateKeyHex == "" {
		return nil, fmt.Errorf("private key cannot be empty")
	}

	// Decode private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil || len(privateKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid private key format (must be 64 hex chars)")
	}

	// Reconstruct ECDSA private key
	privateKey := new(ecdsa.PrivateKey)
	privateKey.Curve = elliptic.P256()
	privateKey.D = new(big.Int).SetBytes(privateKeyBytes)
	privateKey.PublicKey.X, privateKey.PublicKey.Y = privateKey.Curve.ScalarBaseMult(privateKeyBytes)

	// Generate public key
	publicKey := append(privateKey.PublicKey.X.Bytes(), privateKey.PublicKey.Y.Bytes()...)

	// Generate address
	address := generateAddress(publicKey)

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
		CreatedAt:  time.Now().Unix(),
	}, nil
}

// NewWalletManager creates a new wallet manager
func NewWalletManager() *WalletManager {
	return &WalletManager{
		wallets: make(map[string]*Wallet),
	}
}

// CreateWallet creates a new wallet and adds it to the manager
func (wm *WalletManager) CreateWallet() (*Wallet, error) {
	wallet, err := NewWallet()
	if err != nil {
		return nil, err
	}

	wm.mu.Lock()
	wm.wallets[wallet.Address] = wallet
	wm.mu.Unlock()

	fmt.Printf("✅ New wallet created: %s\n", wallet.Address)
	return wallet, nil
}

// ImportWallet imports a wallet from private key hex
func (wm *WalletManager) ImportWallet(privateKeyHex string) (*Wallet, error) {
	wallet, err := ImportWalletFromPrivateKey(privateKeyHex)
	if err != nil {
		return nil, err
	}

	wm.mu.Lock()
	wm.wallets[wallet.Address] = wallet
	wm.mu.Unlock()

	fmt.Printf("✅ Wallet imported: %s\n", wallet.Address)
	return wallet, nil
}

// GetWallet retrieves a wallet by address
func (wm *WalletManager) GetWallet(address string) (*Wallet, bool) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	wallet, exists := wm.wallets[address]
	return wallet, exists
}

// ListWallets returns all wallets
func (wm *WalletManager) ListWallets() []*Wallet {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	wallets := make([]*Wallet, 0, len(wm.wallets))
	for _, wallet := range wm.wallets {
		wallets = append(wallets, wallet)
	}
	return wallets
}

// SaveAllWallets saves all wallets to files
func (wm *WalletManager) SaveAllWallets() error {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	if len(wm.wallets) == 0 {
		fmt.Println("ℹ️  No wallets to save")
		return nil
	}

	for address, wallet := range wm.wallets {
		filename := fmt.Sprintf("wallet_%s.json", address[3:8])
		if err := wallet.SaveToFile(filename); err != nil {
			return fmt.Errorf("failed to save wallet %s: %v", address, err)
		}
	}

	fmt.Printf("✅ Saved %d wallets\n", len(wm.wallets))
	return nil
}

// RemoveWallet removes a wallet from manager
func (wm *WalletManager) RemoveWallet(address string) bool {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.wallets[address]; exists {
		delete(wm.wallets, address)
		fmt.Printf("✅ Wallet removed: %s\n", address)
		return true
	}

	return false
}

// GetWalletCount returns number of managed wallets
func (wm *WalletManager) GetWalletCount() int {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	return len(wm.wallets)
}

// PrintWalletInfo prints wallet information (non-sensitive)
func PrintWalletInfo(wallet *Wallet) {
	if wallet == nil {
		fmt.Println("❌ Wallet is nil")
		return
	}

	fmt.Println()
	fmt.Println("=== Wallet Information ===")
	fmt.Printf("Address:    %s\n", wallet.Address)
	fmt.Printf("Public Key: %s\n", wallet.GetPublicKeyHex()[:32] + "...")
	fmt.Printf("Created:    %s\n", time.Unix(wallet.CreatedAt, 0).Format("2006-01-02 15:04:05"))

	if wallet.Label != "" {
		fmt.Printf("Label:      %s\n", wallet.Label)
	}

	fmt.Println()
	fmt.Println("⚠️  KEEP YOUR PRIVATE KEY SECRET AND SAFE!")
	fmt.Println("❌ Never share your private key with anyone!")
	fmt.Println()
}

// PrintPrivateKeyWarning prints a security warning
func PrintPrivateKeyWarning() {
	fmt.Println()
	fmt.Println("════════════════════════════════════════")
	fmt.Println("⚠️  PRIVATE KEY WARNING ⚠️")
	fmt.Println("════════════════════════════════════════")
	fmt.Println("• NEVER share your private key")
	fmt.Println("• NEVER enter it in untrusted apps")
	fmt.Println("• STORE it in a SAFE LOCATION")
	fmt.Println("• Anyone with your private key can")
	fmt.Println("  steal all your BIG tokens!")
	fmt.Println("════════════════════════════════════════")
	fmt.Println()
}

// GenerateNewWalletCommand generates a new wallet interactively
func GenerateNewWalletCommand() {
	wallet, err := NewWallet()
	if err != nil {
		fmt.Printf("❌ Error creating wallet: %v\n", err)
		return
	}

	// Save automatically
	filename := fmt.Sprintf("wallet_%s.json", wallet.Address[3:8])
	if err := wallet.SaveToFile(filename); err != nil {
		fmt.Printf("❌ Error saving wallet: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("🎉 New wallet created successfully!")
	PrintWalletInfo(wallet)
	PrintPrivateKeyWarning()

	fmt.Printf("Your Private Key (KEEP SECRET): %s\n", wallet.GetPrivateKeyHex())
	fmt.Println()
}

// ImportWalletCommand imports a wallet from private key
func ImportWalletCommand(privateKeyHex string) {
	if privateKeyHex == "" {
		fmt.Println("❌ Private key cannot be empty")
		return
	}

	wallet, err := ImportWalletFromPrivateKey(privateKeyHex)
	if err != nil {
		fmt.Printf("❌ Error importing wallet: %v\n", err)
		return
	}

	// Save automatically
	filename := fmt.Sprintf("wallet_%s.json", wallet.Address[3:8])
	if err := wallet.SaveToFile(filename); err != nil {
		fmt.Printf("❌ Error saving wallet: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("✅ Wallet imported successfully!")
	PrintWalletInfo(wallet)
	fmt.Printf("💾 Wallet saved to: %s\n", filename)
	fmt.Println()
}

// ExportWalletPrivateKey exports wallet private key (with warning)
func ExportWalletPrivateKey(wallet *Wallet) string {
	if wallet == nil {
		return ""
	}

	PrintPrivateKeyWarning()
	return wallet.GetPrivateKeyHex()
}