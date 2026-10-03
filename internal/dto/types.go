package dto

type Config struct {
	ServerURL      string `json:"server_url"`
	MockURL        string `json:"mock_url"`
	Threads        int    `json:"threads"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	PollIntervalMs int    `json:"poll_interval_ms"`
	PhoneNumberID  string `json:"phone_number_id"`
	From           string `json:"from"`
	WaID           string `json:"wa_id"`
	ClienteNome    string `json:"cliente_nome"`
	ScenariosDir   string `json:"scenarios_dir"`
}

type Mensagem struct {
	Texto                  string   `json:"texto"`
	EsperaResposta         bool     `json:"espera_resposta"`
	RespostaEsperadaContem []string `json:"resposta_esperada_contem,omitempty"`
	RespostaNaoContem      []string `json:"resposta_nao_contem,omitempty"`
}

type Scenario struct {
	ID        string     `json:"id"`
	Descricao string     `json:"descricao"`
	Mensagens []Mensagem `json:"mensagens"`
}

type MensagemEnviada struct {
	ScenarioID string `json:"scenario_id"`
	Texto      string `json:"texto"`
	Timestamp  string `json:"timestamp"`
	Para       string `json:"para"`
}

type InboxResponse struct {
	ScenarioID string            `json:"scenario_id"`
	Total      int               `json:"total"`
	Mensagens  []MensagemEnviada `json:"mensagens"`
}
