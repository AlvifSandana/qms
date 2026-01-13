package models

type Service struct {
	ServiceID      string `json:"service_id"`
	BranchID       string `json:"branch_id"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	SLAMinutes     int    `json:"sla_minutes"`
	PriorityPolicy string `json:"priority_policy,omitempty"`
	HoursJSON      string `json:"hours_json,omitempty"`
}
