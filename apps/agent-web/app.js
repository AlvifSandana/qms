const state = {
  sessionId: null,
  branches: [],
  services: [],
  authBase: "http://localhost:8081",
  queueBase: "http://localhost:8080",
  tenantId: "",
  branchId: "",
  serviceId: "",
};

const authBaseInput = document.getElementById("authBase");
const queueBaseInput = document.getElementById("queueBase");
const tenantIdInput = document.getElementById("tenantId");
const emailInput = document.getElementById("email");
const passwordInput = document.getElementById("password");
const loginBtn = document.getElementById("loginBtn");
const loginHint = document.getElementById("loginHint");
const branchSelect = document.getElementById("branchSelect");
const counterInput = document.getElementById("counterId");
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
let socket = null;
let reconnectDelay = 1000;
let reconnectTimer = null;

function setStatus(text) {
  status.textContent = text;
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
    return;
  }

  const data = await response.json();
  state.sessionId = data.session_id;
  state.branches = data.branches || [];
  state.services = data.services || [];

  updateSelect(branchSelect, state.branches, "Select branch");
  setStatus("Logged in");
  loginHint.textContent = "Login success. Pick branch & service.";
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
  const counterId = counterInput.value.trim();
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
    return;
  }
  const ticket = await response.json();
  renderActive(ticket);
}

async function performAction(action) {
  const branchId = branchSelect.value;
  const counterId = counterInput.value.trim();
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
    return;
  }
  setStatus(`Action ok: ${action}`);
  await loadActiveTicket();
  await refreshQueue();
}

async function transferTicket() {
  const branchId = branchSelect.value;
  const counterId = counterInput.value.trim();
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
    return;
  }
  setStatus("Transfer ok");
  await loadActiveTicket();
  await refreshQueue();
}

async function callNext() {
  const branchId = branchSelect.value;
  const counterId = counterInput.value.trim();
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
    return;
  }

  if (!response.ok) {
    setStatus("Call next failed");
    return;
  }

  const ticket = await response.json();
  renderActive(ticket);
  setStatus(`Called ${ticket.ticket_number}`);
  await refreshQueue();
}

loginBtn.addEventListener("click", () => {
  login().catch(() => {
    loginHint.textContent = "Login error.";
    setStatus("Login error");
  });
});

branchSelect.addEventListener("change", () => {
  sendUnsubscribe();
  loadServices().catch(() => setStatus("Failed to load services"));
  loadActiveTicket().catch(() => setStatus("Failed to load active ticket"));
});

refreshBtn.addEventListener("click", () => {
  sendUnsubscribe();
  refreshQueue().catch(() => setStatus("Failed to load queue"));
  loadActiveTicket().catch(() => setStatus("Failed to load active ticket"));
});

counterInput.addEventListener("change", () => {
  loadActiveTicket().catch(() => setStatus("Failed to load active ticket"));
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
  if (state.branchId && counterInput.value.trim()) {
    loadActiveTicket().catch(() => setStatus("Failed to load active ticket"));
  }
}, 10000);
