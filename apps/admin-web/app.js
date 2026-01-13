const adminBase = document.getElementById("adminBase");
const tenantIdInput = document.getElementById("tenantId");
const roleSelect = document.getElementById("role");
const userIdInput = document.getElementById("userId");
const statusEl = document.getElementById("status");
const sessionHint = document.getElementById("sessionHint");

const branchName = document.getElementById("branchName");
const branchList = document.getElementById("branchList");
const serviceBranchId = document.getElementById("serviceBranchId");
const serviceName = document.getElementById("serviceName");
const serviceCode = document.getElementById("serviceCode");
const serviceSla = document.getElementById("serviceSla");
const serviceList = document.getElementById("serviceList");

const counterBranchId = document.getElementById("counterBranchId");
const counterName = document.getElementById("counterName");
const counterList = document.getElementById("counterList");
const mapCounterId = document.getElementById("mapCounterId");
const mapServiceId = document.getElementById("mapServiceId");

const policyBranchId = document.getElementById("policyBranchId");
const policyServiceId = document.getElementById("policyServiceId");
const policyGrace = document.getElementById("policyGrace");
const policyReturn = document.getElementById("policyReturn");
const policyView = document.getElementById("policyView");

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
const auditAction = document.getElementById("auditAction");
const auditUser = document.getElementById("auditUser");
const auditList = document.getElementById("auditList");

document.getElementById("refreshAll").addEventListener("click", refreshAll);
document.getElementById("createBranch").addEventListener("click", onCreateBranch);
document.getElementById("createService").addEventListener("click", onCreateService);
document.getElementById("createCounter").addEventListener("click", onCreateCounter);
document.getElementById("mapCounter").addEventListener("click", onMapCounter);
document.getElementById("savePolicy").addEventListener("click", onSavePolicy);
document.getElementById("registerDevice").addEventListener("click", onRegisterDevice);
document.getElementById("createHoliday").addEventListener("click", onCreateHoliday);
document.getElementById("refreshApprovals").addEventListener("click", loadApprovals);
document.getElementById("refreshAudit").addEventListener("click", loadAudit);

function headers() {
  return {
    "Content-Type": "application/json",
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
      loadApprovals(),
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
      `${service.service_id} · SLA ${service.sla_minutes}m`,
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
  const payload = {
    branch_id: branchId,
    name: serviceName.value.trim(),
    code: serviceCode.value.trim().toUpperCase(),
    sla_minutes: Number(serviceSla.value) || 5,
  };
  const created = await api("/api/admin/services", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  serviceName.value = "";
  serviceCode.value = "";
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

async function loadPolicy() {
  const tenantId = tenantIdInput.value.trim();
  const branchId = policyBranchId.value.trim();
  const serviceId = policyServiceId.value.trim();
  if (!tenantId || !branchId || !serviceId) {
    policyView.textContent = "Tenant, branch, and service IDs required.";
    return;
  }
  const data = await api(`/api/admin/policies/service?tenant_id=${tenantId}&branch_id=${branchId}&service_id=${serviceId}`);
  renderList(policyView, data ? [data] : [], (policy) =>
    itemCard(
      `Grace ${policy.no_show_grace_seconds}s`,
      `Return to queue: ${policy.return_to_queue}`,
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
  const payload = {
    tenant_id: tenantId,
    branch_id: branchId,
    service_id: serviceId,
    no_show_grace_seconds: Number(policyGrace.value) || 300,
    return_to_queue: policyReturn.value === "true",
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
  renderList(approvalList, data, (approval) =>
    itemCard(
      `${approval.request_type} · ${approval.status}`,
      `${approval.approval_id}`,
      approval.status === "pending" ? "Approve" : null,
      () => approveRequest(approval.approval_id)
    )
  );
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
  if (auditAction.value.trim()) {
    params.append("action_type", auditAction.value.trim());
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

refreshAll().catch((err) => setHint(err.message));
