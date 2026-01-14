package store

import (
	"context"
	"time"

	"qms/admin-service/internal/models"
)

type Store interface {
	CreateBranch(ctx context.Context, branch models.Branch) (models.Branch, error)
	UpdateBranch(ctx context.Context, branch models.Branch) (models.Branch, error)
	DeleteBranch(ctx context.Context, tenantID, branchID string) error
	ListBranches(ctx context.Context, tenantID string) ([]models.Branch, error)

	CreateArea(ctx context.Context, area models.Area) (models.Area, error)
	ListAreas(ctx context.Context, branchID string) ([]models.Area, error)

	CreateService(ctx context.Context, service models.Service) (models.Service, error)
	UpdateService(ctx context.Context, service models.Service) (models.Service, error)
	ListServices(ctx context.Context, branchID string) ([]models.Service, error)

	CreateCounter(ctx context.Context, counter models.Counter) (models.Counter, error)
	ListCounters(ctx context.Context, branchID string) ([]models.Counter, error)
	MapCounterService(ctx context.Context, counterID, serviceID string) error
	ListCounterServices(ctx context.Context, counterID string) ([]models.Service, error)
	RemoveCounterService(ctx context.Context, counterID, serviceID string) error

	InsertAudit(ctx context.Context, audit models.AuditLog) error
	ListAudit(ctx context.Context, tenantID, actionType, userID string) ([]models.AuditLog, error)

	RegisterDevice(ctx context.Context, device models.Device) (models.Device, error)
	ListDevices(ctx context.Context, tenantID string) ([]models.Device, error)
	UpdateDeviceStatus(ctx context.Context, deviceID, status string) error
	CreateDeviceConfig(ctx context.Context, deviceID string, version int, payload string) error
	GetLatestDeviceConfig(ctx context.Context, deviceID string) (int, string, error)
	ListDeviceConfigs(ctx context.Context, deviceID string, limit int) ([]models.DeviceConfig, error)
	RollbackDeviceConfig(ctx context.Context, deviceID string, version int) (int, error)

	UpsertServicePolicy(ctx context.Context, policy models.ServicePolicy) (models.ServicePolicy, error)
	GetServicePolicy(ctx context.Context, tenantID, branchID, serviceID string) (models.ServicePolicy, bool, error)

	CreateRole(ctx context.Context, role models.Role) (models.Role, error)
	ListRoles(ctx context.Context, tenantID string) ([]models.Role, error)
	UpdateRoleName(ctx context.Context, tenantID, roleID, name string) error
	DeleteRole(ctx context.Context, tenantID, roleID string) error
	UpdateUserRole(ctx context.Context, tenantID, userID, roleID string) error
	GetUser(ctx context.Context, tenantID, userID string) (models.UserDetail, bool, error)
	ListUsers(ctx context.Context, tenantID, query string, limit, offset int) ([]models.UserDetail, error)
	GetUserAccess(ctx context.Context, tenantID, userID string) (models.UserAccess, error)
	CreateUser(ctx context.Context, tenantID, email, roleID, passwordHash string) (models.UserDetail, error)
	UpdateUserStatus(ctx context.Context, tenantID, userID string, active bool) error
	ResetUserPassword(ctx context.Context, tenantID, userID, passwordHash string) error
	AddUserBranchAccess(ctx context.Context, tenantID, userID, branchID string) error
	RemoveUserBranchAccess(ctx context.Context, tenantID, userID, branchID string) error
	AddUserServiceAccess(ctx context.Context, tenantID, userID, serviceID string) error
	RemoveUserServiceAccess(ctx context.Context, tenantID, userID, serviceID string) error

	CreateHoliday(ctx context.Context, holiday models.Holiday) (models.Holiday, error)
	ListHolidays(ctx context.Context, tenantID, branchID string) ([]models.Holiday, error)

	CreateApproval(ctx context.Context, approval models.ApprovalRequest) (models.ApprovalRequest, error)
	ApproveRequest(ctx context.Context, approvalID, approverID string) error
	ListApprovals(ctx context.Context, tenantID, status string) ([]models.ApprovalRequest, error)
	GetApproval(ctx context.Context, approvalID string) (models.ApprovalRequest, bool, error)
	ApprovalsEnabled(ctx context.Context, tenantID string) (bool, error)
	GetApprovalPrefs(ctx context.Context, tenantID string) (bool, error)
	SetApprovalPrefs(ctx context.Context, tenantID string, enabled bool) error
	GetSession(ctx context.Context, sessionID string) (Session, error)
}

type Session struct {
	SessionID string
	UserID    string
	TenantID  string
	Role      string
	ExpiresAt time.Time
}
