package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

type NetworkConfig struct {
	SeedNodes      []string `json:"seed_nodes"`
	Environment    string   `json:"environment"` // "development", "production"
	SeedIP         string   `json:"seed_ip"`
	BackupSeeds    []string `json:"backup_seeds"`
}

type NodeConfig struct {
	Network  NetworkConfig  `json:"network"`
	SeedNode SeedNodeConfig `json:"seed_node"`
	Mining   MiningConfig   `json:"mining"`
	P2P      P2PConfig      `json:"p2p"`
	Sync     SyncConfig     `json:"sync"`
}

type MiningConfig struct {
	Enabled           bool    `json:"enabled"`
	Difficulty        int     `json:"difficulty"`
	BlockTime         int     `json:"block_time_minutes"`
	Reward            float64 `json:"mining_reward"`
	CooperativeMode   bool    `json:"cooperative_mode"`    // Novo: modo cooperativo
	RewardDistribution string `json:"reward_distribution"` // "all_nodes" ou "miner_only"
}

type P2PConfig struct {
	Port           int `json:"port"`
	BandwidthPort  int `json:"bandwidth_port"`
	APIPort        int `json:"api_port"`
	MaxConnections int `json:"max_connections"`
}

type SeedNodeConfig struct {
	IsSeedNode       bool   `json:"is_seed_node"`
	SeedNodeName     string `json:"seed_node_name"`
	MaxConnections   int    `json:"max_connections"`
	HistoryRetention int    `json:"history_retention_days"`
	PublicAddress    string `json:"public_address"`
}

type SyncConfig struct {
	EnableAutoSync    bool `json:"enable_auto_sync"`
	SyncIntervalMins  int  `json:"sync_interval_mins"`
	MaxBlocksPerSync  int  `json:"max_blocks_per_sync"`
	EnableCompression bool `json:"enable_compression"`
}

var (
		configMutex sync.RWMutex
		globalConfig *NodeConfig
		BIGchainSeedNodes []string
	)

// GetDefaultConfig returns default configuration
func GetDefaultConfig() *NodeConfig {
	return &NodeConfig{
			Network: NetworkConfig{
				SeedNodes: []string{
					"45.191.11.128:8333", // IP do Usuário
				},
				Environment: "development",
				SeedIP:      "45.191.11.128",
				BackupSeeds: []string{},
			},
			SeedNode: SeedNodeConfig{
				IsSeedNode:       false, // Regular node by default
				SeedNodeName:     "BIGchain Official Seed - User PC",
				MaxConnections:   200,
				HistoryRetention: 365,
				PublicAddress:    "45.191.11.128:8333",
			},
		Mining: MiningConfig{
			Enabled:            true,
			Difficulty:         0,           // Dificuldade 0 para mineração cooperativa
			BlockTime:          10,          // 10 minutos
			Reward:             1.0,         // 1 BIG por bloco
			CooperativeMode:    true,        // Modo cooperativo ativado
			RewardDistribution: "all_nodes", // Todos recebem
		},
			P2P: P2PConfig{
				Port:           8333,
				BandwidthPort:  8334,
				APIPort:        8080,
				MaxConnections: 200, // Aumentado para suportar mais conexões como auto-seed
			},
		Sync: SyncConfig{
			EnableAutoSync:   true,
			SyncIntervalMins: 5,
			MaxBlocksPerSync: 1000,
			EnableCompression: true,
		},
	}
}

// GetDefaultSeedConfig returns default seed node configuration
func GetDefaultSeedConfig() *NodeConfig {
	config := GetDefaultConfig()
	config.SeedNode.IsSeedNode = true
	config.SeedNode.PublicAddress = "localhost:8333"
	return config
}

// LoadConfig loads configuration from file or creates default
func LoadConfig() (*NodeConfig, error) {
	configMutex.Lock()
	defer configMutex.Unlock()

	configFile := "bigchain_config.json"

	// If file doesn't exist, create default
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		config := GetDefaultConfig()
		if err := saveConfigLocked(config); err != nil {
			return nil, fmt.Errorf("failed to create default config: %v", err)
		}
		fmt.Println("📄 Created default configuration file: bigchain_config.json")
		fmt.Println("🤝 Cooperative mining mode enabled - All nodes receive rewards!")
		fmt.Println("⚠️  Please update the DigitalOcean Seed Node IP before running in production!")
		globalConfig = config
		return config, nil
	}

	// Load existing file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config NodeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Validate loaded config
	if issues := config.ValidateForProduction(); len(issues) > 0 && config.Network.Environment == "production" {
		fmt.Println("⚠️  Configuration issues found:")
		for _, issue := range issues {
			fmt.Printf("  - %s\n", issue)
		}
	}

	globalConfig = &config
	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *NodeConfig) error {
	configMutex.Lock()
	defer configMutex.Unlock()

	return saveConfigLocked(config)
}

// saveConfigLocked is internal locked version
func saveConfigLocked(config *NodeConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Write with restricted permissions (0600)
	if err := os.WriteFile("bigchain_config.json", data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	fmt.Printf("✅ Configuration saved to: bigchain_config.json\n")
	return nil
}

// UpdateSeedNodeIP updates DigitalOcean Seed Node IP in config
func (config *NodeConfig) UpdateSeedNodeIP(newIP string) error {
	if newIP == "" {
		return fmt.Errorf("Seed Node IP cannot be empty")
	}

	// Validate IP format
	if !isValidIPAddress(newIP) {
		return fmt.Errorf("invalid IP address format: %s", newIP)
	}

	oldIP := config.Network.SeedIP
	config.Network.SeedIP = newIP
	config.SeedNode.PublicAddress = fmt.Sprintf("%s:8333", newIP)

	// Update seed nodes list - replace old IP or placeholders
	for i, seed := range config.Network.SeedNodes {
		// Check if this is the old seed IP
		if oldIP != "" && seed == fmt.Sprintf("%s:8333", oldIP) {
			config.Network.SeedNodes[i] = fmt.Sprintf("%s:8333", newIP)
			continue
		}
		// Check for common placeholders
		if seed == "SEU_IP_SEED:8333" || 
		   seed == "localhost:8333" || 
		   strings.Contains(seed, "placeholder") {
			config.Network.SeedNodes[i] = fmt.Sprintf("%s:8333", newIP)
		}
	}

	fmt.Printf("✅ DigitalOcean Seed Node IP updated: %s → %s\n", oldIP, newIP)
	return SaveConfig(config)
}

// ValidateForProduction validates config is production-ready
func (config *NodeConfig) ValidateForProduction() []string {
	var issues []string

	// Check Seed Node IP
	if config.Network.SeedIP == "" || config.Network.SeedIP == "SEU_IP_SEED" {
		issues = append(issues, "❌ DigitalOcean Seed Node IP not configured")
	} else if !isValidIPAddress(config.Network.SeedIP) {
		issues = append(issues, fmt.Sprintf("❌ Invalid Seed Node IP format: %s", config.Network.SeedIP))
	}

	// Check Seed node address
	if config.SeedNode.PublicAddress == "" || config.SeedNode.PublicAddress == "SEU_IP_SEED:8333" {
		issues = append(issues, "❌ Seed node public address not configured")
	}

	// Check seed nodes list
	for _, seed := range config.Network.SeedNodes {
		if seed == "SEU_IP_SEED:8333" {
			issues = append(issues, "❌ Seed node list contains placeholder IP")
		}
		if !isValidAddress(seed) {
			issues = append(issues, fmt.Sprintf("❌ Invalid seed node address: %s", seed))
		}
	}

	// Check mining config - MODIFICADO: permite difficulty 0 em modo cooperativo
	if config.Mining.CooperativeMode {
		if config.Mining.Difficulty != 0 {
			issues = append(issues, "⚠️  Cooperative mode works best with difficulty 0")
		}
	} else {
		if config.Mining.Difficulty < 1 {
			issues = append(issues, "❌ Mining difficulty must be at least 1 in competitive mode")
		}
	}

	if config.Mining.Reward <= 0 {
		issues = append(issues, "❌ Mining reward must be greater than 0")
	}

	if config.Mining.BlockTime < 1 {
		issues = append(issues, "❌ Block time must be at least 1 minute")
	}

	// Check P2P ports
	if config.P2P.Port < 1024 || config.P2P.Port > 65535 {
		issues = append(issues, fmt.Sprintf("❌ Invalid P2P port: %d", config.P2P.Port))
	}

	return issues
}

// ApplyConfig applies configuration settings globally
func ApplyConfig(config *NodeConfig) {
	if config == nil {
		fmt.Println("❌ Cannot apply nil config")
		return
	}

	configMutex.Lock()
	globalConfig = config
	BIGchainSeedNodes = config.Network.SeedNodes
	configMutex.Unlock()

	fmt.Printf("🔧 Configuration loaded: %s environment\n", config.Network.Environment)
	fmt.Printf("🌱 Seed nodes: %d configured\n", len(config.Network.SeedNodes))
	
	// Informações sobre o modo de mineração
	if config.Mining.CooperativeMode {
		fmt.Printf("🤝 Mining Mode: COOPERATIVE (all nodes receive rewards)\n")
		fmt.Printf("⏱️  Block Time: %d minutes (automatic block generation)\n", config.Mining.BlockTime)
		fmt.Printf("💰 Reward per block: %.2f BIG distributed to all active nodes\n", config.Mining.Reward)
	} else {
		fmt.Printf("⛏️  Mining Mode: COMPETITIVE\n")
		fmt.Printf("⛏️  Mining: enabled=%t, difficulty=%d, reward=%.2f BIG\n",
			config.Mining.Enabled, config.Mining.Difficulty, config.Mining.Reward)
	}

	if config.SeedNode.IsSeedNode {
		fmt.Printf("🌱 Running as SEED NODE: %s (DigitalOcean)\n", config.SeedNode.SeedNodeName)
	}
}

// GetGlobalConfig returns the current global config
func GetGlobalConfig() *NodeConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()

	return globalConfig
}

// SetupDigitalOceanSeed helper function to configure DigitalOcean Seed Node IP
func SetupDigitalOceanSeed(seedIP string) error {
	if seedIP == "" {
		return fmt.Errorf("Seed Node IP cannot be empty")
	}

	config, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := config.UpdateSeedNodeIP(seedIP); err != nil {
		return err
	}

	fmt.Printf("✅ DigitalOcean Seed Node IP configured: %s\n", seedIP)
	fmt.Println("🚀 Configuration ready for production deployment!")

	return nil
}

// ValidateConfig performs comprehensive validation
func ValidateConfig(config *NodeConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Validate network
	if len(config.Network.SeedNodes) == 0 {
		return fmt.Errorf("no seed nodes configured")
	}

	// Validate mining - MODIFICADO: permite difficulty 0 em modo cooperativo
	if config.Mining.CooperativeMode {
		if config.Mining.Difficulty != 0 {
			fmt.Println("⚠️  Warning: Cooperative mode recommended with difficulty 0")
		}
	} else {
		if config.Mining.Difficulty < 1 {
			return fmt.Errorf("mining difficulty must be at least 1 in competitive mode")
		}
	}

	if config.Mining.Reward <= 0 {
		return fmt.Errorf("mining reward must be positive")
	}

	if config.Mining.BlockTime < 1 {
		return fmt.Errorf("block time must be at least 1 minute")
	}

	// Validate P2P
	if config.P2P.Port < 1024 || config.P2P.Port > 65535 {
		return fmt.Errorf("invalid P2P port: %d", config.P2P.Port)
	}

	if config.P2P.MaxConnections < 1 {
		return fmt.Errorf("max connections must be at least 1")
	}

	// Validate sync
	if config.Sync.SyncIntervalMins < 1 {
		return fmt.Errorf("sync interval must be at least 1 minute")
	}

	return nil
}

// PrintConfig prints configuration details
func PrintConfig(config *NodeConfig) {
	if config == nil {
		fmt.Println("❌ Config is nil")
		return
	}

	fmt.Println()
	fmt.Println("=== BIGchain Configuration (DigitalOcean) ===")
	fmt.Printf("Environment: %s\n", config.Network.Environment)
	fmt.Printf("Seed Node IP: %s\n", config.Network.SeedIP)
	fmt.Println()

	fmt.Println("P2P Network:")
	fmt.Printf("  Port: %d\n", config.P2P.Port)
	fmt.Printf("  Bandwidth Port: %d\n", config.P2P.BandwidthPort)
	fmt.Printf("  API Port: %d\n", config.P2P.APIPort)
	fmt.Printf("  Max Connections: %d\n", config.P2P.MaxConnections)
	fmt.Println()

	fmt.Println("Mining Configuration:")
	fmt.Printf("  Enabled: %t\n", config.Mining.Enabled)
	fmt.Printf("  Mode: %s\n", getMiningModeString(config))
	fmt.Printf("  Difficulty: %d\n", config.Mining.Difficulty)
	fmt.Printf("  Block Time: %d minutes\n", config.Mining.BlockTime)
	fmt.Printf("  Reward: %.2f BIG\n", config.Mining.Reward)
	
	if config.Mining.CooperativeMode {
		fmt.Printf("  Distribution: %s\n", config.Mining.RewardDistribution)
		fmt.Println("  ℹ️  All active nodes receive rewards every 10 minutes")
	}
	fmt.Println()

	fmt.Println("Seed Nodes:")
	for i, seed := range config.Network.SeedNodes {
		fmt.Printf("  %d. %s\n", i+1, seed)
	}
	fmt.Println()

	if config.SeedNode.IsSeedNode {
		fmt.Printf("This node is configured as SEED NODE (DigitalOcean): %s\n", config.SeedNode.SeedNodeName)
		fmt.Printf("Max Connections: %d\n", config.SeedNode.MaxConnections)
	}
	fmt.Println()
}

// getMiningModeString returns a string describing the mining mode
func getMiningModeString(config *NodeConfig) string {
	if config.Mining.CooperativeMode {
		return "COOPERATIVE (all nodes receive rewards)"
	}
	return "COMPETITIVE (only miner receives rewards)"
}

// Helper functions

// isValidIPAddress validates IP address format
func isValidIPAddress(ip string) bool {
	// Simple IPv4 validation
	ipv4Regex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if !ipv4Regex.MatchString(ip) {
		return false
	}

	// Validate each octet
	parts := strings.Split(ip, ".")
	for _, part := range parts {
		var num int
		if _, err := fmt.Sscanf(part, "%d", &num); err != nil {
			return false
		}
		if num < 0 || num > 255 {
			return false
		}
	}

	return true
}

// isValidAddress validates host:port format
func isValidAddress(addr string) bool {
	if addr == "" {
		return false
	}

	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return false
	}

	// Check if first part is valid IP or hostname
	ip := parts[0]
	if !isValidIPAddress(ip) && !isValidHostname(ip) {
		return false
	}

	// Check if second part is valid port
	var port int
	if _, err := fmt.Sscanf(parts[1], "%d", &port); err != nil {
		return false
	}

	return port > 0 && port <= 65535
}

// isValidHostname validates hostname format
func isValidHostname(hostname string) bool {
	if hostname == "localhost" {
		return true
	}

	hostnameRegex := regexp.MustCompile(`^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)*[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	return hostnameRegex.MatchString(hostname)
}