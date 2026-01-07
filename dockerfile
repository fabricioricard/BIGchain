# Dockerfile para BIGchain
FROM golang:1.21-alpine AS builder

# Instalar dependências de build
RUN apk add --no-cache git ca-certificates tzdata

# Configurar diretório de trabalho
WORKDIR /app

# Copiar arquivos de dependência
COPY go.mod go.sum ./

# Baixar dependências
RUN go mod download

# Copiar código fonte
COPY . .

# Build otimizado
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bigchain .

# Imagem final mínima
FROM alpine:latest

# Instalar certificados CA
RUN apk --no-cache add ca-certificates

# Criar usuário não-root
RUN adduser -D -s /bin/sh bigchain

# Configurar diretório
WORKDIR /home/bigchain

# Copiar binário
COPY --from=builder /app/bigchain .

# Ajustar permissões
RUN chown bigchain:bigchain bigchain

# Mudar para usuário não-root
USER bigchain

# Portas expostas
EXPOSE 8333 8334 8080

# Comando padrão
CMD ["./bigchain"]