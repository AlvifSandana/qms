const queueBaseInput = document.getElementById("queueBase");
const realtimeBaseInput = document.getElementById("realtimeBase");
const tenantIdInput = document.getElementById("tenantId");
const branchIdInput = document.getElementById("branchId");
const serviceSelect = document.getElementById("serviceSelect");
const loadServicesBtn = document.getElementById("loadServices");
const joinQueueBtn = document.getElementById("joinQueue");
const phoneInput = document.getElementById("phoneInput");
const ticketCard = document.getElementById("ticketCard");
const statusEl = document.getElementById("status");
const setupHint = document.getElementById("setupHint");
const trackTicketId = document.getElementById("trackTicketId");
const trackServiceId = document.getElementById("trackServiceId");
const trackBtn = document.getElementById("trackBtn");
const timeline = document.getElementById("timeline");

const state = {
  queueBase: "http://localhost:8080",
  realtimeBase: "http://localhost:8085",
  tenantId: "",
  branchId: "",
  serviceId: "",
  ticketId: "",
  ticketNumber: "",
  lastAfter: "",
  events: [],
  seenEvents: new Set(),
  poller: null,
  socket: null,
  positionTimer: null,
};

function setStatus(text) {
  statusEl.textContent = text;
}

function setHint(text) {
  setupHint.textContent = text || "";
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

function uuidv4() {
  if (crypto?.randomUUID) {
    return crypto.randomUUID();
  }
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  return [...bytes].map((b, i) => (i === 4 || i === 6 || i === 8 || i === 10 ? "-" : "") + b.toString(16).padStart(2, "0")).join("");
}

async function loadServices() {
  state.queueBase = queueBaseInput.value.trim();
  state.realtimeBase = realtimeBaseInput.value.trim();
  state.tenantId = tenantIdInput.value.trim();
  state.branchId = branchIdInput.value.trim();
  if (!state.queueBase || !state.tenantId || !state.branchId) {
    setHint("Queue base, tenant, and branch are required.");
    return;
  }
  const response = await fetch(`${state.queueBase}/api/services?tenant_id=${state.tenantId}&branch_id=${state.branchId}`);
  if (!response.ok) {
    setHint("Failed to load services.");
    return;
  }
  const services = await response.json();
  updateServiceSelect(services);
  setHint("Services loaded.");
}

function renderTicket() {
  if (!state.ticketId) {
    ticketCard.innerHTML = `<p class="hint">No active ticket yet.</p>`;
    return;
  }
  ticketCard.innerHTML = `
    <h3>${state.ticketNumber}</h3>
    <p>Ticket ID: ${state.ticketId}</p>
    <p id="positionInfo">Position: checking...</p>
  `;
}

async function updatePosition() {
  if (!state.serviceId || !state.branchId || !state.tenantId) {
    return;
  }
  const response = await fetch(`${state.queueBase}/api/tickets/snapshot?tenant_id=${state.tenantId}&branch_id=${state.branchId}&service_id=${state.serviceId}`);
  if (!response.ok) {
    return;
  }
  const tickets = await response.json();
  const index = tickets.findIndex((ticket) => ticket.ticket_id === state.ticketId);
  const positionInfo = document.getElementById("positionInfo");
  if (!positionInfo) {
    return;
  }
  if (index === -1) {
    positionInfo.textContent = "Position: called or completed";
    return;
  }
  positionInfo.textContent = `Position: ${index + 1} in queue`;
}

function renderTimeline(events) {
  timeline.innerHTML = "";
  if (!events.length) {
    timeline.innerHTML = `<p class="hint">No updates yet.</p>`;
    return;
  }
  events.forEach((entry) => {
    const row = document.createElement("div");
    row.className = "event";
    row.innerHTML = `
      <div>
        <strong>${entry.label}</strong>
        <div><small>${entry.detail}</small></div>
      </div>
      <span>${entry.time}</span>
    `;
    timeline.appendChild(row);
  });
}

function normalizePayload(payload) {
  if (!payload) {
    return {};
  }
  if (typeof payload === "string") {
    try {
      return JSON.parse(payload);
    } catch (err) {
      return {};
    }
  }
  return payload;
}

async function pollEvents() {
  if (!state.tenantId || !state.ticketId) {
    return;
  }
  const afterParam = state.lastAfter ? `&after=${encodeURIComponent(state.lastAfter)}` : "";
  const response = await fetch(`${state.queueBase}/api/events?tenant_id=${state.tenantId}${afterParam}&limit=100`);
  if (!response.ok) {
    setStatus("Offline");
    return;
  }
  const events = await response.json();
  const updates = [];
  for (const event of events) {
    state.lastAfter = event.created_at || state.lastAfter;
    if (state.seenEvents.has(event.event_id)) {
      continue;
    }
    const payload = normalizePayload(event.payload);
    if (payload.ticket_id !== state.ticketId) {
      continue;
    }
    if (event.event_id) {
      state.seenEvents.add(event.event_id);
    }
    updates.push({
      label: event.type,
      detail: payload.counter_id ? `Counter ${payload.counter_id}` : payload.status || "",
      time: new Date(event.created_at || Date.now()).toLocaleTimeString(),
    });
  }
  if (updates.length) {
    state.events = [...updates, ...state.events].slice(0, 20);
    renderTimeline(state.events);
  }
  setStatus("Tracking");
}

function connectRealtime() {
  if (!state.realtimeBase || !state.tenantId) {
    return;
  }
  if (state.socket) {
    state.socket.close();
    state.socket = null;
  }
  const endpoint = `${state.realtimeBase}/realtime`;
  const socket = new SockJS(endpoint);
  state.socket = socket;
  socket.onopen = () => {
    const msg = {
      action: "subscribe",
      tenant_id: state.tenantId,
      branch_id: state.branchId,
      service_id: state.serviceId,
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
    state.socket = null;
  };
}

function handleRealtimeEvent(event) {
  if (!event || !event.type) {
    return;
  }
  const payload = normalizePayload(event.payload);
  if (payload.ticket_id !== state.ticketId) {
    return;
  }
  if (event.event_id && state.seenEvents.has(event.event_id)) {
    return;
  }
  if (event.event_id) {
    state.seenEvents.add(event.event_id);
  }
  state.events = [{
    label: event.type,
    detail: payload.counter_id ? `Counter ${payload.counter_id}` : payload.status || "",
    time: new Date(event.created_at || Date.now()).toLocaleTimeString(),
  }, ...state.events].slice(0, 20);
  renderTimeline(state.events);
  setStatus("Tracking");
}

function startPolling() {
  if (state.poller) {
    clearInterval(state.poller);
  }
  state.poller = setInterval(() => {
    pollEvents().catch(() => setStatus("Offline"));
  }, 5000);
  if (state.positionTimer) {
    clearInterval(state.positionTimer);
  }
  state.positionTimer = setInterval(() => {
    updatePosition().catch(() => {});
  }, 15000);
}

async function joinQueue() {
  state.queueBase = queueBaseInput.value.trim();
  state.realtimeBase = realtimeBaseInput.value.trim();
  state.tenantId = tenantIdInput.value.trim();
  state.branchId = branchIdInput.value.trim();
  state.serviceId = serviceSelect.value;
  if (!state.queueBase || !state.tenantId || !state.branchId || !state.serviceId) {
    setHint("Queue base, tenant, branch, and service are required.");
    return;
  }
  const payload = {
    request_id: uuidv4(),
    tenant_id: state.tenantId,
    branch_id: state.branchId,
    service_id: state.serviceId,
    channel: "web",
    priority_class: "regular",
    phone: phoneInput.value.trim(),
  };
  const response = await fetch(`${state.queueBase}/api/tickets`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) {
    setStatus("Request failed");
    return;
  }
  const ticket = await response.json();
  state.ticketId = ticket.ticket_id;
  state.ticketNumber = ticket.ticket_number;
  renderTicket();
  trackTicketId.value = ticket.ticket_id;
  setStatus(`Ticket ${ticket.ticket_number}`);
  state.lastAfter = "";
  state.seenEvents.clear();
  state.events = [];
  renderTimeline(state.events);
  connectRealtime();
  startPolling();
  updatePosition().catch(() => {});
}

function trackTicket() {
  state.queueBase = queueBaseInput.value.trim();
  state.realtimeBase = realtimeBaseInput.value.trim();
  state.tenantId = tenantIdInput.value.trim();
  state.branchId = branchIdInput.value.trim();
  state.serviceId = trackServiceId.value.trim();
  const id = trackTicketId.value.trim();
  if (!id || !state.queueBase || !state.tenantId) {
    setHint("Queue base, tenant ID, and ticket ID required.");
    return;
  }
  state.ticketId = id;
  state.lastAfter = "";
  state.seenEvents.clear();
  state.events = [];
  renderTimeline(state.events);
  connectRealtime();
  startPolling();
  updatePosition().catch(() => {});
}

loadServicesBtn.addEventListener("click", () => {
  loadServices().catch(() => setHint("Failed to load services."));
});

joinQueueBtn.addEventListener("click", () => {
  joinQueue().catch(() => setStatus("Request failed"));
});

trackBtn.addEventListener("click", () => {
  trackTicket();
});

setStatus("Ready");
