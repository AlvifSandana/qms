package models

type Counter struct {
	CounterID string `json:"counter_id"`
	BranchID  string `json:"branch_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}
