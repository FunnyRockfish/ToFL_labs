package domain

type RegexReq struct {
	Rg string `json:"regex"`
}

type RegexResponse struct {
	IsCorrect bool     `json:"is_correct"`
	AST       []string `json:"AST_dot,omitempty"`
	Errors    []string `json:"errors,omitempty"`
	Grammar   []string `json:"grammar,omitempty"`
}
