# openerp-wp-teste

Simulador de conversas WhatsApp para testar o mcp-server-openerp.

## Componentes

- **whatsapp-mock** — finge ser a API do WhatsApp Cloud. Recebe as mensagens que o bot "manda pro cliente" e guarda por cenário.
- **runner** — lê os cenários JSON e dispara as mensagens em paralelo, comparando com a resposta esperada.

## Como rodar

### 1. Sobe o mock

```bash
docker compose -f docker-compose.whatsapp-tester.yml up -d --build
