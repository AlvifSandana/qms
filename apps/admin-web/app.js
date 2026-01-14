const adminBase = document.getElementById("adminBase");
const sessionIdInput = document.getElementById("sessionId");
const tenantIdInput = document.getElementById("tenantId");
const roleSelect = document.getElementById("role");
const userIdInput = document.getElementById("userId");
const statusEl = document.getElementById("status");
const sessionHint = document.getElementById("sessionHint");

const branchName = document.getElementById("branchName");
const branchList = document.getElementById("branchList");
const areaBranchId = document.getElementById("areaBranchId");
const areaName = document.getElementById("areaName");
const areaList = document.getElementById("areaList");
const serviceBranchId = document.getElementById("serviceBranchId");
const serviceName = document.getElementById("serviceName");
const serviceCode = document.getElementById("serviceCode");
const serviceSla = document.getElementById("serviceSla");
const servicePriority = document.getElementById("servicePriority");
const serviceHours = document.getElementById("serviceHours");
const serviceList = document.getElementById("serviceList");

const counterBranchId = document.getElementById("counterBranchId");
const counterName = document.getElementById("counterName");
const counterList = document.getElementById("counterList");
const mapCounterId = document.getElementById("mapCounterId");
const mapServiceId = document.getElementById("mapServiceId");
const skillsCounterId = document.getElementById("skillsCounterId");
const counterSkillsList = document.getElementById("counterSkillsList");

const policyBranchId = document.getElementById("policyBranchId");
const policyServiceId = document.getElementById("policyServiceId");
const policyGrace = document.getElementById("policyGrace");
const policyReturn = document.getElementById("policyReturn");
const policyApptRatio = document.getElementById("policyApptRatio");
const policyApptWindow = document.getElementById("policyApptWindow");
const policyApptBoost = document.getElementById("policyApptBoost");
const policyView = document.getElementById("policyView");
const policyHint = document.getElementById("policyHint");

const deviceBranchId = document.getElementById("deviceBranchId");
const deviceAreaId = document.getElementById("deviceAreaId");
const deviceType = document.getElementById("deviceType");
const deviceList = document.getElementById("deviceList");

const holidayBranchId = document.getElementById("holidayBranchId");
const holidayDate = document.getElementById("holidayDate");
const holidayName = document.getElementById("holidayName");
const holidayList = document.getElementById("holidayList");

const approvalStatus = document.getElementById("approvalStatus");
const approvalList = document.getElementById("approvalList");
const approvalDetail = document.getElementById("approvalDetail");
const approvalPref = document.getElementById("approvalPref");
const approvalPrefHint = document.getElementById("approvalPrefHint");
const auditAction = document.getElementById("auditAction");
const auditUser = document.getElementById("auditUser");
const auditList = document.getElementById("auditList");

const configDeviceId = document.getElementById("configDeviceId");
const configVersion = document.getElementById("configVersion");
const configPayload = document.getElementById("configPayload");
const statusDeviceId = document.getElementById("statusDeviceId");
const statusValue = document.getElementById("statusValue");
const historyDeviceId = document.getElementById("historyDeviceId");
const historyLimit = document.getElementById("historyLimit");
const historyList = document.getElementById("historyList");

const roleName = document.getElementById("roleName");
const roleList = document.getElementById("roleList");
const roleAssign = document.getElementById("roleAssign");
const roleEditName = document.getElementById("roleEditName");
const roleEditId = document.getElementById("roleEditId");
const targetUserId = document.getElementById("targetUserId");
const userDetail = document.getElementById("userDetail");
const userQuery = document.getElementById("userQuery");
const userLimit = document.getElementById("userLimit");
const userPage = document.getElementById("userPage");
const userList = document.getElementById("userList");
const userEmail = document.getElementById("userEmail");
const userRoleSelect = document.getElementById("userRoleSelect");
const userPassword = document.getElementById("userPassword");
const userActive = document.getElementById("userActive");
const resetPasswordValue = document.getElementById("resetPasswordValue");
const accessBranches = document.getElementById("accessBranches");
const accessServices = document.getElementById("accessServices");
const accessBranchId = document.getElementById("accessBranchId");
const accessServiceId = document.getElementById("accessServiceId");

document.getElementById("refreshAll").addEventListener("click", refreshAll);
document.getElementById("createBranch").addEventListener("click", onCreateBranch);
document.getElementById("createArea").addEventListener("click", onCreateArea);
document.getElementById("loadAreas").addEventListener("click", loadAreas);
document.getElementById("createService").addEventListener("click", onCreateService);
document.getElementById("createCounter").addEventListener("click", onCreateCounter);
document.getElementById("mapCounter").addEventListener("click", onMapCounter);
document.getElementById("loadCounterSkills").addEventListener("click", loadCounterSkills);
document.getElementById("savePolicy").addEventListener("click", onSavePolicy);
document.getElementById("registerDevice").addEventListener("click", onRegisterDevice);
document.getElementById("createHoliday").addEventListener("click", onCreateHoliday);
document.getElementById("refreshApprovals").addEventListener("click", loadApprovals);
document.getElementById("refreshAudit").addEventListener("click", loadAudit);
document.getElementById("pushConfig").addEventListener("click", onPushConfig);
document.getElementById("updateStatus").addEventListener("click", onUpdateStatus);
document.getElementById("loadHistory").addEventListener("click", loadDeviceConfigHistory);
document.getElementById("createRole").addEventListener("click", onCreateRole);
document.getElementById("updateRole").addEventListener("click", onUpdateRole);
document.getElementById("deleteRole").addEventListener("click", onDeleteRole);
document.getElementById("assignRole").addEventListener("click", onAssignRole);
document.getElementById("saveApprovalPref").addEventListener("click", onSaveApprovalPref);
document.getElementById("loadUser").addEventListener("click", loadUserDetail);
document.getElementById("clearUser").addEventListener("click", clearUserDetail);
document.getElementById("searchUsers").addEventListener("click", searchUsers);
document.getElementById("createUser").addEventListener("click", onCreateUser);
document.getElementById("saveUserStatus").addEventListener("click", onSaveUserStatus);
document.getElementById("resetPassword").addEventListener("click", onResetPassword);
document.getElementById("loadAccess").addEventListener("click", loadUserAccess);
document.getElementById("addBranchAccess").addEventListener("click", onAddBranchAccess);
document.getElementById("removeBranchAccess").addEventListener("click", onRemoveBranchAccess);
document.getElementById("addServiceAccess").addEventListener("click", onAddServiceAccess);
document.getElementById("removeServiceAccess").addEventListener("click", onRemoveServiceAccess);
document.getElementById("prevPage").addEventListener("click", () => changePage(-1));
document.getElementById("nextPage").addEventListener("click", () => changePage(1));
document.getElementById("filterApprovalAudit").addEventListener("click", filterApprovalAudit);

window.addEventListener("error", (event) => {
  setHint(event.error?.message || event.message || "Unexpected error.");
});
window.addEventListener("unhandledrejection", (event) => {
  setHint(event.reason?.message || "Unexpected error.");
});

function headers() {
  const sessionId = sessionIdInput.value.trim();
  return {
    "Content-Type": "application/json",
    ...(sessionId ? { Authorization: `Bearer ${sessionId}` } : {}),
    "X-Role": roleSelect.value,
    "X-User-ID": userIdInput.value.trim(),
    "X-Tenant-ID": tenantIdInput.value.trim(),
  };
}

function setStatus(text) {
  statusEl.textContent = text;
}

function setHint(text) {
  sessionHint.textContent = text || "";
}

async function api(path, options = {}) {
  const url = `${adminBase.value.replace(/\/$/, "")}${path}`;
  const response = await fetch(url, {
    ...options,
    headers: {
      ...headers(),
      ...(options.headers || {}),
    },
  });
  let body = null;
  if (response.status !== 204) {
    body = await response.json().catch(() => null);
  }
  if (!response.ok) {
    const msg = body?.error?.message || body?.error?.code || response.statusText;
    throw new Error(msg);
  }
  return body;
}

function renderList(target, items, formatter) {
  target.innerHTML = "";
  if (!items || items.length === 0) {
    target.textContent = "No data.";
    return;
  }
  for (const item of items) {
    target.appendChild(formatter(item));
  }
}

function itemCardActions(title, subtitle, actions = []) {
  const wrapper = document.createElement("div");
  wrapper.className = "item";
  const info = document.createElement("div");
  info.innerHTML = `<strong>${title}</strong><br><small>${subtitle}</small>`;
  wrapper.appendChild(info);
  for (const action of actions) {
    const button = document.createElement("button");
    button.textContent = action.label;
    button.addEventListener("click", action.onClick);
    wrapper.appendChild(button);
  }
  return wrapper;
}

function itemCard(title, subtitle, actionLabel, action) {
  const wrapper = document.createElement("div");
  wrapper.className = "item";
  const info = document.createElement("div");
  info.innerHTML = `<strong>${title}</strong><br><small>${subtitle}</small>`;
  wrapper.appendChild(info);
  if (actionLabel) {
    const button = document.createElement("button");
    button.textContent = actionLabel;
    button.addEventListener("click", action);
    wrapper.appendChild(button);
  }
  return wrapper;
}

async function refreshAll() {
  setStatus("Loading...");
  try {
    await Promise.all([
      loadBranches(),
      loadServices(),
      loadCounters(),
      loadPolicy(),
      loadDevices(),
      loadHolidays(),
      loadRoles(),
      loadApprovals(),
      loadApprovalPref(),
      loadAudit(),
    ]);
    setStatus("Ready");
    setHint("");
  } catch (err) {
    setStatus("Error");
    setHint(err.message);
  }
}

async function loadBranches() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId) {
    branchList.textContent = "Tenant ID required.";
    return;
  }
  const data = await api(`/api/admin/branches?tenant_id=${tenantId}`);
  renderList(branchList, data, (branch) =>
    itemCard(branch.name, branch.branch_id, "Delete", () => deleteBranch(branch.branch_id))
  );
}

async function loadAreas() {
  const branchId = areaBranchId.value.trim();
  if (!branchId) {
    areaList.textContent = "Branch ID required.";
    return;
  }
  const data = await api(`/api/admin/areas?branch_id=${branchId}`);
  renderList(areaList, data, (area) =>
    itemCard(area.name, area.area_id, null, null)
  );
}

async function onCreateBranch() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId || !branchName.value.trim()) {
    setHint("Tenant ID and branch name are required.");
    return;
  }
  const payload = { tenant_id: tenantId, name: branchName.value.trim() };
  const created = await api("/api/admin/branches", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  branchName.value = "";
  if (created?.status === "pending") {
    setHint("Branch pending approval.");
  }
  await loadBranches();
}

async function onCreateArea() {
  const branchId = areaBranchId.value.trim();
  if (!branchId || !areaName.value.trim()) {
    setHint("Branch ID and area name are required.");
    return;
  }
  const payload = { branch_id: branchId, name: areaName.value.trim() };
  const created = await api("/api/admin/areas", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  areaName.value = "";
  if (created?.status === "pending") {
    setHint("Area pending approval.");
  }
  await loadAreas();
}

async function deleteBranch(branchId) {
  const tenantId = tenantIdInput.value.trim();
  await api(`/api/admin/branches/${branchId}?tenant_id=${tenantId}`, { method: "DELETE" });
  await loadBranches();
}

async function loadServices() {
  const branchId = serviceBranchId.value.trim();
  if (!branchId) {
    serviceList.textContent = "Branch ID required.";
    return;
  }
  const data = await api(`/api/admin/services?branch_id=${branchId}`);
  renderList(serviceList, data, (service) =>
    itemCard(
      `${service.name} (${service.code})`,
      `${service.service_id} · SLA ${service.sla_minutes}m · ${service.priority_policy || "fifo"}`,
      null,
      null
    )
  );
}

async function onCreateService() {
  const branchId = serviceBranchId.value.trim();
  if (!branchId || !serviceName.value.trim() || !serviceCode.value.trim()) {
    setHint("Branch ID, name, and code are required.");
    return;
  }
  if (serviceHours.value.trim()) {
    try {
      JSON.parse(serviceHours.value);
    } catch (err) {
      setHint("Hours JSON is invalid.");
      return;
    }
  }
  const payload = {
    branch_id: branchId,
    name: serviceName.value.trim(),
    code: serviceCode.value.trim().toUpperCase(),
    sla_minutes: Number(serviceSla.value) || 5,
    priority_policy: servicePriority.value || "fifo",
    hours_json: serviceHours.value.trim(),
  };
  const created = await api("/api/admin/services", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  serviceName.value = "";
  serviceCode.value = "";
  serviceHours.value = "";
  if (created?.status === "pending") {
    setHint("Service pending approval.");
  }
  await loadServices();
}

async function loadCounters() {
  const branchId = counterBranchId.value.trim();
  if (!branchId) {
    counterList.textContent = "Branch ID required.";
    return;
  }
  const data = await api(`/api/admin/counters?branch_id=${branchId}`);
  renderList(counterList, data, (counter) =>
    itemCard(counter.name, counter.counter_id, null, null)
  );
}

async function onCreateCounter() {
  const branchId = counterBranchId.value.trim();
  if (!branchId || !counterName.value.trim()) {
    setHint("Branch ID and counter name are required.");
    return;
  }
  const payload = { branch_id: branchId, name: counterName.value.trim(), status: "active" };
  const created = await api("/api/admin/counters", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  counterName.value = "";
  if (created?.status === "pending") {
    setHint("Counter pending approval.");
  }
  await loadCounters();
}

async function onMapCounter() {
  if (!mapCounterId.value.trim() || !mapServiceId.value.trim()) {
    setHint("Counter ID and Service ID required.");
    return;
  }
  await api(`/api/admin/counters/${mapCounterId.value.trim()}/services`, {
    method: "POST",
    body: JSON.stringify({ service_id: mapServiceId.value.trim() }),
  });
  setHint("Mapping saved.");
}

async function loadCounterSkills() {
  const counterId = skillsCounterId.value.trim();
  if (!counterId) {
    counterSkillsList.textContent = "Counter ID required.";
    return;
  }
  const data = await api(`/api/admin/counters/${counterId}/services`);
  renderList(counterSkillsList, data, (service) =>
    itemCardActions(
      `${service.name} (${service.code})`,
      service.service_id,
      [
        { label: "Remove", onClick: () => removeCounterSkill(counterId, service.service_id) },
      ]
    )
  );
}

async function removeCounterSkill(counterId, serviceId) {
  await api(`/api/admin/counters/${counterId}/services`, {
    method: "DELETE",
    body: JSON.stringify({ service_id: serviceId }),
  });
  await loadCounterSkills();
}

async function loadPolicy() {
  const tenantId = tenantIdInput.value.trim();
  const branchId = policyBranchId.value.trim();
  const serviceId = policyServiceId.value.trim();
  if (!tenantId || !branchId || !serviceId) {
    policyView.textContent = "Tenant, branch, and service IDs required.";
    return;
  }
  policyHint.textContent = "";
  const data = await api(`/api/admin/policies/service?tenant_id=${tenantId}&branch_id=${branchId}&service_id=${serviceId}`);
  if (data) {
    if (typeof data.no_show_grace_seconds === "number") {
      policyGrace.value = String(data.no_show_grace_seconds);
    }
    if (typeof data.return_to_queue === "boolean") {
      policyReturn.value = data.return_to_queue ? "true" : "false";
    }
    if (typeof data.appointment_ratio_percent === "number") {
      policyApptRatio.value = String(data.appointment_ratio_percent);
    }
    if (typeof data.appointment_window_size === "number") {
      policyApptWindow.value = String(data.appointment_window_size);
    }
    if (typeof data.appointment_boost_minutes === "number") {
      policyApptBoost.value = String(data.appointment_boost_minutes);
    }
  }
  renderList(policyView, data ? [data] : [], (policy) =>
    itemCard(
      `Grace ${policy.no_show_grace_seconds}s · Ratio ${policy.appointment_ratio_percent || 0}%`,
      `Return: ${policy.return_to_queue} · Window ${policy.appointment_window_size || 10} · Boost ${policy.appointment_boost_minutes || 0}m`,
      null,
      null
    )
  );
}

async function onSavePolicy() {
  const tenantId = tenantIdInput.value.trim();
  const branchId = policyBranchId.value.trim();
  const serviceId = policyServiceId.value.trim();
  if (!tenantId || !branchId || !serviceId) {
    setHint("Tenant, branch, service IDs required.");
    return;
  }
  policyHint.textContent = "";
  const ratio = Number(policyApptRatio.value);
  const windowSize = Number(policyApptWindow.value);
  const boostMinutes = Number(policyApptBoost.value);
  if (Number.isNaN(ratio) || ratio < 0 || ratio > 100) {
    policyHint.textContent = "Appointment ratio must be between 0 and 100.";
    return;
  }
  if (Number.isNaN(windowSize) || windowSize < 1) {
    policyHint.textContent = "Appointment window must be 1 or greater.";
    return;
  }
  if (Number.isNaN(boostMinutes) || boostMinutes < 0) {
    policyHint.textContent = "Appointment boost must be 0 or greater.";
    return;
  }
  const payload = {
    tenant_id: tenantId,
    branch_id: branchId,
    service_id: serviceId,
    no_show_grace_seconds: Number(policyGrace.value) || 300,
    return_to_queue: policyReturn.value === "true",
    appointment_ratio_percent: ratio || 0,
    appointment_window_size: windowSize || 10,
    appointment_boost_minutes: boostMinutes || 0,
  };
  const created = await api("/api/admin/policies/service", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  if (created?.status === "pending") {
    setHint("Policy update pending approval.");
  }
  await loadPolicy();
}

async function loadDevices() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId) {
    deviceList.textContent = "Tenant ID required.";
    return;
  }
  const data = await api(`/api/admin/devices?tenant_id=${tenantId}`);
  renderList(deviceList, data, (device) =>
    itemCard(
      `${device.type} · ${device.status}`,
      `${device.device_id} · branch ${device.branch_id}`,
      null,
      null
    )
  );
}

async function onRegisterDevice() {
  const tenantId = tenantIdInput.value.trim();
  const branchId = deviceBranchId.value.trim();
  if (!tenantId || !branchId) {
    setHint("Tenant and branch IDs required.");
    return;
  }
  const payload = {
    tenant_id: tenantId,
    branch_id: branchId,
    area_id: deviceAreaId.value.trim(),
    type: deviceType.value,
    status: "offline",
  };
  const created = await api("/api/admin/devices", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  if (created?.status === "pending") {
    setHint("Device registration pending approval.");
  }
  await loadDevices();
}

async function loadHolidays() {
  const tenantId = tenantIdInput.value.trim();
  const branchId = holidayBranchId.value.trim();
  if (!tenantId || !branchId) {
    holidayList.textContent = "Tenant and branch IDs required.";
    return;
  }
  const data = await api(`/api/admin/holidays?tenant_id=${tenantId}&branch_id=${branchId}`);
  renderList(holidayList, data, (holiday) =>
    itemCard(holiday.name, `${holiday.date} · ${holiday.holiday_id}`, null, null)
  );
}

async function onCreateHoliday() {
  const tenantId = tenantIdInput.value.trim();
  const branchId = holidayBranchId.value.trim();
  if (!tenantId || !branchId || !holidayDate.value || !holidayName.value.trim()) {
    setHint("Tenant, branch, date, and name required.");
    return;
  }
  const payload = {
    tenant_id: tenantId,
    branch_id: branchId,
    date: holidayDate.value,
    name: holidayName.value.trim(),
  };
  const created = await api("/api/admin/holidays", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  if (created?.status === "pending") {
    setHint("Holiday pending approval.");
  }
  await loadHolidays();
}

async function loadApprovals() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId) {
    approvalList.textContent = "Tenant ID required.";
    return;
  }
  const status = approvalStatus.value;
  const data = await api(`/api/admin/approvals?tenant_id=${tenantId}&status=${status}`);
  renderList(approvalList, data, (approval) => {
    const actions = [
      { label: "View", onClick: () => showApprovalDetail(approval) },
    ];
    if (approval.status === "pending") {
      actions.push({ label: "Approve", onClick: () => approveRequest(approval.approval_id) });
    }
    return itemCardActions(
      `${approval.request_type} · ${approval.status}`,
      `${approval.approval_id}`,
      actions
    );
  });
}

async function approveRequest(approvalId) {
  await api(`/api/admin/approvals/${approvalId}/approve`, { method: "PUT" });
  await loadApprovals();
}

async function loadAudit() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId) {
    auditList.textContent = "Tenant ID required.";
    return;
  }
  const params = new URLSearchParams({
    tenant_id: tenantId,
  });
  const action = auditAction.value.trim();
  if (action) {
    params.append("action_type", action);
  }
  if (auditUser.value.trim()) {
    params.append("user_id", auditUser.value.trim());
  }
  const data = await api(`/api/admin/audit?${params.toString()}`);
  renderList(auditList, data, (entry) =>
    itemCard(
      `${entry.action_type}`,
      `${entry.actor_user_id || "system"} · ${entry.created_at}`,
      null,
      null
    )
  );
}

function showApprovalDetail(approval) {
  const payload = approval.payload || "";
  try {
    const parsed = JSON.parse(payload);
    approvalDetail.textContent = JSON.stringify(parsed, null, 2);
  } catch (err) {
    approvalDetail.textContent = payload || "No payload.";
  }
}

async function loadApprovalPref() {
  approvalPrefHint.textContent = "";
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId) {
    approvalPrefHint.textContent = "Tenant ID required.";
    return;
  }
  const data = await api(`/api/admin/approvals/prefs?tenant_id=${tenantId}`);
  approvalPref.value = data.approvals_enabled ? "true" : "false";
}

async function onSaveApprovalPref() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId) {
    approvalPrefHint.textContent = "Tenant ID required.";
    return;
  }
  const payload = {
    tenant_id: tenantId,
    approvals_enabled: approvalPref.value === "true",
  };
  const data = await api("/api/admin/approvals/prefs", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  approvalPref.value = data.approvals_enabled ? "true" : "false";
  approvalPrefHint.textContent = "Preferences saved.";
}

function filterApprovalAudit() {
  auditAction.value = "approval.prefs_update";
  loadAudit().catch((err) => setHint(err.message));
}

function clearUserDetail() {
  userDetail.textContent = "No user loaded.";
}

async function loadUserDetail() {
  const tenantId = tenantIdInput.value.trim();
  const userId = targetUserId.value.trim();
  if (!tenantId || !userId) {
    userDetail.textContent = "Tenant ID and user ID required.";
    return;
  }
  const data = await api(`/api/admin/users/${userId}?tenant_id=${tenantId}`);
  if (typeof data.active === "boolean") {
    userActive.value = data.active ? "true" : "false";
  }
  userDetail.textContent = JSON.stringify(data, null, 2);
}

async function onPushConfig() {
  if (!configDeviceId.value.trim()) {
    setHint("Device ID required.");
    return;
  }
  let payloadValue = {};
  if (configPayload.value.trim()) {
    try {
      payloadValue = JSON.parse(configPayload.value);
    } catch (err) {
      setHint("Config JSON is invalid.");
      return;
    }
  }
  const payload = {
    device_id: configDeviceId.value.trim(),
    version: Number(configVersion.value) || 1,
    payload: payloadValue,
  };
  const created = await api("/api/admin/device-configs", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  if (created?.status === "pending") {
    setHint("Device config pending approval.");
  } else {
    setHint("Device config pushed.");
  }
}

async function onUpdateStatus() {
  if (!statusDeviceId.value.trim()) {
    setHint("Device ID required.");
    return;
  }
  await api(`/api/admin/devices/${statusDeviceId.value.trim()}/status`, {
    method: "PUT",
    body: JSON.stringify({ status: statusValue.value }),
  });
  setHint("Device status updated.");
  await loadDevices();
}

async function loadDeviceConfigHistory() {
  const deviceId = historyDeviceId.value.trim();
  if (!deviceId) {
    historyList.textContent = "Device ID required.";
    return;
  }
  const limit = Number(historyLimit.value) || 10;
  const data = await api(`/api/admin/device-configs/${deviceId}?limit=${limit}`);
  renderList(historyList, data, (config) =>
    itemCardActions(
      `Version ${config.version}`,
      `${config.created_at || "unknown"} · ${config.device_id}`,
      [
        { label: "Rollback", onClick: () => rollbackDeviceConfig(deviceId, config.version) },
      ]
    )
  );
}

async function rollbackDeviceConfig(deviceId, version) {
  if (!confirm(`Rollback device ${deviceId} to version ${version}?`)) {
    return;
  }
  await api(`/api/admin/device-configs/${deviceId}/rollback`, {
    method: "POST",
    body: JSON.stringify({ version }),
  });
  setHint(`Rollback queued for version ${version}.`);
  await loadDeviceConfigHistory();
}

async function loadRoles() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId) {
    roleList.textContent = "Tenant ID required.";
    roleAssign.innerHTML = "";
    userRoleSelect.innerHTML = "";
    return;
  }
  const data = await api(`/api/admin/roles?tenant_id=${tenantId}`);
  renderList(roleList, data, (role) =>
    itemCardActions(
      role.name,
      role.role_id,
      [
        {
          label: "Edit",
          onClick: () => {
            roleEditId.value = role.role_id;
            roleEditName.value = role.name;
          },
        },
      ]
    )
  );
  roleAssign.innerHTML = "";
  userRoleSelect.innerHTML = "";
  for (const role of data || []) {
    const option = document.createElement("option");
    option.value = role.role_id;
    option.textContent = `${role.name} (${role.role_id.slice(0, 6)})`;
    roleAssign.appendChild(option);
    userRoleSelect.appendChild(option.cloneNode(true));
  }
}

async function onCreateRole() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId || !roleName.value.trim()) {
    setHint("Tenant ID and role name required.");
    return;
  }
  await api("/api/admin/roles", {
    method: "POST",
    body: JSON.stringify({ tenant_id: tenantId, name: roleName.value.trim() }),
  });
  roleName.value = "";
  await loadRoles();
}

async function onUpdateRole() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId || !roleEditId.value.trim() || !roleEditName.value.trim()) {
    setHint("Tenant ID, role ID, and new name required.");
    return;
  }
  await api(`/api/admin/roles/${roleEditId.value.trim()}`, {
    method: "PUT",
    body: JSON.stringify({ tenant_id: tenantId, name: roleEditName.value.trim() }),
  });
  setHint("Role updated.");
  await loadRoles();
}

async function onDeleteRole() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId || !roleEditId.value.trim()) {
    setHint("Tenant ID and role ID required.");
    return;
  }
  if (!confirm("Delete this role? This cannot be undone.")) {
    return;
  }
  await api(`/api/admin/roles/${roleEditId.value.trim()}?tenant_id=${tenantId}`, {
    method: "DELETE",
  });
  roleEditId.value = "";
  roleEditName.value = "";
  setHint("Role deleted.");
  await loadRoles();
}

async function onAssignRole() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId || !targetUserId.value.trim() || !roleAssign.value) {
    setHint("Tenant ID, user ID, and role required.");
    return;
  }
  await api(`/api/admin/users/${targetUserId.value.trim()}/role`, {
    method: "PUT",
    body: JSON.stringify({ tenant_id: tenantId, role_id: roleAssign.value }),
  });
  setHint("Role assigned.");
  loadUserDetail().catch(() => {});
}

async function onCreateUser() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId || !userEmail.value.trim() || !userRoleSelect.value) {
    setHint("Tenant ID, email, and role required.");
    return;
  }
  const payload = {
    tenant_id: tenantId,
    email: userEmail.value.trim(),
    role_id: userRoleSelect.value,
    password: userPassword.value.trim(),
  };
  const data = await api("/api/admin/users", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  userEmail.value = "";
  userPassword.value = "";
  if (data?.temp_password) {
    setHint(`User created. Temp password: ${data.temp_password}`);
  } else {
    setHint("User created.");
  }
  await searchUsers();
}

async function onSaveUserStatus() {
  const tenantId = tenantIdInput.value.trim();
  const userId = targetUserId.value.trim();
  if (!tenantId || !userId) {
    setHint("Tenant ID and user ID required.");
    return;
  }
  await api(`/api/admin/users/${userId}/status`, {
    method: "PUT",
    body: JSON.stringify({ tenant_id: tenantId, active: userActive.value === "true" }),
  });
  setHint("User status updated.");
  await loadUserDetail();
}

async function onResetPassword() {
  const tenantId = tenantIdInput.value.trim();
  const userId = targetUserId.value.trim();
  if (!tenantId || !userId) {
    setHint("Tenant ID and user ID required.");
    return;
  }
  if (!confirm(`Reset password for user ${userId}?`)) {
    return;
  }
  const payload = {
    tenant_id: tenantId,
    new_password: resetPasswordValue.value.trim(),
  };
  const data = await api(`/api/admin/users/${userId}/reset-password`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
  resetPasswordValue.value = "";
  setHint(`Temp password: ${data.temp_password}`);
}

async function searchUsers() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId) {
    userList.textContent = "Tenant ID required.";
    return;
  }
  const query = userQuery.value.trim();
  const limit = Number(userLimit.value) || 25;
  const page = Number(userPage.value) || 1;
  const params = new URLSearchParams({
    tenant_id: tenantId,
    query,
    limit: String(limit),
    page: String(page),
  });
  const data = await api(`/api/admin/users?${params.toString()}`);
  renderList(userList, data, (user) =>
    itemCardActions(
      `${user.email} · ${user.role_name}`,
      `${user.user_id}`,
      [
        { label: "Load", onClick: () => loadUserDetailFrom(user.user_id) },
      ]
    )
  );
}

async function loadUserDetailFrom(userId) {
  targetUserId.value = userId;
  await loadUserDetail();
}

function changePage(delta) {
  const current = Number(userPage.value) || 1;
  const next = Math.max(1, current + delta);
  userPage.value = String(next);
  searchUsers().catch(() => {});
}

async function loadUserAccess() {
  const tenantId = tenantIdInput.value.trim();
  const userId = targetUserId.value.trim();
  if (!tenantId || !userId) {
    accessBranches.textContent = "Tenant ID and user ID required.";
    accessServices.textContent = "";
    return;
  }
  const data = await api(`/api/admin/users/${userId}/access?tenant_id=${tenantId}`);
  renderList(accessBranches, data.branches, (item) =>
    itemCard(item.name, item.id, null, null)
  );
  renderList(accessServices, data.services, (item) =>
    itemCard(item.name, item.id, null, null)
  );
}

async function onAddBranchAccess() {
  const tenantId = tenantIdInput.value.trim();
  const userId = targetUserId.value.trim();
  const branchId = accessBranchId.value.trim();
  if (!tenantId || !userId || !branchId) {
    setHint("Tenant ID, user ID, and branch ID required.");
    return;
  }
  await api(`/api/admin/users/${userId}/access/branches`, {
    method: "POST",
    body: JSON.stringify({ tenant_id: tenantId, id: branchId }),
  });
  accessBranchId.value = "";
  await loadUserAccess();
}

async function onRemoveBranchAccess() {
  const tenantId = tenantIdInput.value.trim();
  const userId = targetUserId.value.trim();
  const branchId = accessBranchId.value.trim();
  if (!tenantId || !userId || !branchId) {
    setHint("Tenant ID, user ID, and branch ID required.");
    return;
  }
  if (!confirm(`Remove branch ${branchId} from user ${userId}?`)) {
    return;
  }
  await api(`/api/admin/users/${userId}/access/branches`, {
    method: "DELETE",
    body: JSON.stringify({ tenant_id: tenantId, id: branchId }),
  });
  accessBranchId.value = "";
  await loadUserAccess();
}

async function onAddServiceAccess() {
  const tenantId = tenantIdInput.value.trim();
  const userId = targetUserId.value.trim();
  const serviceId = accessServiceId.value.trim();
  if (!tenantId || !userId || !serviceId) {
    setHint("Tenant ID, user ID, and service ID required.");
    return;
  }
  await api(`/api/admin/users/${userId}/access/services`, {
    method: "POST",
    body: JSON.stringify({ tenant_id: tenantId, id: serviceId }),
  });
  accessServiceId.value = "";
  await loadUserAccess();
}

async function onRemoveServiceAccess() {
  const tenantId = tenantIdInput.value.trim();
  const userId = targetUserId.value.trim();
  const serviceId = accessServiceId.value.trim();
  if (!tenantId || !userId || !serviceId) {
    setHint("Tenant ID, user ID, and service ID required.");
    return;
  }
  if (!confirm(`Remove service ${serviceId} from user ${userId}?`)) {
    return;
  }
  await api(`/api/admin/users/${userId}/access/services`, {
    method: "DELETE",
    body: JSON.stringify({ tenant_id: tenantId, id: serviceId }),
  });
  accessServiceId.value = "";
  await loadUserAccess();
}

refreshAll().catch((err) => setHint(err.message));
