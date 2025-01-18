package domain

type GrammarRequest struct {
	Productions []string `json:"productions"`
}

type GrammarResponse struct {
	Productions []string `json:"productions"`
}
