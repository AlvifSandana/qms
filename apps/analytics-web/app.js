const analyticsBase = document.getElementById("analyticsBase");
const tenantIdInput = document.getElementById("tenantId");
const branchIdInput = document.getElementById("branchId");
const serviceIdInput = document.getElementById("serviceId");
const fromDate = document.getElementById("fromDate");
const toDate = document.getElementById("toDate");
const statusEl = document.getElementById("status");
const alertBox = document.getElementById("alert");
const refreshBtn = document.getElementById("refreshBtn");
const toggleAuto = document.getElementById("toggleAuto");
const exportBtn = document.getElementById("exportBtn");

const avgWait = document.getElementById("avgWait");
const avgService = document.getElementById("avgService");
const totalCount = document.getElementById("totalCount");
const queueLength = document.getElementById("queueLength");
const servingCount = document.getElementById("servingCount");

const reportCron = document.getElementById("reportCron");
const reportChannel = document.getElementById("reportChannel");
const reportRecipient = document.getElementById("reportRecipient");
const reportList = document.getElementById("reportList");

const anomalyList = document.getElementById("anomalyList");

let autoTimer = null;

function setStatus(text) {
  statusEl.textContent = text;
}

function setAlert(message) {
  if (!message) {
    alertBox.hidden = true;
    alertBox.textContent = "";
    return;
  }
  alertBox.textContent = message;
  alertBox.hidden = false;
}

function formatSeconds(value) {
  if (value === undefined || value === null || Number.isNaN(value)) {
    return "-";
  }
  const minutes = Math.round(value / 60);
  return `${minutes} min`;
}

function toRFC3339(localValue) {
  if (!localValue) {
    return "";
  }
  const date = new Date(localValue);
  return date.toISOString();
}

function itemCard(title, subtitle) {
  const wrapper = document.createElement("div");
  wrapper.className = "item";
  const info = document.createElement("div");
  info.innerHTML = `<strong>${title}</strong><br><small>${subtitle}</small>`;
  wrapper.appendChild(info);
  return wrapper;
}

function baseUrl() {
  return analyticsBase.value.replace(/\/$/, "");
}

function queryParams() {
  const tenantId = tenantIdInput.value.trim();
  const branchId = branchIdInput.value.trim();
  const serviceId = serviceIdInput.value.trim();
  return {
    tenantId,
    branchId,
    serviceId,
    from: toRFC3339(fromDate.value),
    to: toRFC3339(toDate.value),
  };
}

async function api(path) {
  const response = await fetch(`${baseUrl()}${path}`);
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    const msg = body?.error?.message || body?.error?.code || response.statusText;
    throw new Error(msg);
  }
  return response.json();
}

async function refreshKPIs() {
  const params = queryParams();
  if (!params.tenantId || !params.branchId || !params.serviceId) {
    throw new Error("Tenant, branch, and service IDs are required.");
  }
  const search = new URLSearchParams({
    tenant_id: params.tenantId,
    branch_id: params.branchId,
    service_id: params.serviceId,
    from: params.from,
    to: params.to,
  });
  const data = await api(`/api/analytics/kpis?${search.toString()}`);
  avgWait.textContent = formatSeconds(data.avg_wait_seconds);
  avgService.textContent = formatSeconds(data.avg_service_seconds);
  totalCount.textContent = data.count ?? "-";
}

async function refreshRealtime() {
  const params = queryParams();
  const search = new URLSearchParams({
    tenant_id: params.tenantId,
    branch_id: params.branchId,
    service_id: params.serviceId,
  });
  const data = await api(`/api/analytics/realtime?${search.toString()}`);
  queueLength.textContent = data.queue_length ?? "-";
  servingCount.textContent = data.serving ?? "-";
}

async function refreshReports() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId) {
    reportList.textContent = "Tenant ID required.";
    return;
  }
  const data = await api(`/api/analytics/reports?tenant_id=${tenantId}`);
  renderList(reportList, data, (report) =>
    itemCard(
      `${report.channel} · ${report.cron}`,
      `${report.recipient} · ${report.report_id}`
    )
  );
}

async function refreshAnomalies() {
  const tenantId = tenantIdInput.value.trim();
  if (!tenantId) {
    anomalyList.textContent = "Tenant ID required.";
    return;
  }
  const data = await api(`/api/analytics/anomalies?tenant_id=${tenantId}`);
  renderList(anomalyList, data, (anomaly) =>
    itemCard(
      `${anomaly.type} · ${Math.round(anomaly.value)} > ${Math.round(anomaly.threshold)}`,
      `${anomaly.branch_id} · ${anomaly.service_id}`
    )
  );
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

async function refreshAll() {
  setStatus("Loading...");
  setAlert("");
  try {
    await Promise.all([
      refreshKPIs(),
      refreshRealtime(),
      refreshReports(),
      refreshAnomalies(),
    ]);
    setStatus("Live");
  } catch (err) {
    setStatus("Error");
    setAlert(err.message);
  }
}

function setRange(hours) {
  const end = new Date();
  const start = new Date(end.getTime() - hours * 3600 * 1000);
  fromDate.value = start.toISOString().slice(0, 16);
  toDate.value = end.toISOString().slice(0, 16);
}

async function createReport() {
  const tenantId = tenantIdInput.value.trim();
  const branchId = branchIdInput.value.trim();
  const serviceId = serviceIdInput.value.trim();
  if (!tenantId || !branchId || !serviceId || !reportCron.value.trim() || !reportRecipient.value.trim()) {
    setAlert("Tenant, branch, service, cron, and recipient are required.");
    return;
  }
  const payload = {
    tenant_id: tenantId,
    branch_id: branchId,
    service_id: serviceId,
    cron: reportCron.value.trim(),
    channel: reportChannel.value,
    recipient: reportRecipient.value.trim(),
  };
  const response = await fetch(`${baseUrl()}/api/analytics/reports`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => null);
    const msg = body?.error?.message || body?.error?.code || response.statusText;
    setAlert(msg);
    return;
  }
  reportCron.value = "";
  reportRecipient.value = "";
  await refreshReports();
}

function exportCsv() {
  const params = queryParams();
  if (!params.tenantId || !params.branchId || !params.serviceId) {
    setAlert("Tenant, branch, and service IDs are required for export.");
    return;
  }
  const search = new URLSearchParams({
    tenant_id: params.tenantId,
    branch_id: params.branchId,
    service_id: params.serviceId,
    from: params.from,
    to: params.to,
  });
  window.location.href = `${baseUrl()}/api/analytics/export?${search.toString()}`;
}

function toggleAutoRefresh() {
  if (autoTimer) {
    clearInterval(autoTimer);
    autoTimer = null;
    toggleAuto.textContent = "Auto Refresh: Off";
    return;
  }
  autoTimer = setInterval(() => {
    refreshAll().catch(() => {});
  }, 15000);
  toggleAuto.textContent = "Auto Refresh: On";
}

refreshBtn.addEventListener("click", () => {
  refreshAll().catch(() => {});
});

toggleAuto.addEventListener("click", () => {
  toggleAutoRefresh();
});

exportBtn.addEventListener("click", () => {
  exportCsv();
});

document.querySelectorAll("[data-range]").forEach((btn) => {
  btn.addEventListener("click", () => {
    setRange(Number(btn.dataset.range));
    refreshAll().catch(() => {});
  });
});

document.getElementById("createReport").addEventListener("click", () => {
  createReport().catch(() => {});
});

setRange(24);
setStatus("Idle");
