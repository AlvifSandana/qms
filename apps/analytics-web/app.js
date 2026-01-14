const analyticsBase = document.getElementById("analyticsBase");
const sessionIdInput = document.getElementById("sessionId");
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
const activeCounters = document.getElementById("activeCounters");
const busyCounters = document.getElementById("busyCounters");

const reportCron = document.getElementById("reportCron");
const reportChannel = document.getElementById("reportChannel");
const reportRecipient = document.getElementById("reportRecipient");
const reportList = document.getElementById("reportList");
const previewBtn = document.getElementById("previewReport");
const clearPreviewBtn = document.getElementById("clearPreview");
const previewTable = document.getElementById("previewTable");

const anomalyList = document.getElementById("anomalyList");
const anomalyScope = document.getElementById("anomalyScope");
const anomalyMin = document.getElementById("anomalyMin");
const anomalyMax = document.getElementById("anomalyMax");
const anomalyMeta = document.getElementById("anomalyMeta");
const refreshAnomaliesBtn = document.getElementById("refreshAnomalies");

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

function authHeaders(extra = {}) {
  const headers = { ...extra };
  const sessionId = sessionIdInput.value.trim();
  if (sessionId) {
    headers.Authorization = `Bearer ${sessionId}`;
  }
  return headers;
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
  const response = await fetch(`${baseUrl()}${path}`, {
    headers: authHeaders(),
  });
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
  activeCounters.textContent = data.active_counters ?? "-";
  busyCounters.textContent = data.busy_counters ?? "-";
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
      `${report.recipient} · ${report.report_id} · ${report.last_sent_at || "never"}`
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
  const params = queryParams();
  const scoped = anomalyScope.value === "on";
  const minRatio = Number(anomalyMin.value) || 1;
  const maxItems = Number(anomalyMax.value) || 20;
  const filtered = (data || []).filter((anomaly) => {
    const ratio = anomaly.threshold ? anomaly.value / anomaly.threshold : 0;
    if (ratio < minRatio) {
      return false;
    }
    if (scoped) {
      if (params.branchId && anomaly.branch_id !== params.branchId) {
        return false;
      }
      if (params.serviceId && anomaly.service_id !== params.serviceId) {
        return false;
      }
    }
    return true;
  }).slice(0, maxItems);
  anomalyMeta.textContent = `${filtered.length} of ${(data || []).length} anomalies`;
  renderList(anomalyList, filtered, (anomaly) => {
    const ratio = anomaly.threshold ? (anomaly.value / anomaly.threshold) : 0;
    return itemCard(
      `${anomaly.type} · ${Math.round(anomaly.value)} > ${Math.round(anomaly.threshold)} (${ratio.toFixed(1)}x)`,
      `${anomaly.branch_id} · ${anomaly.service_id}`
    );
  });
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
    headers: authHeaders({ "Content-Type": "application/json" }),
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

async function exportCsv() {
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
  const response = await fetch(`${baseUrl()}/api/analytics/export?${search.toString()}`, {
    headers: authHeaders(),
  });
  if (!response.ok) {
    setAlert("Failed to export CSV.");
    return;
  }
  const blob = await response.blob();
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = "report.csv";
  link.click();
  URL.revokeObjectURL(link.href);
}

async function previewCsv() {
  const params = queryParams();
  if (!params.tenantId || !params.branchId || !params.serviceId) {
    setAlert("Tenant, branch, and service IDs are required for preview.");
    return;
  }
  const search = new URLSearchParams({
    tenant_id: params.tenantId,
    branch_id: params.branchId,
    service_id: params.serviceId,
    from: params.from,
    to: params.to,
  });
  const response = await fetch(`${baseUrl()}/api/analytics/export?${search.toString()}`, {
    headers: authHeaders(),
  });
  if (!response.ok) {
    setAlert("Failed to load preview.");
    return;
  }
  const csv = await response.text();
  renderPreview(csv);
}

function clearPreview() {
  previewTable.innerHTML = "";
}

function renderPreview(csv) {
  const rows = parseCsv(csv);
  if (rows.length === 0) {
    previewTable.textContent = "No data.";
    return;
  }
  const [header, ...body] = rows;
  const table = document.createElement("table");
  const thead = document.createElement("thead");
  const headRow = document.createElement("tr");
  header.forEach((cell) => {
    const th = document.createElement("th");
    th.textContent = cell;
    headRow.appendChild(th);
  });
  thead.appendChild(headRow);
  table.appendChild(thead);

  const tbody = document.createElement("tbody");
  body.slice(0, 50).forEach((row) => {
    const tr = document.createElement("tr");
    row.forEach((cell) => {
      const td = document.createElement("td");
      td.textContent = cell;
      tr.appendChild(td);
    });
    tbody.appendChild(tr);
  });
  table.appendChild(tbody);
  previewTable.innerHTML = "";
  previewTable.appendChild(table);
}

function parseCsv(csv) {
  if (!csv) {
    return [];
  }
  return csv.trim().split("\n").map((line) => line.split(","));
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

previewBtn.addEventListener("click", () => {
  previewCsv().catch(() => {});
});

clearPreviewBtn.addEventListener("click", () => {
  clearPreview();
});

refreshAnomaliesBtn.addEventListener("click", () => {
  refreshAnomalies().catch(() => {});
});

setRange(24);
setStatus("Idle");
