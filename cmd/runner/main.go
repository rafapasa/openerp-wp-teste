package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"os"
	"rafapasa/openerp-wp-teste/internal/client"
	"rafapasa/openerp-wp-teste/internal/config"
	"rafapasa/openerp-wp-teste/internal/dto"
	"sort"
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
	Cenarios  []resultadoCenario
}

type resultadoCenario struct {
	Worker int
	ID     string
	Ok     int
	Falha  int
	Erro   int
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

func (e *estatisticas) fechar(r resultadoCenario) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Cenarios = append(e.Cenarios, r)
}

type fluxo struct {
	mu     sync.Mutex
	linhas map[int][]string
}

func (f *fluxo) add(worker int, linha string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.linhas[worker] = append(f.linhas[worker], linha)
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

	fmt.Printf("🧪 Runner WhatsApp — %d cenários, %d threads, server=%s\n", len(scenarios), cfg.Threads, cfg.ServerURL)
	stats := &estatisticas{PorErro: make(map[int]int)}
	log := &fluxo{linhas: make(map[int][]string)}
	inicio := time.Now()
	fila := make(chan dto.Scenario, len(scenarios))
	for _, s := range scenarios {
		fila <- s
	}
	close(fila)

	var wg sync.WaitGroup
	workers := cfg.Threads
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			cli := client.New(cfg)
			for s := range fila {
				executarScenario(cli, cfg, s, stats, log, worker)
			}
		}(i + 1)
	}
	wg.Wait()
	imprimirFluxos(log)
	imprimirResumo(stats, time.Since(inicio))
	if stats.Falhas > 0 || stats.ErrosHTTP > 0 {
		os.Exit(1)
	}
}

func imprimirFluxos(log *fluxo) {
	ids := make([]int, 0, len(log.linhas))
	for id := range log.linhas {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	fmt.Println()
	for _, id := range ids {
		fmt.Printf("════════ [w%d] ════════\n", id)
		for _, linha := range log.linhas[id] {
			fmt.Println(linha)
		}
		fmt.Println()
	}
}

func imprimirResumo(stats *estatisticas, duracao time.Duration) {
	fmt.Println("══════════════════════════════════════════════")
	sort.Slice(stats.Cenarios, func(i, j int) bool {
		if stats.Cenarios[i].Worker == stats.Cenarios[j].Worker {
			return stats.Cenarios[i].ID < stats.Cenarios[j].ID
		}
		return stats.Cenarios[i].Worker < stats.Cenarios[j].Worker
	})
	for _, c := range stats.Cenarios {
		marca := "✅"
		if c.Falha > 0 || c.Erro > 0 {
			marca = "❌"
		}
		fmt.Printf("  %s [w%d] %s  ok=%d falha=%d erro=%d\n", marca, c.Worker, c.ID, c.Ok, c.Falha, c.Erro)
	}
	fmt.Println("──────────────────────────────────────────────")
	fmt.Printf("  ✅ Ok:                       %d\n", stats.Ok)
	fmt.Printf("  ❌ Respostas fora do padrão: %d\n", stats.Falhas)
	fmt.Printf("  ⚠️  Erros HTTP:               %d\n", stats.ErrosHTTP)
	for code, n := range stats.PorErro {
		fmt.Printf("       HTTP %d: %d\n", code, n)
	}
	fmt.Printf("  ⏱  Duração total:            %v\n", duracao.Round(time.Millisecond))
	fmt.Println("══════════════════════════════════════════════")
}

func executarScenario(cli *client.Client, cfg *dto.Config, s dto.Scenario, stats *estatisticas, log *fluxo, worker int) {
	fone := foneDoCenario(s.ID)
	res := resultadoCenario{Worker: worker, ID: s.ID}
	log.add(worker, fmt.Sprintf("▶ %s — %s", s.ID, s.Descricao))
	log.add(worker, fmt.Sprintf("  fone %s", fone))
	_ = cli.LimparInbox(fone)

	for i, msg := range s.Mensagens {
		if i > 0 {
			time.Sleep(1500 * time.Millisecond)
		}
		log.add(worker, fmt.Sprintf("  → %s", msg.Texto))
		antes := contarInbox(cli, fone)
		status, err := cli.EnviarWebhook(s.ID, fone, msg.Texto)
		if err != nil || status >= 400 {
			log.add(worker, fmt.Sprintf("  ⚠️  HTTP %d err=%v", status, err))
			stats.registrar("erro", status)
			res.Erro++
			continue
		}
		if !msg.EsperaResposta {
			stats.registrar("ok", status)
			res.Ok++
			continue
		}
		resposta, ok := aguardarResposta(cli, fone, cfg, antes)
		if !ok {
			log.add(worker, "  ⏱ timeout esperando resposta")
			stats.registrar("falha", 0)
			res.Falha++
			continue
		}
		log.add(worker, fmt.Sprintf("  ← %s", umaLinha(resposta)))
		if len(msg.RespostaEsperadaContem) > 0 && !contemAlgum(resposta, msg.RespostaEsperadaContem) {
			log.add(worker, fmt.Sprintf("  ❌ esperava: %s", strings.Join(msg.RespostaEsperadaContem, " | ")))
			stats.registrar("falha", status)
			res.Falha++
			continue
		}
		if proibido := contemAlgumTexto(resposta, msg.RespostaNaoContem); proibido != "" {
			log.add(worker, fmt.Sprintf("  ❌ não podia conter: %s", proibido))
			stats.registrar("falha", status)
			res.Falha++
			continue
		}
		log.add(worker, "  ✅")
		stats.registrar("ok", status)
		res.Ok++
	}
	log.add(worker, "")
	stats.fechar(res)
}

func umaLinha(s string) string {
	return strings.ReplaceAll(s, "\n", " | ")
}

func aguardarResposta(cli *client.Client, phone string, cfg *dto.Config, antes int) (string, bool) {
	deadline := time.Now().Add(time.Duration(cfg.TimeoutSeconds) * time.Second)
	intervalo := time.Duration(cfg.PollIntervalMs) * time.Millisecond
	for time.Now().Before(deadline) {
		time.Sleep(intervalo)
		inbox, err := cli.BuscarInbox(phone)
		if err != nil || inbox == nil {
			continue
		}
		if inbox.Total > antes && len(inbox.Mensagens) > 0 {
			return inbox.Mensagens[len(inbox.Mensagens)-1].Texto, true
		}
	}
	return "", false
}

func contarInbox(cli *client.Client, phone string) int {
	inbox, err := cli.BuscarInbox(phone)
	if err != nil || inbox == nil {
		return 0
	}
	return inbox.Total
}

func contemAlgumTexto(texto string, alvos []string) string {
	low := strings.ToLower(texto)
	for _, a := range alvos {
		a = strings.ToLower(strings.TrimSpace(a))
		if a != "" && strings.Contains(low, a) {
			return a
		}
	}
	return ""
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

func foneDoCenario(id string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return fmt.Sprintf("5549%09d", h.Sum32()%1_000_000_000)
}
