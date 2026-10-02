# ============================================================
# Makefile para openerp-wp-teste
# Simulador de conversas WhatsApp
# ============================================================

BIN_DIR    := ./bin
RUNNER     := $(BIN_DIR)/runner
MOCK       := $(BIN_DIR)/mock
CONFIG     ?= ./config.json
SCENARIOS  := ./scenarios

GREEN = \033[0;32m
YELLOW = \033[0;33m
BLUE = \033[0;34m
NC = \033[0m

# ============================================================
# AJUDA
# ============================================================

help: ## Mostra esta ajuda
	@echo "$(BLUE)📋 Comandos disponíveis$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

# ============================================================
# BUILD
# ============================================================

build: ## Compila runner e mock
	@mkdir -p $(BIN_DIR)
	go build -o $(RUNNER) ./cmd/runner
	go build -o $(MOCK) ./cmd/mock
	@echo "$(GREEN)✅ binários em $(BIN_DIR)/$(NC)"

# ============================================================
# RUNNER — roda todos os cenários
# ============================================================

run: build ## Roda todos os cenários: make run
	@echo "$(BLUE)🧪 Rodando todos os cenários$(NC)"
	@$(RUNNER) -config $(CONFIG)

# ============================================================
# MOCK — servidor fake do WhatsApp
# ============================================================

mock-up: build ## Sobe o mock local (foreground, porta 9000)
	@echo "$(BLUE)🟢 Subindo mock em :9000$(NC)"
	$(MOCK)

mock-docker-up: ## Sobe o mock via docker compose
	docker compose -f docker-compose.whatsapp-tester.yml up -d --build
	@echo "$(GREEN)✅ whatsapp-mock no ar em http://localhost:9000$(NC)"

mock-docker-down: ## Derruba o mock
	docker compose -f docker-compose.whatsapp-tester.yml down
	@echo "$(GREEN)✅ whatsapp-mock parado$(NC)"

mock-health: ## Checa se o mock está respondendo
	@curl -s http://localhost:9000/health || echo "mock não está no ar"

mock-inbox: ## Mostra o inbox global
	@curl -s "http://localhost:9000/inbox?scenario_id=_global" | python3 -m json.tool 2>/dev/null || \
		curl -s "http://localhost:9000/inbox?scenario_id=_global"

mock-clean: ## Limpa o inbox global
	@curl -s -X DELETE "http://localhost:9000/inbox?scenario_id=_global" && echo ""

# ============================================================
# DEV / LIMPEZA
# ============================================================

tidy: ## go mod tidy
	go mod tidy

vet: ## go vet
	go vet ./...

clean: ## Limpa binários
	rm -rf $(BIN_DIR)
	@echo "$(GREEN)✅ limpo$(NC)"

.PHONY: help build run mock-up mock-docker-up mock-docker-down \
        mock-health mock-inbox mock-clean tidy vet clean

.DEFAULT_GOAL := help
