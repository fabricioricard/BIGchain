# 🌐 BIGchain - BIGFOOT Connect

**Real P2P Blockchain with Cooperative Mining**

BIGchain is a decentralized blockchain where all active nodes automatically receive rewards every 10 minutes just by staying online.

---

## 📋 Table of Contents

- [Requirements](#requirements)
- [Regular Node (Users)](#-regular-node-users)
- [Seed Node (Servers)](#-seed-node-servers)
- [Useful Commands](#-useful-commands)
- [FAQ](#-faq)

---

## 📦 Requirements

### All Nodes:
- **Go 1.21+** installed ([Download](https://go.dev/dl/))
- **Stable internet connection**
- **Open ports:**
  - `8333` (P2P)
  - `8080` (REST API - Seed Nodes only)

### Check installation:
```bash
go version
```

---

## 💻 Regular Node (Users)

### What is it?
A regular node connects to the BIGchain network and receives rewards automatically just by staying online.

### 🎯 How to run:

#### Windows:

1. **Clone the repository:**
```powershell
git clone https://github.com/your-username/BIGchain.git
cd BIGchain
```

2. **Compile:**
```powershell
go build -o bigchain.exe
```

3. **Run:**
```powershell
.\bigchain.exe
```

#### Linux/Mac:

1. **Clone the repository:**
```bash
git clone https://github.com/your-username/BIGchain.git
cd BIGchain
```

2. **Compile:**
```bash
go build -o bigchain
chmod +x bigchain
```

3. **Run:**
```bash
./bigchain
```

### ✅ What happens when starting:

```
=== BIGchain - BIGFOOT Connect ===
Real P2P Blockchain

🔄 Downloading latest blockchain from seed node...
⚠️  This is required to stay synchronized with the network
📥 Trying to download from: 45.55.251.149:8080
✅ Blockchain downloaded: 5 blocks, 3.00 BIG supply
🌐 P2P Node started on port 8333
✅ Connected to peer: 45.55.251.149:8333
🤝 Miner joined cooperative: bigxxxxx...
🚀 BIGchain node is now running!
```

### 💰 How to earn BIG tokens:

- **Just keep your node running!**
- Every 10 minutes, the seed node mines a block
- **All active nodes share 1 BIG** equally
- Example: 10 nodes online = 0.1 BIG for each

### 📝 Available commands:

```
BIGchain> status      # View blockchain status
BIGchain> balance     # View your balance
BIGchain> send        # Send BIG tokens
BIGchain> peers       # View connected peers
BIGchain> wallet      # View wallet info
BIGchain> save        # Save blockchain manually
BIGchain> quit        # Exit
```

---

## 🌱 Seed Node (Servers)

### What is it?
A seed node is a server that:
- ✅ Mines blocks every 10 minutes
- ✅ Distributes rewards to all active nodes
- ✅ Serves the blockchain via REST API
- ✅ Keeps the network synchronized

### ⚠️ Additional requirements:
- Server with **fixed public IP**
- **24/7** availability
- **2GB+ RAM** recommended
- **10GB+ disk** recommended

---

### 🔧 Configuration:

#### 1. Edit the configuration file

Open `bigchain_config.json` and configure:

```json
{
  "seed_node": {
    "is_seed_node": true,
    "seed_node_name": "My Seed Node",
    "public_address": "YOUR_PUBLIC_IP:8333",
    "max_connections": 100,
    "history_retention": 30
  },
  "p2p": {
    "port": 8333,
    "enable_upnp": true
  }
}
```

**⚠️ IMPORTANT:** Replace `YOUR_PUBLIC_IP` with your server's public IP!

To find your IP:
```bash
curl ifconfig.me
```

---

#### 2. Add your seed to the list (so others can find it)

Edit `seed_main.go`, find the `getOtherSeedsList()` function and add:

```go
func getOtherSeedsList() []string {
    return []string{
        "45.55.251.149:8080",        // Official seed
        "YOUR_PUBLIC_IP:8080",       // Your seed (add here)
        // Other future seeds...
    }
}
```

---

### 🚀 How to run:

#### Linux (Recommended - with systemd):

1. **Compile:**
```bash
cd /root/BIGchain  # or your directory
go build -o bigchain
chmod +x bigchain
```

2. **Create systemd service:**

Create file `/etc/systemd/system/bigchain-seed.service`:

```ini
[Unit]
Description=BIGchain Seed Node
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/root/BIGchain
ExecStart=/root/BIGchain/bigchain --seed --daemon
Restart=always
RestartSec=10
StandardOutput=append:/root/BIGchain/bigchain.log
StandardError=append:/root/BIGchain/bigchain.log

[Install]
WantedBy=multi-user.target
```

3. **Start the service:**
```bash
sudo systemctl daemon-reload
sudo systemctl enable bigchain-seed
sudo systemctl start bigchain-seed
```

4. **Check status:**
```bash
sudo systemctl status bigchain-seed
sudo journalctl -u bigchain-seed -f
```

#### Run manually (testing):

```bash
./bigchain --seed
```

---

### ✅ What happens when starting a Seed Node:

```
=== BIGchain SEED NODE ===
🔄 Attempting to sync with other seed nodes...
📥 Trying to sync from: 45.55.251.149:8080
✅ Synced from other seed: 10 blocks, 8.00 BIG
🌐 P2P Node started on port 8333
🌱 Initializing as SEED NODE: My Seed Node
📡 Public address: 123.45.67.89:8333
🌐 Seed node API started on port 8080
🤝 Miner joined cooperative: bigxxxxx...
🚀 Decentralized cooperative mining started!
🌱 SEED NODE ACTIVE
```

### 🔥 Firewall Configuration:

**Open required ports:**

```bash
# UFW (Ubuntu/Debian)
sudo ufw allow 8333/tcp
sudo ufw allow 8080/tcp

# Firewalld (CentOS/RHEL)
sudo firewall-cmd --permanent --add-port=8333/tcp
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload

# iptables
sudo iptables -A INPUT -p tcp --dport 8333 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

---

## 🛠️ Useful Commands

### Regular Node:

```bash
# View logs (Windows)
Get-Content blockchain.log -Tail 50

# View logs (Linux)
tail -f bigchain.log

# View local blockchain
cat blockchain.json
```

### Seed Node:

```bash
# View service status
sudo systemctl status bigchain-seed

# View logs in real-time
sudo journalctl -u bigchain-seed -f

# Restart service
sudo systemctl restart bigchain-seed

# Stop service
sudo systemctl stop bigchain-seed

# Test REST API
curl http://localhost:8080/api/network/status
curl http://localhost:8080/api/blockchain/full
curl http://localhost:8080/api/seed/info
```

---

## 🔍 REST API (Seed Nodes)

### Available endpoints:

| Endpoint | Description |
|----------|-----------|
| `GET /api/network/status` | Network status |
| `GET /api/blockchain/full` | Complete blockchain |
| `GET /api/blockchain/range?start=0&end=10` | Block range |
| `GET /api/seed/info` | Seed information |

### Examples:

```bash
# Network status
curl http://45.55.251.149:8080/api/network/status

# Download complete blockchain
curl http://45.55.251.149:8080/api/blockchain/full > blockchain_download.json

# Specific blocks
curl http://45.55.251.149:8080/api/blockchain/range?start=0&end=5
```

---

## ❓ FAQ

### How does cooperative mining work?

- **Seed node** mines blocks every 10 minutes
- **All active nodes** share the 1 BIG reward
- **You don't need to do anything**, just stay online
- **More nodes = smaller individual reward**

### What happens if my node goes offline?

- You **stop receiving rewards**
- Your balance **is not lost**
- When back online, you'll receive rewards normally again

### How do I know if I'm receiving rewards?

```bash
BIGchain> balance
Your balance: 2.50 BIG

BIGchain> status
Active Nodes: 5
Reward per Node: 0.20000000 BIG (every 10min)
```

### Can I run multiple nodes?

- **Yes**, each node needs a **different wallet**
- Each node will receive its share of rewards
- **Not recommended** to run on same machine (port conflicts)

### How do I send BIG to another address?

```bash
BIGchain> send
Enter recipient address: big[recipient's address]
Enter amount: 0.5
✅ Transaction sent: 0.50 BIG to bigxxxxx...
```

### What if I can't download the blockchain?

**Check:**
1. Internet connection
2. Firewall not blocking port 8080
3. Seed node is online: `curl http://45.55.251.149:8080/api/seed/info`

If problem persists, contact support.

### How long does synchronization take?

- Current blockchain: **~1-5 seconds**
- Grows over time, but always fast
- HTTP REST API download is optimized

### Can I change the P2P port?

Yes, edit `bigchain_config.json`:

```json
{
  "p2p": {
    "port": 9333
  }
}
```

---

## 🔐 Security

### ⚠️ IMPORTANT:

1. **Never share your private key!**
2. **Backup your wallet** (`wallet_*.json`)
3. **Store in a safe place**
4. **Seed nodes: use firewall and SSL/TLS**

### Wallet Backup:

```bash
# Copy wallet to safe location
cp wallet_*.json ~/backup/
```

---

## 📊 BIGchain Economics

| Item | Value |
|------|-------|
| **Block reward** | 1.00 BIG |
| **Frequency** | 10 minutes |
| **Max supply** | 21,000,000 BIG |
| **Halving** | Every 210,000 blocks |
| **Difficulty** | 0 (no heavy PoW) |

---

## 🤝 Contributing

Pull requests are welcome! For major changes, please open an issue first.

---

## 📞 Support

- **Issues:** [GitHub Issues](https://github.com/your-username/BIGchain/issues)
- **Discord:** https://discord.gg/mkfmncN5Sa
- **Email:** contact@bigfootconnect.tech

---

## 📜 License

MIT License - see [LICENSE](LICENSE) for details.

---

## 🎉 Get Started Now!

```bash
# Clone and run in 3 commands:
git clone https://github.com/your-username/BIGchain.git
cd BIGchain
go build -o bigchain.exe && .\bigchain.exe
```

**Welcome to the BIGchain network!** 🚀
