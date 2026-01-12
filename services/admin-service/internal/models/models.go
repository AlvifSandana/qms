package models

type Branch struct {
	BranchID string `json:"branch_id"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
}

type Area struct {
	AreaID   string `json:"area_id"`
	BranchID string `json:"branch_id"`
	Name     string `json:"name"`
}

type Service struct {
	ServiceID  string `json:"service_id"`
	BranchID   string `json:"branch_id"`
	Name       string `json:"name"`
	Code       string `json:"code"`
	SLAMinutes int    `json:"sla_minutes"`
	Active     bool   `json:"active"`
}

type Counter struct {
	CounterID string `json:"counter_id"`
	BranchID  string `json:"branch_id"`
	AreaID    string `json:"area_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

type AuditLog struct {
	AuditID     string `json:"audit_id"`
	TenantID    string `json:"tenant_id"`
	ActorUserID string `json:"actor_user_id"`
	ActionType  string `json:"action_type"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
	CreatedAt   string `json:"created_at"`
	IP          string `json:"ip"`
	UserAgent   string `json:"user_agent"`
}

type Device struct {
	DeviceID string `json:"device_id"`
	TenantID string `json:"tenant_id"`
	BranchID string `json:"branch_id"`
	AreaID   string `json:"area_id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	LastSeen string `json:"last_seen"`
}

type ServicePolicy struct {
	TenantID          string `json:"tenant_id"`
	BranchID          string `json:"branch_id"`
	ServiceID         string `json:"service_id"`
	NoShowGraceSeconds int   `json:"no_show_grace_seconds"`
	ReturnToQueue     bool   `json:"return_to_queue"`
}

type Role struct {
	RoleID   string `json:"role_id"`
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
}

type Holiday struct {
	HolidayID string `json:"holiday_id"`
	TenantID  string `json:"tenant_id"`
	BranchID  string `json:"branch_id"`
	Date      string `json:"date"`
	Name      string `json:"name"`
}

type ApprovalRequest struct {
	ApprovalID string `json:"approval_id"`
	TenantID   string `json:"tenant_id"`
	RequestType string `json:"request_type"`
	Payload     string `json:"payload"`
	Status      string `json:"status"`
	CreatedBy   string `json:"created_by"`
	ApprovedBy  string `json:"approved_by"`
}
