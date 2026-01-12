const state = {
  queueBase: "http://localhost:8080",
  tenantId: "",
  branchId: "",
  serviceId: "",
  lastAfter: "",
  audioEnabled: true,
  language: "id",
  calls: [],
  poller: null,
};

const queueBaseInput = document.getElementById("queueBase");
const tenantInput = document.getElementById("tenantId");
const branchInput = document.getElementById("branchId");
const deviceInput = document.getElementById("deviceId");
const serviceInput = document.getElementById("serviceId");
const langSelect = document.getElementById("langSelect");
const audioToggle = document.getElementById("audioToggle");
const quietStartInput = document.getElementById("quietStart");
const quietEndInput = document.getElementById("quietEnd");
const audioGapInput = document.getElementById("audioGap");
const connectBtn = document.getElementById("connectBtn");
const callList = document.getElementById("callList");
const status = document.getElementById("status");
const playlistEl = document.getElementById("playlist");

const maxCalls = 5;
let configVersion = 0;
let lastAudioAt = 0;
const audioQueue = [];
let socket = null;
let pollInterval = null;
const playlistItems = [
  "Welcome to QMS",
  "Please prepare your ID",
  "Thank you for waiting",
];
let playlistIndex = 0;

function setStatus(text) {
  status.textContent = text;
}

function renderCalls() {
  callList.innerHTML = "";
  if (state.calls.length === 0) {
    callList.innerHTML = "<p class=\"hint\">No calls yet.</p>";
    return;
  }
  state.calls.forEach((call) => {
    const row = document.createElement("div");
    row.className = "call";
    row.innerHTML = `
      <div>
        <strong>${call.ticket_number}</strong>
        <div><span>Counter: ${call.counter_id || "-"}</span></div>
      </div>
      <span>${new Date(call.called_at || call.created_at).toLocaleTimeString()}</span>
    `;
    callList.appendChild(row);
  });
}

function sayCall(call) {
  if (!state.audioEnabled || !window.speechSynthesis) {
    return;
  }
  if (isQuietHours()) {
    return;
  }
  audioQueue.push(call);
  playAudioQueue();
}

function playAudioQueue() {
  const gapSeconds = parseInt(audioGapInput.value, 10) || 5;
  if (Date.now() - lastAudioAt < gapSeconds * 1000) {
    return;
  }
  if (audioQueue.length === 0) {
    return;
  }
  const call = audioQueue.shift();
  const number = call.ticket_number;
  const counter = call.counter_id || "";
  let text = "";
  if (state.language === "id") {
    text = `Nomor ${number} menuju loket ${counter}`;
  } else {
    text = `Ticket ${number} please go to counter ${counter}`;
  }
  const utterance = new SpeechSynthesisUtterance(text);
  lastAudioAt = Date.now();
  speechSynthesis.cancel();
  speechSynthesis.speak(utterance);
}

function isQuietHours() {
  const start = quietStartInput.value;
  const end = quietEndInput.value;
  if (!start || !end) {
    return false;
  }
  const now = new Date();
  const [sh, sm] = start.split(":").map(Number);
  const [eh, em] = end.split(":").map(Number);
  if (Number.isNaN(sh) || Number.isNaN(eh)) {
    return false;
  }
  const startTime = new Date(now);
  startTime.setHours(sh, sm || 0, 0, 0);
  const endTime = new Date(now);
  endTime.setHours(eh, em || 0, 0, 0);
  if (endTime <= startTime) {
    return now >= startTime || now <= endTime;
  }
  return now >= startTime && now <= endTime;
}
function addCall(call) {
  state.calls = [call, ...state.calls.filter((item) => item.ticket_id !== call.ticket_id)].slice(0, maxCalls);
  renderCalls();
  sayCall(call);
}

function matchFilter(payload) {
  if (state.branchId && payload.branch_id && payload.branch_id !== state.branchId) {
    return false;
  }
  if (state.serviceId && payload.service_id && payload.service_id !== state.serviceId) {
    return false;
  }
  return true;
}

async function loadSnapshot() {
  if (!state.tenantId || !state.branchId || !state.serviceId) {
    return;
  }
  const response = await fetch(`${state.queueBase}/api/tickets/snapshot?tenant_id=${state.tenantId}&branch_id=${state.branchId}&service_id=${state.serviceId}`);
  if (!response.ok) {
    return;
  }
  const tickets = await response.json();
  const called = tickets.filter((t) => t.status === "called" || t.status === "serving");
  called.forEach((ticket) => addCall({
    ticket_id: ticket.ticket_id,
    ticket_number: ticket.ticket_number,
    counter_id: ticket.counter_id,
    called_at: ticket.called_at || ticket.created_at,
  }));
}

async function pollEvents() {
  if (!state.tenantId) {
    return;
  }
  const afterParam = state.lastAfter ? `&after=${encodeURIComponent(state.lastAfter)}` : "";
  const response = await fetch(`${state.queueBase}/api/events?tenant_id=${state.tenantId}${afterParam}&limit=50`);
  if (!response.ok) {
    setStatus("Disconnected");
    sendDeviceStatus("offline");
    return;
  }
  const events = await response.json();
  events.forEach(handleEvent);
  setStatus("Live");
  sendDeviceStatus("online");
}

function connect() {
  state.queueBase = queueBaseInput.value.trim();
  state.tenantId = tenantInput.value.trim();
  state.branchId = branchInput.value.trim();
  state.serviceId = serviceInput.value.trim();
  state.language = langSelect.value;
  state.audioEnabled = audioToggle.value === "on";

  if (pollInterval) {
    clearInterval(pollInterval);
    pollInterval = null;
  }

  state.calls = [];
  state.lastAfter = "";
  renderCalls();
  loadSnapshot().catch(() => setStatus("Snapshot failed"));

  connectSockJS();
}

function connectSockJS() {
  if (socket) {
    socket.close();
  }
  const endpoint = `${state.queueBase}/realtime`;
  socket = new SockJS(endpoint);
  socket.onopen = () => {
    setStatus("Live");
    sendDeviceStatus("online");
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
      handleEvent(parsed);
    } catch (err) {
      return;
    }
  };
  socket.onclose = () => {
    setStatus("Disconnected");
    sendDeviceStatus("offline");
    startPollingFallback();
  };
}

function startPollingFallback() {
  if (pollInterval) {
    return;
  }
  pollInterval = setInterval(() => {
    pollEvents().catch(() => setStatus("Disconnected"));
  }, 1000);
}

function handleEvent(event) {
  if (!event || (!event.type && !event.payload)) {
    return;
  }
  const payload = event.payload || {};
  if (!matchFilter(payload)) {
    return;
  }
  if (event.type !== "ticket.called" && event.type !== "ticket.recalled") {
    return;
  }
  addCall({
    ticket_id: payload.ticket_id,
    ticket_number: payload.ticket_number,
    counter_id: payload.counter_id,
    called_at: payload.called_at || event.created_at,
    created_at: event.created_at,
  });
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
  applyConfig(data.payload || {});
}

function applyConfig(payload) {
  if (payload.language) {
    langSelect.value = payload.language;
    state.language = payload.language;
  }
  if (payload.audio_enabled === true) {
    audioToggle.value = "on";
    state.audioEnabled = true;
  }
  if (payload.audio_enabled === false) {
    audioToggle.value = "off";
    state.audioEnabled = false;
  }
  if (payload.service_id) {
    serviceInput.value = payload.service_id;
    state.serviceId = payload.service_id;
  }
  if (Array.isArray(payload.playlist) && payload.playlist.length > 0) {
    playlistItems.length = 0;
    payload.playlist.forEach((item) => playlistItems.push(String(item)));
    playlistIndex = 0;
    updatePlaylist();
  }
}

connectBtn.addEventListener("click", () => {
  connect();
});

langSelect.addEventListener("change", () => {
  state.language = langSelect.value;
});

audioToggle.addEventListener("change", () => {
  state.audioEnabled = audioToggle.value === "on";
});

renderCalls();
updatePlaylist();

setInterval(() => {
  if (state.queueBase) {
    fetchDeviceConfig().catch(() => {});
  }
  playAudioQueue();
}, 10000);

function updatePlaylist() {
  if (playlistItems.length === 0) {
    return;
  }
  playlistEl.textContent = playlistItems[playlistIndex % playlistItems.length];
  playlistIndex += 1;
}

setInterval(updatePlaylist, 8000);
