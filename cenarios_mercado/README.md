
# Cenários de Teste - Segmento Mercado (30 cenários)

Implementação do segmento MERCADO está completa - suite de caixa-preta para runner Go.

## Estrutura do runner esperada
Cada arquivo JSON é um cenário. Campos:
- id: identificador único (usado para gerar phone determinístico + runSalt)
- mensagens: array de {de: usuario, texto: ..., esperado: {contem: [], nao_contem: [], intent, min_produtos, etc}}
- produtos_envolvidos: SKUs esperados

Seu runner deve:
1. Ler `mensagens[].texto` e mandar via webhook com phone = getFoneDoCenario(id) com runSalt
2. Verificar resposta do bot contém todos `esperado.contem`
3. Garantir `nao_contem` não aparece
4. Respeitar delay 1200ms entre mensagens do mesmo cenário

## Cobertura dos 30 cenários
- 01-05: Básico, busca, preço, lista
- 06: Robustez typo (arros, fejao, macarao)
- 07-08: Quantidade, carrinho add/remove
- 09-14: Comparação, receita, café da manhã, snack
- 15-16: Stress / atacado / 15 itens
- 17: Fallback sem produto
- 18: Linguagem coloquial BR
- 19: Troca de ideia
- 20, 30: Checkout completo E2E
- 21: Produto indisponível -> sugestão
- 22, 29: Unidade medida confusa (un vs kg) - hortifruti
- 23: Mix categorias 215+216+217
- 24: Dieta fitness
- 25: Busca por SKU
- 26: Duplicidade / atualizar quantidade
- 27: Negação (não quero branco)
- 28: Orçamento limitado R$50

## Como rodar sem tomar 429 (correção da conversa anterior)
No main.go aplique:

var runSalt = time.Now().UnixMilli() % 1000
var phones sync.Map
func getFoneDoCenario(id string) string { ... fmt.Sprintf("55499%03d%05d", runSalt, hash) ... }

E antes da bateria chame DELETE /test/limpar-tudo para limpar rate-limit do Redis.

Rode com threads=1 e delay 300ms entre cenários.

## Produtos disponíveis
100 produtos do arquivo produtos_2026-10-22_142836.json
- cat 215: 40 mercearia
- cat 216: 40 hortifruti
- cat 217: 20 açougue
