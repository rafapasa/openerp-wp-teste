package main

import (
	"flag"
	"fmt"
	"os"
	"rafapasa/openerp-wp-teste/internal/client"
	"rafapasa/openerp-wp-teste/internal/config"
	"rafapasa/openerp-wp-teste/internal/dto"
	"strings"
	"sync"
	"time"
)

type estatisticas struct {
	mu        sync.Mutex
	Total     int
	Ok        int
	Falhas    int
	ErrosHTTP int
	PorErro   map[int]int
}

func (e *estatisticas) registrar(resultado string, httpStatus int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Total++
	switch resultado {
	case "ok":
		e.Ok++
	case "falha":
		e.Falhas++
	case "erro":
		e.ErrosHTTP++
		if httpStatus > 0 {
			e.PorErro[httpStatus]++
		}
	}
}

func main() {
	cfgPath := flag.String("config", "./config.json", "arquivo de configuração")
	scenarioFiltro := flag.String("scenario", "", "executa só um cenário (id)")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Printf("❌ config: %v\n", err)
		os.Exit(1)
	}

	scenarios, err := config.LoadScenarios(cfg.ScenariosDir)
	if err != nil {
		fmt.Printf("❌ cenários: %v\n", err)
		os.Exit(1)
	}

	if *scenarioFiltro != "" {
		var filtrados []dto.Scenario
		for _, s := range scenarios {
			if s.ID == *scenarioFiltro {
				filtrados = append(filtrados, s)
			}
		}
		scenarios = filtrados
	}

	if len(scenarios) == 0 {
		fmt.Println("❌ nenhum cenário encontrado")
		os.Exit(1)
	}

	fmt.Printf("🧪 Runner WhatsApp — %d cenários, %d threads, server=%s\n\n",
		len(scenarios), cfg.Threads, cfg.ServerURL)

	stats := &estatisticas{PorErro: make(map[int]int)}
	inicio := time.Now()

	fila := make(chan dto.Scenario, len(scenarios))
	for _, s := range scenarios {
		fila <- s
	}
	close(fila)

	var wg sync.WaitGroup
	for i := 0; i < cfg.Threads; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			cli := client.New(cfg)
			for s := range fila {
				executarScenario(cli, cfg, s, stats, worker)
			}
		}(i)
	}

	wg.Wait()
	duracao := time.Since(inicio)

	fmt.Println()
	fmt.Println("══════════════════════════════════════════════")
	fmt.Println("  RELATÓRIO FINAL")
	fmt.Println("══════════════════════════════════════════════")
	fmt.Printf("  Total de mensagens enviadas: %d\n", stats.Total)
	fmt.Printf("  ✅ Respostas esperadas:      %d\n", stats.Ok)
	fmt.Printf("  ❌ Respostas fora do padrão: %d\n", stats.Falhas)
	fmt.Printf("  ⚠️  Erros HTTP:               %d\n", stats.ErrosHTTP)
	for code, n := range stats.PorErro {
		fmt.Printf("       HTTP %d: %d\n", code, n)
	}
	fmt.Printf("  ⏱  Duração total:            %v\n", duracao.Round(time.Millisecond))
	fmt.Println("══════════════════════════════════════════════")
}

func executarScenario(cli *client.Client, cfg *dto.Config, s dto.Scenario, stats *estatisticas, worker int) {
	fmt.Printf("[w%d] ▶ %s — %s\n", worker, s.ID, s.Descricao)

	_ = cli.LimparInbox(s.ID)

	for i, msg := range s.Mensagens {
		if i > 0 {
			time.Sleep(1500 * time.Millisecond)
		}

		fmt.Printf("[w%d]   → %.60s\n", worker, msg.Texto)

		status, err := cli.EnviarWebhook(s.ID, msg.Texto)
		if err != nil || status >= 400 {
			fmt.Printf("[w%d]   ⚠️  HTTP %d err=%v\n", worker, status, err)
			stats.registrar("erro", status)
			continue
		}

		if !msg.EsperaResposta {
			stats.registrar("ok", status)
			continue
		}

		antes := contarInbox(cli, s.ID)
		resposta, ok := aguardarResposta(cli, s.ID, cfg, antes)
		if !ok {
			fmt.Printf("[w%d]   ⏱ timeout esperando resposta\n", worker)
			stats.registrar("falha", 0)
			continue
		}

		fmt.Printf("[w%d]   ← %.80s\n", worker, resposta)

		if len(msg.RespostaEsperadaContem) == 0 {
			stats.registrar("ok", status)
			continue
		}

		if contemAlgum(resposta, msg.RespostaEsperadaContem) {
			stats.registrar("ok", status)
		} else {
			fmt.Printf("[w%d]   ❌ esperava: %v\n", worker, msg.RespostaEsperadaContem)
			stats.registrar("falha", status)
		}
	}
}

func aguardarResposta(cli *client.Client, scenarioID string, cfg *dto.Config, antes int) (string, bool) {
	deadline := time.Now().Add(time.Duration(cfg.TimeoutSeconds) * time.Second)
	intervalo := time.Duration(cfg.PollIntervalMs) * time.Millisecond

	for time.Now().Before(deadline) {
		time.Sleep(intervalo)
		inbox, err := cli.BuscarInbox(scenarioID)
		if err != nil || inbox == nil {
			continue
		}
		if inbox.Total > antes {
			return inbox.Mensagens[len(inbox.Mensagens)-1].Texto, true
		}
	}
	return "", false
}

func contarInbox(cli *client.Client, scenarioID string) int {
	inbox, err := cli.BuscarInbox(scenarioID)
	if err != nil || inbox == nil {
		return 0
	}
	return inbox.Total
}

func contemAlgum(texto string, alvos []string) bool {
	lower := strings.ToLower(texto)
	for _, a := range alvos {
		if strings.Contains(lower, strings.ToLower(a)) {
			return true
		}
	}
	return false
}
