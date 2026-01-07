# Makefile para BIGchain
.PHONY: build run clean test format install-deps build-all docker

# Configurações
APP_NAME = bigchain
VERSION = 1.0.0
BUILD_DIR = build
DOCKER_IMAGE = bigchain:$(VERSION)

# Build principal
build:
	@echo "🔨 Building BIGchain..."
	go build -o $(APP_NAME) .
	@echo "✅ Build complete!"

# Executar diretamente
run:
	@echo "🚀 Starting BIGchain..."
	go run .

# Build para produção com otimizações
build-prod:
	@echo "🔨 Building BIGchain for production..."
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(APP_NAME) .
	@echo "✅ Production build complete!"

# Build para múltiplas plataformas
build-all:
	@echo "🔨 Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	@echo "Building for Linux AMD64..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 .
	
	@echo "Building for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 .
	
	@echo "Building for Windows AMD64..."
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .
	
	@echo "Building for macOS AMD64..."
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .
	
	@echo "Building for macOS ARM64 (M1/M2)..."
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .
	
	@echo "✅ Multi-platform build complete!"
	@ls -la $(BUILD_DIR)/

# Limpeza
clean:
	@echo "🧹 Cleaning..."
	rm -f $(APP_NAME)
	rm -f $(APP_NAME).exe
	rm -rf $(BUILD_DIR)
	rm -f blockchain.json
	rm -f *.log
	@echo "✅ Cleaned!"

# Executar testes
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Instalar dependências
install-deps:
	@echo "📦 Installing dependencies..."
	go mod tidy
	go mod download
	@echo "✅ Dependencies installed!"

# Formatação de código
format:
	@echo "📐 Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted!"

# Verificação de código (linting)
lint:
	@echo "🔍 Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run
	@echo "✅ Lint complete!"

# Build Docker
docker:
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "✅ Docker image built: $(DOCKER_IMAGE)"

# Executar com Docker
docker-run:
	@echo "🐳 Running BIGchain in Docker..."
	docker run -it --rm -p 8333:8333 -p 8334:8334 -p 8080:8080 $(DOCKER_IMAGE)

# Deploy para produção
deploy: build-prod
	@echo "🚀 Deploying BIGchain..."
	@echo "Copying binary to /usr/local/bin/..."
	sudo cp $(APP_NAME) /usr/local/bin/
	@echo "✅ Deployed!"

# Benchmark
benchmark:
	@echo "⚡ Running benchmarks..."
	go test -bench=. -benchmem ./...

# Verificação de segurança
security:
	@echo "🔒 Running security checks..."
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...
	@echo "✅ Security check complete!"

# Gerar documentação
docs:
	@echo "📚 Generating documentation..."
	@which godoc > /dev/null || (echo "Installing godoc..." && go install golang.org/x/tools/cmd/godoc@latest)
	godoc -http=:6060
	@echo "📖 Documentation server running at http://localhost:6060"

# Setup completo para desenvolvimento
setup: install-deps format lint test
	@echo "🎉 Setup complete! Ready for development."

# Informações do sistema
info:
	@echo "📊 BIGchain Build Information"
	@echo "=============================="
	@echo "App Name: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Go Version: $(shell go version)"
	@echo "Build Time: $(shell date)"
	@echo "Git Commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

# Help
help:
	@echo "🚀 BIGchain Makefile Commands"
	@echo "============================="
	@echo "build        - Build BIGchain binary"
	@echo "run          - Run BIGchain directly"
	@echo "build-prod   - Build optimized for production"
	@echo "build-all    - Build for all platforms"
	@echo "clean        - Clean build artifacts"
	@echo "test         - Run tests"
	@echo "format       - Format Go code"
	@echo "lint         - Run code linter"
	@echo "docker       - Build Docker image"
	@echo "docker-run   - Run in Docker"
	@echo "deploy       - Deploy to production"
	@echo "benchmark    - Run performance benchmarks"
	@echo "security     - Run security checks"
	@echo "docs         - Start documentation server"
	@echo "setup        - Complete development setup"
	@echo "info         - Show build information"
	@echo "help         - Show this help"