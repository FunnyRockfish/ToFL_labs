package domain

type TestCase struct {
	InLanguage bool   `json:"in_language"`
	Word       string `json:"word"`
}

type TestResponse struct {
	Tests []TestCase `json:"tests"`
}

type TestReq struct {
	Productions []string `json:"productions"`
	Pos         int      `json:"pos"`
	Neg         int      `json:"neg"`
	WProb       float64  `json:"wprob"`
	Attempts    int      `json:"attempts"`
	MaxSteps    int      `json:"max_steps"`
}
