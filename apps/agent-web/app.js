const state = {
  sessionId: null,
  branches: [],
  services: [],
  counters: [],
  role: "",
  authBase: "http://localhost:8081",
  queueBase: "http://localhost:8080",
  tenantId: "",
  branchId: "",
  serviceId: "",
  supervisor: false,
};

const authBaseInput = document.getElementById("authBase");
const queueBaseInput = document.getElementById("queueBase");
const tenantIdInput = document.getElementById("tenantId");
const emailInput = document.getElementById("email");
const passwordInput = document.getElementById("password");
const loginBtn = document.getElementById("loginBtn");
const loginHint = document.getElementById("loginHint");
const branchSelect = document.getElementById("branchSelect");
const counterSelect = document.getElementById("counterSelect");
const counterInput = document.getElementById("counterInput");
const addCounterBtn = document.getElementById("addCounter");
const removeCounterBtn = document.getElementById("removeCounter");
const presenceSelect = document.getElementById("presenceSelect");
const savePresenceBtn = document.getElementById("savePresence");
const supervisorToggle = document.getElementById("supervisorToggle");
const supervisorPanel = document.getElementById("supervisorPanel");
const counterList = document.getElementById("counterList");
const serviceSelect = document.getElementById("serviceSelect");
const refreshBtn = document.getElementById("refreshBtn");
const ticketList = document.getElementById("ticketList");
const status = document.getElementById("status");
const activeTicket = document.getElementById("activeTicket");
const callNextBtn = document.getElementById("callNextBtn");
const recallBtn = document.getElementById("recallBtn");
const startBtn = document.getElementById("startBtn");
const completeBtn = document.getElementById("completeBtn");
const holdBtn = document.getElementById("holdBtn");
const unholdBtn = document.getElementById("unholdBtn");
const noShowBtn = document.getElementById("noShowBtn");
const cancelBtn = document.getElementById("cancelBtn");
const transferBtn = document.getElementById("transferBtn");
const transferSelect = document.getElementById("transferService");
const alertBox = document.getElementById("alert");
const logoutBtn = document.getElementById("logoutBtn");
let socket = null;
let reconnectDelay = 1000;
let reconnectTimer = null;
let availableCounters = [];

function setStatus(text) {
  status.textContent = text;
}

function setAlert(message) {
  if (!message) {
    alertBox.hidden = true;
    alertBox.textContent = "";
    return;
  }
  alertBox.textContent = message;
  alertBox.hidden = false;
  clearTimeout(setAlert.timer);
  setAlert.timer = setTimeout(() => {
    alertBox.hidden = true;
  }, 5000);
}

function updateSelect(select, items, placeholder) {
  select.innerHTML = "";
  const empty = document.createElement("option");
  empty.value = "";
  empty.textContent = placeholder;
  select.appendChild(empty);

  items.forEach((item) => {
    const option = document.createElement("option");
    option.value = item;
    option.textContent = item;
    select.appendChild(option);
  });
}

function updateCounterSelect() {
  counterSelect.innerHTML = "";
  const empty = document.createElement("option");
  empty.value = "";
  empty.textContent = "Select counter";
  counterSelect.appendChild(empty);

  state.counters.forEach((counter) => {
    const option = document.createElement("option");
    option.value = counter.counter_id;
    option.textContent = `${counter.name || "Counter"} (${counter.counter_id.slice(0, 6)})`;
    option.dataset.status = counter.status || "";
    counterSelect.appendChild(option);
  });
}

function setPresenceFromSelection() {
  const status = counterSelect.selectedOptions[0]?.dataset?.status || "available";
  if (status === "active") {
    presenceSelect.value = "available";
    return;
  }
  if (status) {
    presenceSelect.value = status;
  }
}

function setRoleGate(role) {
  const allowed = role === "agent" || role === "supervisor";
  loginHint.textContent = allowed ? "" : "Role not permitted for agent console.";
  document.querySelectorAll("button, select, input").forEach((el) => {
    if (el.id === "authBase" || el.id === "queueBase" || el.id === "tenantId" || el.id === "email" || el.id === "password" || el.id === "loginBtn") {
      return;
    }
    el.disabled = !allowed;
  });
  supervisorToggle.disabled = role !== "supervisor";
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

function updateServiceSelect(services) {
  serviceSelect.innerHTML = "";
  const empty = document.createElement("option");
  empty.value = "";
  empty.textContent = "Select service";
  serviceSelect.appendChild(empty);

  transferSelect.innerHTML = "";
  const transferEmpty = document.createElement("option");
  transferEmpty.value = "";
  transferEmpty.textContent = "Select target";
  transferSelect.appendChild(transferEmpty);

  services.forEach((service) => {
    const option = document.createElement("option");
    option.value = service.service_id;
    option.textContent = `${service.name} (${service.code})`;
    option.dataset.sla = service.sla_minutes;
    serviceSelect.appendChild(option);

    const transferOption = document.createElement("option");
    transferOption.value = service.service_id;
    transferOption.textContent = `${service.name} (${service.code})`;
    transferSelect.appendChild(transferOption);
  });
}

async function login() {
  loginHint.textContent = "";
  setAlert("");
  state.authBase = authBaseInput.value.trim();
  state.queueBase = queueBaseInput.value.trim();
  state.tenantId = tenantIdInput.value.trim();

  const payload = {
    tenant_id: state.tenantId,
    email: emailInput.value.trim(),
    password: passwordInput.value,
  };

  const response = await fetch(`${state.authBase}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    loginHint.textContent = "Login failed.";
    setStatus("Login failed");
    setAlert("Login failed. Check credentials, tenant ID, and branch access.");
    return;
  }

  const data = await response.json();
  state.sessionId = data.session_id;
  state.branches = data.branches || [];
  state.services = data.services || [];
  state.role = data.user?.role || "";

  updateSelect(branchSelect, state.branches, "Select branch");
  setStatus("Logged in");
  loginHint.textContent = "Login success. Pick branch & service.";
  setAlert("");
  setRoleGate(state.role);
  supervisorToggle.value = state.role === "supervisor" ? supervisorToggle.value : "off";
  state.supervisor = supervisorToggle.value === "on" && state.role === "supervisor";
  connectRealtime();
}

async function loadServices() {
  const branchId = branchSelect.value;
  if (!branchId) {
    updateServiceSelect([]);
    return;
  }
  state.branchId = branchId;

  const response = await fetch(`${state.queueBase}/api/services?tenant_id=${state.tenantId}&branch_id=${branchId}`);
  if (!response.ok) {
    setStatus("Failed to load services");
    return;
  }
  const services = await response.json();
  const filtered = services.filter((svc) => state.services.length === 0 || state.services.includes(svc.service_id));
  updateServiceSelect(filtered);
  connectRealtime();
}

async function loadCounters() {
  const branchId = branchSelect.value;
  if (!branchId) {
    return;
  }
  const response = await fetch(`${state.queueBase}/api/counters?tenant_id=${state.tenantId}&branch_id=${branchId}`);
  if (!response.ok) {
    setStatus("Failed to load counters");
    return;
  }
  availableCounters = await response.json();
  if (state.counters.length === 0) {
    state.counters = availableCounters;
  }
  updateCounterSelect();
  setPresenceFromSelection();
}

function estimateEta(position, slaMinutes) {
  if (!slaMinutes || Number.isNaN(slaMinutes)) {
    return "ETA: n/a";
  }
  return `ETA ~${position * slaMinutes}m`;
}

async function refreshQueue() {
  const serviceId = serviceSelect.value;
  if (!serviceId) {
    ticketList.innerHTML = "<p class=\"hint\">Pick a service to see tickets.</p>";
    return;
  }
  state.serviceId = serviceId;
  const response = await fetch(`${state.queueBase}/api/tickets/snapshot?tenant_id=${state.tenantId}&branch_id=${state.branchId}&service_id=${serviceId}`);
  if (!response.ok) {
    setStatus("Failed to load queue");
    return;
  }
  const tickets = await response.json();
  const sla = Number(serviceSelect.selectedOptions[0]?.dataset?.sla || 0);

  ticketList.innerHTML = "";
  if (tickets.length === 0) {
    ticketList.innerHTML = "<p class=\"hint\">No waiting tickets.</p>";
    return;
  }

  tickets.forEach((ticket, index) => {
    const card = document.createElement("div");
    card.className = "ticket";
    card.innerHTML = `
      <div>
        <strong>${ticket.ticket_number}</strong>
        <div><span>Status: ${ticket.status}</span></div>
      </div>
      <span>${estimateEta(index + 1, sla)}</span>
    `;
    ticketList.appendChild(card);
  });
  setStatus(`Loaded ${tickets.length} tickets`);
}

function connectRealtime() {
  if (!state.queueBase || !state.tenantId) {
    return;
  }
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (socket) {
    socket.close();
  }
  socket = new SockJS(`${state.queueBase}/realtime`);
  socket.onopen = () => {
    reconnectDelay = 1000;
    const msg = {
      action: "subscribe",
      tenant_id: state.tenantId,
      branch_id: branchSelect.value || "",
      service_id: serviceSelect.value || "",
    };
    socket.send(JSON.stringify(msg));
  };
  socket.onmessage = (event) => {
    try {
      const parsed = JSON.parse(event.data);
      handleRealtimeEvent(parsed);
    } catch (err) {
      return;
    }
  };
  socket.onclose = () => {
    scheduleReconnect();
  };
}

function scheduleReconnect() {
  if (reconnectTimer) {
    return;
  }
  const delay = reconnectDelay;
  reconnectDelay = Math.min(reconnectDelay * 2, 30000);
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    connectRealtime();
  }, delay);
}

function sendUnsubscribe() {
  if (!socket || socket.readyState !== SockJS.OPEN) {
    return;
  }
  socket.send(JSON.stringify({ action: "unsubscribe" }));
}

function handleRealtimeEvent(event) {
  if (!event || !event.type) {
    return;
  }
  if (!event.type.startsWith("ticket.")) {
    return;
  }
  const payload = event.payload || {};
  if (payload.branch_id && branchSelect.value && payload.branch_id !== branchSelect.value) {
    return;
  }
  if (payload.service_id && serviceSelect.value && payload.service_id !== serviceSelect.value) {
    return;
  }
  refreshQueue().catch(() => setStatus("Failed to load queue"));
  loadActiveTicket().catch(() => setStatus("Failed to load active ticket"));
}

function renderActive(ticket) {
  if (!ticket) {
    activeTicket.classList.add("empty");
    activeTicket.innerHTML = "No active ticket.";
    activeTicket.removeAttribute("data-ticket-id");
    recallBtn.disabled = true;
    startBtn.disabled = true;
    completeBtn.disabled = true;
    holdBtn.disabled = true;
    unholdBtn.disabled = true;
    noShowBtn.disabled = true;
    cancelBtn.disabled = true;
    transferBtn.disabled = true;
    return;
  }
  activeTicket.classList.remove("empty");
  activeTicket.dataset.ticketId = ticket.ticket_id;
  activeTicket.innerHTML = `
    <div>
      <strong>${ticket.ticket_number}</strong>
      <div><span>Status: ${ticket.status}</span></div>
    </div>
    <span>${ticket.called_at ? "Called" : "Active"}</span>
  `;

  recallBtn.disabled = ticket.status !== "called";
  startBtn.disabled = ticket.status !== "called";
  completeBtn.disabled = ticket.status !== "serving";
  holdBtn.disabled = ticket.status !== "waiting" && ticket.status !== "called";
  unholdBtn.disabled = ticket.status !== "held";
  noShowBtn.disabled = ticket.status !== "called";
  cancelBtn.disabled = ticket.status !== "waiting";
  transferBtn.disabled = ticket.status !== "waiting" && ticket.status !== "called" && ticket.status !== "serving";
}

async function loadActiveTicket() {
  const branchId = branchSelect.value;
  const counterId = counterSelect.value;
  if (!branchId || !counterId) {
    renderActive(null);
    return;
  }
  state.branchId = branchId;
  const response = await fetch(`${state.queueBase}/api/tickets/active?tenant_id=${state.tenantId}&branch_id=${branchId}&counter_id=${counterId}`);
  if (response.status === 204) {
    renderActive(null);
    return;
  }
  if (!response.ok) {
    setStatus("Failed to load active ticket");
    setAlert("Failed to load active ticket. Try refresh.");
    return;
  }
  const ticket = await response.json();
  renderActive(ticket);
}

async function performAction(action) {
  const branchId = branchSelect.value;
  const counterId = counterSelect.value;
  if (!branchId || !counterId) {
    setStatus("Branch and counter required");
    return;
  }
  const current = activeTicket.querySelector("strong");
  if (!current) {
    return;
  }
  const ticketId = activeTicket.dataset.ticketId;
  if (!ticketId) {
    return;
  }
  const payload = {
    request_id: uuidv4(),
    tenant_id: state.tenantId,
    branch_id: branchId,
    counter_id: counterId,
  };
  const response = await fetch(`${state.queueBase}/api/tickets/${ticketId}/actions/${action}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    setStatus(`Action failed: ${action}`);
    setAlert(`Action failed: ${action}. Please retry.`);
    return;
  }
  setStatus(`Action ok: ${action}`);
  setAlert("");
  await loadActiveTicket();
  await refreshQueue();
}

async function transferTicket() {
  const branchId = branchSelect.value;
  const counterId = counterSelect.value;
  const serviceId = transferSelect.value;
  const ticketId = activeTicket.dataset.ticketId;
  if (!branchId || !counterId || !serviceId || !ticketId) {
    setStatus("Select target service and active ticket");
    return;
  }

  const payload = {
    request_id: uuidv4(),
    tenant_id: state.tenantId,
    branch_id: branchId,
    counter_id: counterId,
    to_service_id: serviceId,
  };

  const response = await fetch(`${state.queueBase}/api/tickets/${ticketId}/actions/transfer`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    setStatus("Transfer failed");
    setAlert("Transfer failed. Check mapping or permissions.");
    return;
  }
  setStatus("Transfer ok");
  setAlert("");
  await loadActiveTicket();
  await refreshQueue();
}

async function callNext() {
  const branchId = branchSelect.value;
  const counterId = counterSelect.value;
  const serviceId = serviceSelect.value;
  if (!branchId || !counterId || !serviceId) {
    setStatus("Branch, counter, and service required");
    return;
  }
  const payload = {
    request_id: uuidv4(),
    tenant_id: state.tenantId,
    branch_id: branchId,
    service_id: serviceId,
    counter_id: counterId,
  };
  const response = await fetch(`${state.queueBase}/api/tickets/actions/call-next`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  if (response.status === 409) {
    setStatus("No tickets available");
    renderActive(null);
    setAlert("No tickets available for the selected service.");
    ticketList.innerHTML = "<p class=\"hint\">No waiting tickets.</p>";
    return;
  }

  if (!response.ok) {
    setStatus("Call next failed");
    setAlert("Call next failed. Check service mapping.");
    return;
  }

  const ticket = await response.json();
  renderActive(ticket);
  setStatus(`Called ${ticket.ticket_number}`);
  setAlert("");
  await refreshQueue();
}

function addCounter() {
  const counterId = counterInput.value.trim();
  if (!counterId) {
    setAlert("Counter ID is required.");
    return;
  }
  const existing = state.counters.find((counter) => counter.counter_id === counterId);
  if (existing) {
    setAlert("Counter already added.");
    return;
  }
  const matched = availableCounters.find((counter) => counter.counter_id === counterId);
  state.counters.push(matched || { counter_id: counterId, name: "Counter", status: "available" });
  updateCounterSelect();
  counterSelect.value = counterId;
  counterInput.value = "";
  setAlert("");
}

function removeCounter() {
  const counterId = counterSelect.value;
  if (!counterId) {
    setAlert("Select a counter to remove.");
    return;
  }
  state.counters = state.counters.filter((counter) => counter.counter_id !== counterId);
  updateCounterSelect();
  setPresenceFromSelection();
}

async function savePresence() {
  const branchId = branchSelect.value;
  const counterId = counterSelect.value;
  if (!branchId || !counterId) {
    setAlert("Select branch and counter first.");
    return;
  }
  const payload = {
    tenant_id: state.tenantId,
    branch_id: branchId,
    status: presenceSelect.value,
  };
  const response = await fetch(`${state.queueBase}/api/counters/${counterId}/status`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    setAlert("Failed to update presence.");
    return;
  }
  const counter = state.counters.find((item) => item.counter_id === counterId);
  if (counter) {
    counter.status = presenceSelect.value;
  }
  setStatus(`Presence set to ${presenceSelect.value}`);
  setAlert("");
}

async function loadSupervisorPanel() {
  if (!state.supervisor || state.role !== "supervisor") {
    supervisorPanel.hidden = true;
    return;
  }
  supervisorPanel.hidden = false;
  const branchId = branchSelect.value;
  if (!branchId) {
    counterList.textContent = "Select branch to view counters.";
    return;
  }
  await loadCounters();
  counterList.innerHTML = "";
  if (availableCounters.length === 0) {
    counterList.textContent = "No counters found.";
    return;
  }
  for (const counter of availableCounters) {
    const response = await fetch(`${state.queueBase}/api/tickets/active?tenant_id=${state.tenantId}&branch_id=${branchId}&counter_id=${counter.counter_id}`);
    let activeLabel = "No active ticket";
    if (response.ok && response.status !== 204) {
      const ticket = await response.json();
      activeLabel = `${ticket.ticket_number} · ${ticket.status}`;
    }
    const card = document.createElement("div");
    card.className = "ticket";
    card.innerHTML = `
      <div>
        <strong>${counter.name}</strong>
        <div><span>${counter.counter_id}</span></div>
      </div>
      <span>${activeLabel}</span>
    `;
    card.addEventListener("click", () => {
      state.counters = availableCounters;
      updateCounterSelect();
      counterSelect.value = counter.counter_id;
      setPresenceFromSelection();
      loadActiveTicket().catch(() => setStatus("Failed to load active ticket"));
    });
    counterList.appendChild(card);
  }
}

function logout() {
  state.sessionId = null;
  state.branches = [];
  state.services = [];
  state.counters = [];
  state.role = "";
  state.tenantId = "";
  state.branchId = "";
  state.serviceId = "";
  branchSelect.innerHTML = "";
  counterSelect.innerHTML = "";
  serviceSelect.innerHTML = "";
  transferSelect.innerHTML = "";
  ticketList.innerHTML = "<p class=\"hint\">Pick a service to see tickets.</p>";
  renderActive(null);
  setStatus("Logged out");
  setAlert("");
  supervisorToggle.value = "off";
  state.supervisor = false;
  supervisorPanel.hidden = true;
  sendUnsubscribe();
  if (socket) {
    socket.close();
    socket = null;
  }
}

loginBtn.addEventListener("click", () => {
  login().catch(() => {
    loginHint.textContent = "Login error.";
    setStatus("Login error");
    setAlert("Login error. Check network connectivity.");
  });
});

logoutBtn.addEventListener("click", () => {
  logout();
});

branchSelect.addEventListener("change", () => {
  sendUnsubscribe();
  loadServices().catch(() => setStatus("Failed to load services"));
  loadCounters().catch(() => setStatus("Failed to load counters"));
  loadActiveTicket().catch(() => setStatus("Failed to load active ticket"));
  loadSupervisorPanel().catch(() => setStatus("Failed to load supervisor panel"));
});

refreshBtn.addEventListener("click", () => {
  sendUnsubscribe();
  refreshQueue().catch(() => setStatus("Failed to load queue"));
  loadCounters().catch(() => setStatus("Failed to load counters"));
  loadActiveTicket().catch(() => setStatus("Failed to load active ticket"));
  loadSupervisorPanel().catch(() => setStatus("Failed to load supervisor panel"));
});

counterSelect.addEventListener("change", () => {
  setPresenceFromSelection();
  loadActiveTicket().catch(() => setStatus("Failed to load active ticket"));
});

addCounterBtn.addEventListener("click", () => {
  addCounter();
});

removeCounterBtn.addEventListener("click", () => {
  removeCounter();
});

savePresenceBtn.addEventListener("click", () => {
  savePresence().catch(() => setStatus("Failed to save presence"));
});

supervisorToggle.addEventListener("change", () => {
  state.supervisor = supervisorToggle.value === "on";
  loadSupervisorPanel().catch(() => setStatus("Failed to load supervisor panel"));
});

recallBtn.addEventListener("click", () => {
  performAction("recall").catch(() => setStatus("Recall failed"));
});

startBtn.addEventListener("click", () => {
  performAction("start").catch(() => setStatus("Start failed"));
});

completeBtn.addEventListener("click", () => {
  performAction("complete").catch(() => setStatus("Complete failed"));
});

holdBtn.addEventListener("click", () => {
  performAction("hold").catch(() => setStatus("Hold failed"));
});

unholdBtn.addEventListener("click", () => {
  performAction("unhold").catch(() => setStatus("Unhold failed"));
});

noShowBtn.addEventListener("click", () => {
  performAction("no-show").catch(() => setStatus("No-show failed"));
});

cancelBtn.addEventListener("click", () => {
  performAction("cancel").catch(() => setStatus("Cancel failed"));
});

transferBtn.addEventListener("click", () => {
  transferTicket().catch(() => setStatus("Transfer failed"));
});

callNextBtn.addEventListener("click", () => {
  callNext().catch(() => setStatus("Call next failed"));
});

setInterval(() => {
  if (state.serviceId) {
    refreshQueue().catch(() => setStatus("Failed to load queue"));
  }
  if (state.branchId && counterSelect.value) {
    loadActiveTicket().catch(() => setStatus("Failed to load active ticket"));
  }
  if (state.supervisor) {
    loadSupervisorPanel().catch(() => setStatus("Failed to load supervisor panel"));
  }
}, 10000);
