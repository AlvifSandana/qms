const state = {
  queueBase: "http://localhost:8080",
  tenantId: "",
  branchId: "",
  serviceId: "",
  lastTicket: null,
};

const queueBaseInput = document.getElementById("queueBase");
const tenantInput = document.getElementById("tenantId");
const branchInput = document.getElementById("branchId");
const deviceInput = document.getElementById("deviceId");
const serviceSelect = document.getElementById("serviceSelect");
const loadBtn = document.getElementById("loadBtn");
const issueBtn = document.getElementById("issueBtn");
const printBtn = document.getElementById("printBtn");
const syncBtn = document.getElementById("syncBtn");
const ticketPanel = document.getElementById("ticketPanel");
const status = document.getElementById("status");
const langSelect = document.getElementById("langSelect");
const phoneToggle = document.getElementById("phoneToggle");
const phoneLabel = document.getElementById("phoneLabel");
const phoneInput = document.getElementById("phoneInput");
const contrastBtn = document.getElementById("contrastBtn");
const fontBtn = document.getElementById("fontBtn");
const appointmentInput = document.getElementById("appointmentInput");
const checkinBtn = document.getElementById("checkinBtn");
const connState = document.getElementById("connState");
const offlineCount = document.getElementById("offlineCount");
const configVersionEl = document.getElementById("configVersion");
const lastSyncEl = document.getElementById("lastSync");
const configPreview = document.getElementById("configPreview");

let idleTimer;
const idleTimeoutMs = 60000;

const translations = {
  id: {
    ready: "Siap",
    loadFailed: "Gagal memuat layanan",
    loadOk: "Layanan tersedia",
    needConfig: "URL, tenant, dan branch wajib diisi",
    selectService: "Pilih layanan dulu",
    issued: "Tiket terbit",
    issueFailed: "Gagal membuat tiket",
    offline: "Offline - tiket lokal",
    noTicket: "Belum ada tiket",
    printFailed: "Cetak gagal, gunakan QR di layar",
    syncOk: "Sinkron berhasil",
    syncFail: "Sinkron gagal",
    checkinFail: "Check-in gagal",
  },
  en: {
    ready: "Ready",
    loadFailed: "Failed to load services",
    loadOk: "Services loaded",
    needConfig: "Queue URL, tenant, and branch required",
    selectService: "Select a service first",
    issued: "Ticket issued",
    issueFailed: "Failed to issue ticket",
    offline: "Offline - local ticket",
    noTicket: "No ticket issued yet",
    printFailed: "Print failed, use on-screen QR",
    syncOk: "Sync complete",
    syncFail: "Sync failed",
    checkinFail: "Check-in failed",
  },
};

let configVersion = 0;

function setStatus(text) {
  status.textContent = text;
}

function setConnState(text) {
  connState.textContent = text;
}

function updateOfflineCount() {
  const stored = JSON.parse(localStorage.getItem("offlineTickets") || "[]");
  offlineCount.textContent = stored.length;
}

function updateConfigPreview(payload) {
  configPreview.textContent = JSON.stringify(payload || {}, null, 2);
}

function t(key) {
  const lang = langSelect.value || "id";
  return translations[lang][key] || key;
}

function resetIdle() {
  clearTimeout(idleTimer);
  idleTimer = setTimeout(() => {
    ticketPanel.innerHTML = `<p class="hint">${t("noTicket")}</p>`;
    setStatus(t("ready"));
  }, idleTimeoutMs);
}

function uuidv4() {
  if (crypto?.randomUUID) {
    return crypto.randomUUID();
  }
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  return [...bytes].map((b, i) => (i === 4 || i === 6 || i === 8 || i === 10 ? "-" : "") + b.toString(16).padStart(2, "0")).join("");
}

function parseAppointmentValue(value) {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }
  const match = trimmed.match(/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}/);
  return match ? match[0] : trimmed;
}

function updateServiceSelect(services) {
  serviceSelect.innerHTML = "";
  const empty = document.createElement("option");
  empty.value = "";
  empty.textContent = "Select service";
  serviceSelect.appendChild(empty);

  services.forEach((service) => {
    const option = document.createElement("option");
    option.value = service.service_id;
    option.textContent = `${service.name} (${service.code})`;
    serviceSelect.appendChild(option);
  });
}

async function loadServices() {
  state.queueBase = queueBaseInput.value.trim();
  state.tenantId = tenantInput.value.trim();
  state.branchId = branchInput.value.trim();
  if (!state.queueBase || !state.tenantId || !state.branchId) {
    setStatus(t("needConfig"));
    return;
  }

  const response = await fetch(`${state.queueBase}/api/services?tenant_id=${state.tenantId}&branch_id=${state.branchId}`);
  if (!response.ok) {
    setStatus(t("loadFailed"));
    return;
  }
  const services = await response.json();
  updateServiceSelect(services);
  setStatus(t("loadOk"));
  setConnState("Online");
}

function renderTicket(ticket) {
  if (!ticket) {
    ticketPanel.innerHTML = `<p class="hint">${t("noTicket")}</p>`;
    printBtn.disabled = true;
    return;
  }
  const qrValue = `${state.queueBase}/ticket/${ticket.ticket_id}`;
  ticketPanel.innerHTML = `
    <h3>${ticket.ticket_number}</h3>
    <p>Status: ${ticket.status}</p>
    <div class="qr">QR: ${qrValue}</div>
  `;
  printBtn.disabled = false;
}

async function issueTicket() {
  state.serviceId = serviceSelect.value;
  if (!state.serviceId) {
    setStatus(t("selectService"));
    return;
  }
  const payload = {
    request_id: uuidv4(),
    tenant_id: state.tenantId,
    branch_id: state.branchId,
    service_id: state.serviceId,
    channel: "kiosk",
    phone: phoneToggle.value === "on" ? phoneInput.value.trim() : "",
  };
  try {
    const response = await fetch(`${state.queueBase}/api/tickets`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!response.ok) {
      throw new Error("server error");
    }
    const ticket = await response.json();
    state.lastTicket = ticket;
    renderTicket(ticket);
    setStatus(`${t("issued")} ${ticket.ticket_number}`);
  } catch (err) {
    const localTicket = {
      ticket_id: payload.request_id,
      ticket_number: `L-${Date.now().toString().slice(-6)}`,
      status: "waiting",
    };
    state.lastTicket = localTicket;
    renderTicket(localTicket);
    setStatus(t("offline"));
    queueOfflineTicket(payload);
  }
}

async function checkInAppointment() {
  const appointmentId = parseAppointmentValue(appointmentInput.value);
  if (!appointmentId) {
    setStatus("Appointment ID required");
    return;
  }
  appointmentInput.value = appointmentId;
  const payload = {
    request_id: uuidv4(),
    tenant_id: state.tenantId,
    branch_id: state.branchId,
    appointment_id: appointmentId,
  };
  try {
    const response = await fetch(`${state.queueBase}/api/appointments/checkin`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!response.ok) {
      throw new Error("checkin failed");
    }
    const ticket = await response.json();
    state.lastTicket = ticket;
    renderTicket(ticket);
    setStatus(`Checked in ${ticket.ticket_number}`);
  } catch (err) {
    setStatus(t("checkinFail"));
  }
}

function queueOfflineTicket(payload) {
  const stored = JSON.parse(localStorage.getItem("offlineTickets") || "[]");
  stored.push(payload);
  localStorage.setItem("offlineTickets", JSON.stringify(stored));
  updateOfflineCount();
}

async function syncOfflineTickets() {
  const stored = JSON.parse(localStorage.getItem("offlineTickets") || "[]");
  if (!stored.length) {
    return;
  }
  const remaining = [];
  for (const payload of stored) {
    try {
      const response = await fetch(`${state.queueBase}/api/tickets`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      if (!response.ok) {
        remaining.push(payload);
      }
    } catch (err) {
      remaining.push(payload);
    }
  }
  localStorage.setItem("offlineTickets", JSON.stringify(remaining));
  updateOfflineCount();
}

async function healthCheck() {
  try {
    const response = await fetch(`${state.queueBase}/healthz`);
    if (!response.ok) {
      throw new Error("offline");
    }
    setStatus(t("ready"));
    setConnState("Online");
    await syncOfflineTickets();
    lastSyncEl.textContent = new Date().toLocaleTimeString();
    sendDeviceStatus("online");
  } catch (err) {
    setStatus(t("offline"));
    setConnState("Offline");
    sendDeviceStatus("offline");
  }
}

async function sendDeviceStatus(statusText) {
  const deviceId = deviceInput.value.trim();
  if (!deviceId) {
    return;
  }
  try {
    await fetch(`${state.queueBase.replace("8080", "8083")}/api/devices/status`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ device_id: deviceId, status: statusText }),
    });
  } catch (err) {
    return;
  }
}

async function fetchDeviceConfig() {
  const deviceId = deviceInput.value.trim();
  if (!deviceId) {
    return;
  }
  const response = await fetch(`${state.queueBase.replace("8080", "8083")}/api/devices/config?device_id=${deviceId}`);
  if (response.status === 204) {
    return;
  }
  if (!response.ok) {
    return;
  }
  const data = await response.json();
  if (data.version && data.version <= configVersion) {
    return;
  }
  configVersion = data.version || configVersion;
  configVersionEl.textContent = String(configVersion);
  updateConfigPreview(data.payload || {});
  applyConfig(data.payload || {});
}

function applyConfig(payload) {
  if (payload.language) {
    langSelect.value = payload.language;
  }
  if (payload.phone_enabled === true) {
    phoneToggle.value = "on";
    phoneLabel.classList.remove("hidden");
  }
  if (payload.phone_enabled === false) {
    phoneToggle.value = "off";
    phoneLabel.classList.add("hidden");
  }
  if (payload.high_contrast === true) {
    document.body.classList.add("high-contrast");
  }
  if (payload.high_contrast === false) {
    document.body.classList.remove("high-contrast");
  }
  if (payload.large_text === true) {
    document.body.classList.add("large-text");
  }
  if (payload.large_text === false) {
    document.body.classList.remove("large-text");
  }
  renderTicket(state.lastTicket);
  setStatus(t("ready"));
}

loadBtn.addEventListener("click", () => {
  loadServices().catch(() => setStatus(t("loadFailed")));
});

syncBtn.addEventListener("click", () => {
  healthCheck()
    .then(() => setStatus(t("syncOk")))
    .catch(() => setStatus(t("syncFail")));
});

issueBtn.addEventListener("click", () => {
  issueTicket().catch(() => setStatus(t("issueFailed")));
});

checkinBtn.addEventListener("click", () => {
  checkInAppointment().catch(() => setStatus("Check-in failed"));
});

printBtn.addEventListener("click", () => {
  try {
    window.print();
  } catch (err) {
    setStatus(t("printFailed"));
  }
});

langSelect.addEventListener("change", () => {
  renderTicket(state.lastTicket);
  setStatus(t("ready"));
});

phoneToggle.addEventListener("change", () => {
  if (phoneToggle.value === "on") {
    phoneLabel.classList.remove("hidden");
  } else {
    phoneLabel.classList.add("hidden");
  }
});

appointmentInput.addEventListener("change", () => {
  appointmentInput.value = parseAppointmentValue(appointmentInput.value);
});

appointmentInput.addEventListener("paste", () => {
  setTimeout(() => {
    appointmentInput.value = parseAppointmentValue(appointmentInput.value);
  }, 0);
});

contrastBtn.addEventListener("click", () => {
  document.body.classList.toggle("high-contrast");
});

fontBtn.addEventListener("click", () => {
  document.body.classList.toggle("large-text");
});

["click", "touchstart", "keydown"].forEach((evt) => {
  document.addEventListener(evt, resetIdle);
});

setInterval(() => {
  if (state.queueBase) {
    healthCheck().catch(() => setStatus(t("offline")));
    fetchDeviceConfig().catch(() => {});
  }
}, 10000);

renderTicket(null);
resetIdle();
updateOfflineCount();
