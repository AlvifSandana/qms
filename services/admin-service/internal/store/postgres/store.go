package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"qms/admin-service/internal/models"
	"qms/admin-service/internal/store"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateBranch(ctx context.Context, branch models.Branch) (models.Branch, error) {
	if branch.BranchID == "" {
		branch.BranchID = uuid.NewString()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO branches (branch_id, tenant_id, name)
		VALUES ($1, $2, $3)
	`, branch.BranchID, branch.TenantID, branch.Name)
	if err != nil {
		return models.Branch{}, err
	}
	return branch, nil
}

func (s *Store) UpdateBranch(ctx context.Context, branch models.Branch) (models.Branch, error) {
	_, err := s.pool.Exec(ctx, `
		UPDATE branches
		SET name = $1
		WHERE branch_id = $2 AND tenant_id = $3
	`, branch.Name, branch.BranchID, branch.TenantID)
	if err != nil {
		return models.Branch{}, err
	}
	return branch, nil
}

func (s *Store) DeleteBranch(ctx context.Context, tenantID, branchID string) error {
	var count int
	row := s.pool.QueryRow(ctx, `
		SELECT COUNT(1)
		FROM services
		WHERE branch_id = $1 AND active = TRUE
	`, branchID)
	if err := row.Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return store.ErrBranchHasServices
	}
	_, err := s.pool.Exec(ctx, `
		DELETE FROM branches
		WHERE branch_id = $1 AND tenant_id = $2
	`, branchID, tenantID)
	return err
}

func (s *Store) ListBranches(ctx context.Context, tenantID string) ([]models.Branch, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT branch_id, tenant_id, name
		FROM branches
		WHERE tenant_id = $1
		ORDER BY name ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var branches []models.Branch
	for rows.Next() {
		var branch models.Branch
		if err := rows.Scan(&branch.BranchID, &branch.TenantID, &branch.Name); err != nil {
			return nil, err
		}
		branches = append(branches, branch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return branches, nil
}

func (s *Store) CreateArea(ctx context.Context, area models.Area) (models.Area, error) {
	if area.AreaID == "" {
		area.AreaID = uuid.NewString()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO areas (area_id, branch_id, name)
		VALUES ($1, $2, $3)
	`, area.AreaID, area.BranchID, area.Name)
	if err != nil {
		return models.Area{}, err
	}
	return area, nil
}

func (s *Store) ListAreas(ctx context.Context, branchID string) ([]models.Area, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT area_id, branch_id, name
		FROM areas
		WHERE branch_id = $1
		ORDER BY name ASC
	`, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var areas []models.Area
	for rows.Next() {
		var area models.Area
		if err := rows.Scan(&area.AreaID, &area.BranchID, &area.Name); err != nil {
			return nil, err
		}
		areas = append(areas, area)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return areas, nil
}

func (s *Store) CreateService(ctx context.Context, service models.Service) (models.Service, error) {
	if service.ServiceID == "" {
		service.ServiceID = uuid.NewString()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO services (service_id, branch_id, name, code, sla_minutes, active)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, service.ServiceID, service.BranchID, service.Name, service.Code, service.SLAMinutes, service.Active)
	if err != nil {
		return models.Service{}, err
	}
	return service, nil
}

func (s *Store) UpdateService(ctx context.Context, service models.Service) (models.Service, error) {
	_, err := s.pool.Exec(ctx, `
		UPDATE services
		SET name = $1, code = $2, sla_minutes = $3, active = $4
		WHERE service_id = $5 AND branch_id = $6
	`, service.Name, service.Code, service.SLAMinutes, service.Active, service.ServiceID, service.BranchID)
	if err != nil {
		return models.Service{}, err
	}
	return service, nil
}

func (s *Store) ListServices(ctx context.Context, branchID string) ([]models.Service, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT service_id, branch_id, name, code, sla_minutes, active
		FROM services
		WHERE branch_id = $1
		ORDER BY name ASC
	`, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var svc models.Service
		if err := rows.Scan(&svc.ServiceID, &svc.BranchID, &svc.Name, &svc.Code, &svc.SLAMinutes, &svc.Active); err != nil {
			return nil, err
		}
		services = append(services, svc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return services, nil
}

func (s *Store) CreateCounter(ctx context.Context, counter models.Counter) (models.Counter, error) {
	if counter.CounterID == "" {
		counter.CounterID = uuid.NewString()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO counters (counter_id, branch_id, area_id, name, status)
		VALUES ($1, $2, $3, $4, $5)
	`, counter.CounterID, counter.BranchID, nullIfEmpty(counter.AreaID), counter.Name, counter.Status)
	if err != nil {
		return models.Counter{}, err
	}
	return counter, nil
}

func (s *Store) ListCounters(ctx context.Context, branchID string) ([]models.Counter, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT counter_id, branch_id, area_id, name, status
		FROM counters
		WHERE branch_id = $1
		ORDER BY name ASC
	`, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counters []models.Counter
	for rows.Next() {
		var counter models.Counter
		if err := rows.Scan(&counter.CounterID, &counter.BranchID, &counter.AreaID, &counter.Name, &counter.Status); err != nil {
			return nil, err
		}
		counters = append(counters, counter)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counters, nil
}

func (s *Store) MapCounterService(ctx context.Context, counterID, serviceID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO counter_services (counter_id, service_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, counterID, serviceID)
	return err
}

func (s *Store) InsertAudit(ctx context.Context, audit models.AuditLog) error {
	if audit.AuditID == "" {
		audit.AuditID = uuid.NewString()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_logs (audit_id, tenant_id, actor_user_id, action_type, target_type, target_id, ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, audit.AuditID, audit.TenantID, nullIfEmpty(audit.ActorUserID), audit.ActionType, audit.TargetType, nullIfEmpty(audit.TargetID), nullIfEmpty(audit.IP), nullIfEmpty(audit.UserAgent))
	return err
}

func (s *Store) ListAudit(ctx context.Context, tenantID, actionType, userID string) ([]models.AuditLog, error) {
	query := `
		SELECT audit_id, tenant_id, actor_user_id, action_type, target_type, target_id, created_at, ip, user_agent
		FROM audit_logs
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	if actionType != "" {
		query += " AND action_type = $2"
		args = append(args, actionType)
	}
	if userID != "" {
		placeholder := len(args) + 1
		query += fmt.Sprintf(" AND actor_user_id = $%d", placeholder)
		args = append(args, userID)
	}
	query += " ORDER BY created_at DESC LIMIT 200"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var logEntry models.AuditLog
		if err := rows.Scan(&logEntry.AuditID, &logEntry.TenantID, &logEntry.ActorUserID, &logEntry.ActionType, &logEntry.TargetType, &logEntry.TargetID, &logEntry.CreatedAt, &logEntry.IP, &logEntry.UserAgent); err != nil {
			return nil, err
		}
		logs = append(logs, logEntry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

func (s *Store) RegisterDevice(ctx context.Context, device models.Device) (models.Device, error) {
	if device.DeviceID == "" {
		device.DeviceID = uuid.NewString()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO devices (device_id, tenant_id, branch_id, area_id, type, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (device_id) DO UPDATE SET branch_id = EXCLUDED.branch_id, area_id = EXCLUDED.area_id, status = EXCLUDED.status
	`, device.DeviceID, device.TenantID, device.BranchID, nullIfEmpty(device.AreaID), device.Type, device.Status)
	if err != nil {
		return models.Device{}, err
	}
	return device, nil
}

func (s *Store) ListDevices(ctx context.Context, tenantID string) ([]models.Device, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT device_id, tenant_id, branch_id, area_id, type, status, last_seen
		FROM devices
		WHERE tenant_id = $1
		ORDER BY device_id ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var device models.Device
		if err := rows.Scan(&device.DeviceID, &device.TenantID, &device.BranchID, &device.AreaID, &device.Type, &device.Status, &device.LastSeen); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return devices, nil
}

func (s *Store) UpdateDeviceStatus(ctx context.Context, deviceID, status string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE devices
		SET status = $1, last_seen = NOW()
		WHERE device_id = $2
	`, status, deviceID)
	return err
}

func (s *Store) CreateDeviceConfig(ctx context.Context, deviceID string, version int, payload string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO device_configs (config_id, device_id, version, payload)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (device_id, version) DO NOTHING
	`, uuid.NewString(), deviceID, version, payload)
	return err
}

func (s *Store) GetLatestDeviceConfig(ctx context.Context, deviceID string) (int, string, error) {
	var version int
	var payload string
	row := s.pool.QueryRow(ctx, `
		SELECT version, payload
		FROM device_configs
		WHERE device_id = $1
		ORDER BY version DESC
		LIMIT 1
	`, deviceID)
	if err := row.Scan(&version, &payload); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, "", nil
		}
		return 0, "", err
	}
	return version, payload, nil
}

func (s *Store) UpsertServicePolicy(ctx context.Context, policy models.ServicePolicy) (models.ServicePolicy, error) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO service_policies (tenant_id, branch_id, service_id, no_show_grace_seconds, return_to_queue)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, branch_id, service_id)
		DO UPDATE SET no_show_grace_seconds = EXCLUDED.no_show_grace_seconds, return_to_queue = EXCLUDED.return_to_queue
	`, policy.TenantID, policy.BranchID, policy.ServiceID, policy.NoShowGraceSeconds, policy.ReturnToQueue)
	if err != nil {
		return models.ServicePolicy{}, err
	}
	return policy, nil
}

func (s *Store) GetServicePolicy(ctx context.Context, tenantID, branchID, serviceID string) (models.ServicePolicy, bool, error) {
	var policy models.ServicePolicy
	row := s.pool.QueryRow(ctx, `
		SELECT tenant_id, branch_id, service_id, no_show_grace_seconds, return_to_queue
		FROM service_policies
		WHERE tenant_id = $1 AND branch_id = $2 AND service_id = $3
	`, tenantID, branchID, serviceID)
	if err := row.Scan(&policy.TenantID, &policy.BranchID, &policy.ServiceID, &policy.NoShowGraceSeconds, &policy.ReturnToQueue); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ServicePolicy{}, false, nil
		}
		return models.ServicePolicy{}, false, err
	}
	return policy, true, nil
}

func (s *Store) CreateRole(ctx context.Context, role models.Role) (models.Role, error) {
	if role.RoleID == "" {
		role.RoleID = uuid.NewString()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO roles (role_id, tenant_id, name)
		VALUES ($1, $2, $3)
	`, role.RoleID, role.TenantID, role.Name)
	if err != nil {
		return models.Role{}, err
	}
	return role, nil
}

func (s *Store) ListRoles(ctx context.Context, tenantID string) ([]models.Role, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT role_id, tenant_id, name
		FROM roles
		WHERE tenant_id = $1
		ORDER BY name ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role.RoleID, &role.TenantID, &role.Name); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *Store) UpdateUserRole(ctx context.Context, tenantID, userID, roleID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users
		SET role_id = $1
		WHERE user_id = $2 AND tenant_id = $3
	`, roleID, userID, tenantID)
	return err
}

func (s *Store) GetUser(ctx context.Context, tenantID, userID string) (models.UserDetail, bool, error) {
	var user models.UserDetail
	row := s.pool.QueryRow(ctx, `
		SELECT u.user_id, u.tenant_id, u.email, u.role_id, r.name, u.active, u.created_at
		FROM users u
		JOIN roles r ON r.role_id = u.role_id
		WHERE u.tenant_id = $1 AND u.user_id = $2
	`, tenantID, userID)
	if err := row.Scan(&user.UserID, &user.TenantID, &user.Email, &user.RoleID, &user.RoleName, &user.Active, &user.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.UserDetail{}, false, nil
		}
		return models.UserDetail{}, false, err
	}
	return user, true, nil
}

func (s *Store) ListUsers(ctx context.Context, tenantID, query string, limit, offset int) ([]models.UserDetail, error) {
	if limit <= 0 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}
	args := []interface{}{tenantID}
	filter := ""
	if query != "" {
		filter = " AND (u.email ILIKE $2 OR u.user_id::text ILIKE $2)"
		args = append(args, "%"+query+"%")
	}
	args = append(args, limit, offset)
	limitPos := len(args) - 1
	offsetPos := len(args)
	rows, err := s.pool.Query(ctx, `
		SELECT u.user_id, u.tenant_id, u.email, u.role_id, r.name, u.active, u.created_at
		FROM users u
		JOIN roles r ON r.role_id = u.role_id
		WHERE u.tenant_id = $1`+filter+`
		ORDER BY u.created_at DESC
		LIMIT $`+strconv.Itoa(limitPos)+` OFFSET $`+strconv.Itoa(offsetPos)+`
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.UserDetail
	for rows.Next() {
		var user models.UserDetail
		if err := rows.Scan(&user.UserID, &user.TenantID, &user.Email, &user.RoleID, &user.RoleName, &user.Active, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *Store) GetUserAccess(ctx context.Context, tenantID, userID string) (models.UserAccess, error) {
	var access models.UserAccess

	branchRows, err := s.pool.Query(ctx, `
		SELECT b.branch_id, b.name
		FROM user_branch_access uba
		JOIN branches b ON b.branch_id = uba.branch_id
		WHERE uba.user_id = $1 AND b.tenant_id = $2
		ORDER BY b.name ASC
	`, userID, tenantID)
	if err != nil {
		return models.UserAccess{}, err
	}
	defer branchRows.Close()
	for branchRows.Next() {
		var item models.UserAccessItem
		if err := branchRows.Scan(&item.ID, &item.Name); err != nil {
			return models.UserAccess{}, err
		}
		access.Branches = append(access.Branches, item)
	}
	if err := branchRows.Err(); err != nil {
		return models.UserAccess{}, err
	}

	serviceRows, err := s.pool.Query(ctx, `
		SELECT s.service_id, s.name
		FROM user_service_access usa
		JOIN services s ON s.service_id = usa.service_id
		JOIN branches b ON b.branch_id = s.branch_id
		WHERE usa.user_id = $1 AND b.tenant_id = $2
		ORDER BY s.name ASC
	`, userID, tenantID)
	if err != nil {
		return models.UserAccess{}, err
	}
	defer serviceRows.Close()
	for serviceRows.Next() {
		var item models.UserAccessItem
		if err := serviceRows.Scan(&item.ID, &item.Name); err != nil {
			return models.UserAccess{}, err
		}
		access.Services = append(access.Services, item)
	}
	if err := serviceRows.Err(); err != nil {
		return models.UserAccess{}, err
	}

	return access, nil
}

func (s *Store) CreateHoliday(ctx context.Context, holiday models.Holiday) (models.Holiday, error) {
	if holiday.HolidayID == "" {
		holiday.HolidayID = uuid.NewString()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO holidays (holiday_id, tenant_id, branch_id, date, name)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, branch_id, date) DO UPDATE SET name = EXCLUDED.name
	`, holiday.HolidayID, holiday.TenantID, holiday.BranchID, holiday.Date, holiday.Name)
	if err != nil {
		return models.Holiday{}, err
	}
	return holiday, nil
}

func (s *Store) ListHolidays(ctx context.Context, tenantID, branchID string) ([]models.Holiday, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT holiday_id, tenant_id, branch_id, date, name
		FROM holidays
		WHERE tenant_id = $1 AND branch_id = $2
		ORDER BY date ASC
	`, tenantID, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holidays []models.Holiday
	for rows.Next() {
		var h models.Holiday
		if err := rows.Scan(&h.HolidayID, &h.TenantID, &h.BranchID, &h.Date, &h.Name); err != nil {
			return nil, err
		}
		holidays = append(holidays, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return holidays, nil
}

func (s *Store) CreateApproval(ctx context.Context, approval models.ApprovalRequest) (models.ApprovalRequest, error) {
	if approval.ApprovalID == "" {
		approval.ApprovalID = uuid.NewString()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO approval_requests (approval_id, tenant_id, request_type, payload, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, approval.ApprovalID, approval.TenantID, approval.RequestType, approval.Payload, "pending", nullIfEmpty(approval.CreatedBy))
	if err != nil {
		return models.ApprovalRequest{}, err
	}
	return approval, nil
}

func (s *Store) ApproveRequest(ctx context.Context, approvalID, approverID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE approval_requests
		SET status = 'approved', approved_by = $2, approved_at = NOW()
		WHERE approval_id = $1 AND status = 'pending'
	`, approvalID, nullIfEmpty(approverID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	var exists bool
	row := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM approval_requests WHERE approval_id = $1
		)
	`, approvalID)
	if err := row.Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return store.ErrApprovalNotFound
	}
	return store.ErrApprovalNotPending
}

func (s *Store) ListApprovals(ctx context.Context, tenantID, status string) ([]models.ApprovalRequest, error) {
	query := `
		SELECT approval_id, tenant_id, request_type, payload, status, created_by, approved_by
		FROM approval_requests
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var approvals []models.ApprovalRequest
	for rows.Next() {
		var a models.ApprovalRequest
		if err := rows.Scan(&a.ApprovalID, &a.TenantID, &a.RequestType, &a.Payload, &a.Status, &a.CreatedBy, &a.ApprovedBy); err != nil {
			return nil, err
		}
		approvals = append(approvals, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return approvals, nil
}

func (s *Store) GetApproval(ctx context.Context, approvalID string) (models.ApprovalRequest, bool, error) {
	var a models.ApprovalRequest
	row := s.pool.QueryRow(ctx, `
		SELECT approval_id, tenant_id, request_type, payload, status, created_by, approved_by
		FROM approval_requests
		WHERE approval_id = $1
	`, approvalID)
	if err := row.Scan(&a.ApprovalID, &a.TenantID, &a.RequestType, &a.Payload, &a.Status, &a.CreatedBy, &a.ApprovedBy); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ApprovalRequest{}, false, nil
		}
		return models.ApprovalRequest{}, false, err
	}
	return a, true, nil
}

func (s *Store) ApprovalsEnabled(ctx context.Context, tenantID string) (bool, error) {
	var enabled bool
	row := s.pool.QueryRow(ctx, `
		SELECT approvals_enabled
		FROM tenant_approval_prefs
		WHERE tenant_id = $1
	`, tenantID)
	if err := row.Scan(&enabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return enabled, nil
}

func (s *Store) GetApprovalPrefs(ctx context.Context, tenantID string) (bool, error) {
	return s.ApprovalsEnabled(ctx, tenantID)
}

func (s *Store) SetApprovalPrefs(ctx context.Context, tenantID string, enabled bool) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO tenant_approval_prefs (tenant_id, approvals_enabled)
		VALUES ($1, $2)
		ON CONFLICT (tenant_id) DO UPDATE SET approvals_enabled = EXCLUDED.approvals_enabled
	`, tenantID, enabled)
	return err
}

func nullIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

var _ = errors.Is
