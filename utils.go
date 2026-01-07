package main

import (
	"fmt"
)

func shortenAddress(address string) string {
	if len(address) > 12 {
		return address[:12] + "..."
	}
	return address
}

func formatTimestamp(ts int64) string {
	return fmt.Sprintf("%d", ts)
}

func (bc *Blockchain) GetActiveMiners() []string {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	var active []string
	for addr, miner := range bc.ActiveMiners {
		if miner.IsActive {
			active = append(active, addr)
		}
	}
	return active
}

func (bc *Blockchain) getCurrentMiningReward() float64 {
	return bc.MiningReward
}

func (bc *Blockchain) ValidateChain() bool {
	return true // Simplificado para PoR
}
