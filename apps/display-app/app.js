const state = {
  queueBase: "http://localhost:8080",
  realtimeBase: "http://localhost:8085",
  sessionId: "",
  tenantId: "",
  branchId: "",
  areaId: "",
  serviceId: "",
  serviceIds: [],
  lastAfter: "",
  audioEnabled: true,
  language: "id",
  calls: [],
  poller: null,
};

const queueBaseInput = document.getElementById("queueBase");
const realtimeBaseInput = document.getElementById("realtimeBase");
const sessionIdInput = document.getElementById("sessionId");
const tenantInput = document.getElementById("tenantId");
const branchInput = document.getElementById("branchId");
const deviceInput = document.getElementById("deviceId");
const serviceInput = document.getElementById("serviceId");
const areaInput = document.getElementById("areaId");
const langSelect = document.getElementById("langSelect");
const audioToggle = document.getElementById("audioToggle");
const quietStartInput = document.getElementById("quietStart");
const quietEndInput = document.getElementById("quietEnd");
const audioGapInput = document.getElementById("audioGap");
const connectBtn = document.getElementById("connectBtn");
const callList = document.getElementById("callList");
const status = document.getElementById("status");
const playlistEl = document.getElementById("playlist");
const brandLogo = document.getElementById("brandLogo");
const nowNumber = document.getElementById("nowNumber");
const nowCounter = document.getElementById("nowCounter");
const nowTime = document.getElementById("nowTime");
const fullscreenBtn = document.getElementById("fullscreenBtn");
const connState = document.getElementById("connState");
const callCount = document.getElementById("callCount");
const alertBox = document.getElementById("alert");

const maxCalls = 5;
let configVersion = 0;
let lastAudioAt = 0;
const audioQueue = [];
let sockets = [];
let pollInterval = null;
let reconnectDelay = 1000;
let reconnectTimer = null;
const playlistItems = [
  "Welcome to QMS",
  "Please prepare your ID",
  "Thank you for waiting",
];
let playlistIndex = 0;
let playlistSchedule = [];

function setStatus(text) {
  status.textContent = text;
}

function setAlert(message) {
  if (!message) {
    alertBox.hidden = true;
    return;
  }
  alertBox.textContent = message;
  alertBox.hidden = false;
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
        <div><span>Counter: ${call.counter_id || "-"} · ${call.service_id || "-"}</span></div>
      </div>
      <span>${new Date(call.called_at || call.created_at).toLocaleTimeString()}</span>
    `;
    callList.appendChild(row);
  });
  callCount.value = String(state.calls.length);
}

function renderNow(call) {
  if (!call) {
    nowNumber.textContent = "-";
    nowCounter.textContent = "Counter -";
    nowTime.textContent = "--:--";
    return;
  }
  nowNumber.textContent = call.ticket_number;
  nowCounter.textContent = `Counter ${call.counter_id || "-"}`;
  nowTime.textContent = new Date(call.called_at || call.created_at).toLocaleTimeString();
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
  renderNow(state.calls[0]);
  renderCalls();
  sayCall(call);
}

function matchFilter(payload) {
  if (state.branchId && payload.branch_id && payload.branch_id !== state.branchId) {
    return false;
  }
  if (state.areaId && payload.area_id !== state.areaId) {
    return false;
  }
  if (state.serviceIds.length > 0 && payload.service_id && !state.serviceIds.includes(payload.service_id)) {
    return false;
  }
  return true;
}

function authHeaders(extra = {}) {
  const headers = { ...extra };
  if (state.sessionId) {
    headers.Authorization = `Bearer ${state.sessionId}`;
  }
  return headers;
}

async function loadSnapshot() {
  if (!state.tenantId || !state.branchId || state.serviceIds.length === 0) {
    return;
  }
  for (const serviceId of state.serviceIds) {
    const response = await fetch(`${state.queueBase}/api/tickets/snapshot?tenant_id=${state.tenantId}&branch_id=${state.branchId}&service_id=${serviceId}`, {
      headers: authHeaders(),
    });
    if (!response.ok) {
      continue;
    }
    const tickets = await response.json();
    const called = tickets.filter((t) => {
      if (t.status !== "called" && t.status !== "serving") {
        return false;
      }
      if (state.areaId && t.area_id !== state.areaId) {
        return false;
      }
      return true;
    });
    called.forEach((ticket) => addCall({
      ticket_id: ticket.ticket_id,
      ticket_number: ticket.ticket_number,
      counter_id: ticket.counter_id,
      called_at: ticket.called_at || ticket.created_at,
      service_id: ticket.service_id,
      area_id: ticket.area_id,
    }));
  }
}

async function pollEvents() {
  if (!state.tenantId) {
    return;
  }
  const afterParam = state.lastAfter ? `&after=${encodeURIComponent(state.lastAfter)}` : "";
  const response = await fetch(`${state.queueBase}/api/events?tenant_id=${state.tenantId}${afterParam}&limit=50`, {
    headers: authHeaders(),
  });
  if (!response.ok) {
    setStatus("Disconnected");
    sendDeviceStatus("offline");
    return;
  }
  const events = await response.json();
  events.forEach(handleEvent);
  setStatus("Live");
  connState.value = "Live";
  setAlert("");
  sendDeviceStatus("online");
}

function connect() {
  state.queueBase = queueBaseInput.value.trim();
  state.realtimeBase = realtimeBaseInput.value.trim();
  state.sessionId = sessionIdInput.value.trim();
  state.tenantId = tenantInput.value.trim();
  state.branchId = branchInput.value.trim();
  state.areaId = areaInput.value.trim();
  state.serviceIds = parseServiceIds(serviceInput.value);
  state.serviceId = state.serviceIds[0] || "";
  state.language = langSelect.value;
  state.audioEnabled = audioToggle.value === "on";

  sendUnsubscribe();

  if (pollInterval) {
    clearInterval(pollInterval);
    pollInterval = null;
  }

  state.calls = [];
  state.lastAfter = "";
  renderCalls();
  renderNow(null);
  loadSnapshot().catch(() => setStatus("Snapshot failed"));

  connectSockJS();
}

function connectSockJS() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  sockets.forEach((item) => item.close());
  sockets = [];
  if (!state.sessionId) {
    setAlert("Session required for realtime. Using polling.");
    if (pollInterval) {
      clearInterval(pollInterval);
    }
    pollInterval = setInterval(() => {
      pollEvents().catch(() => setStatus("Disconnected"));
    }, 5000);
    return;
  }
  const sessionParam = encodeURIComponent(state.sessionId);
  const endpoint = `${state.realtimeBase}/realtime?session_id=${sessionParam}`;
  if (state.serviceIds.length === 0) {
    return;
  }
  state.serviceIds.forEach((serviceId) => {
    const socket = new SockJS(endpoint);
    sockets.push(socket);
    socket.onopen = () => {
      setStatus("Live");
      connState.value = "Live";
      setAlert("");
      sendDeviceStatus("online");
      reconnectDelay = 1000;
      const msg = {
        action: "subscribe",
        tenant_id: state.tenantId,
        branch_id: state.branchId,
        service_id: serviceId,
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
      connState.value = "Disconnected";
      setAlert("Connection lost. Using fallback polling.");
      sendDeviceStatus("offline");
      startPollingFallback();
      scheduleReconnect();
    };
  });
}

function scheduleReconnect() {
  if (reconnectTimer) {
    return;
  }
  const delay = reconnectDelay;
  reconnectDelay = Math.min(reconnectDelay * 2, 30000);
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    connectSockJS();
  }, delay);
}

function sendUnsubscribe() {
  sockets.forEach((socket) => {
    if (socket && socket.readyState === SockJS.OPEN) {
      socket.send(JSON.stringify({ action: "unsubscribe" }));
    }
  });
}

function startPollingFallback() {
  if (pollInterval) {
    return;
  }
  setAlert("Connection lost. Using fallback polling.");
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
    service_id: payload.service_id,
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
    state.serviceIds = parseServiceIds(payload.service_id);
    state.serviceId = state.serviceIds[0] || "";
  }
  if (payload.area_id) {
    areaInput.value = payload.area_id;
    state.areaId = String(payload.area_id).trim();
  }
  if (Array.isArray(payload.service_ids) && payload.service_ids.length > 0) {
    serviceInput.value = payload.service_ids.join(", ");
    state.serviceIds = payload.service_ids.map((id) => String(id).trim()).filter(Boolean);
    state.serviceId = state.serviceIds[0] || "";
  }
  if (Array.isArray(payload.playlist) && payload.playlist.length > 0) {
    playlistItems.length = 0;
    payload.playlist.forEach((item) => playlistItems.push(String(item)));
    playlistIndex = 0;
    updatePlaylist();
  }
  if (Array.isArray(payload.playlist_schedule)) {
    playlistSchedule = payload.playlist_schedule.map((item) => ({
      start: String(item.start || ""),
      end: String(item.end || ""),
      items: Array.isArray(item.items) ? item.items.map((val) => String(val)) : [],
    }));
    playlistIndex = 0;
    updatePlaylist();
  }
  if (payload.branding && typeof payload.branding === "object") {
    const branding = payload.branding;
    if (branding.logo_url) {
      brandLogo.src = String(branding.logo_url);
      brandLogo.hidden = false;
    }
    if (branding.accent_color) {
      document.documentElement.style.setProperty("--accent", String(branding.accent_color));
    }
    if (branding.background_url) {
      document.documentElement.style.setProperty("--bg-image", `url(${String(branding.background_url)})`);
    }
  }
}

connectBtn.addEventListener("click", () => {
  connect();
});

fullscreenBtn.addEventListener("click", () => {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen().catch(() => {});
    fullscreenBtn.textContent = "Exit Fullscreen";
  } else {
    document.exitFullscreen().catch(() => {});
    fullscreenBtn.textContent = "Fullscreen";
  }
});

langSelect.addEventListener("change", () => {
  state.language = langSelect.value;
});

audioToggle.addEventListener("change", () => {
  state.audioEnabled = audioToggle.value === "on";
});

renderCalls();
updatePlaylist();
renderNow(null);
connState.value = "Connecting...";

setInterval(() => {
  if (state.queueBase) {
    fetchDeviceConfig().catch(() => {});
  }
  playAudioQueue();
}, 10000);

function updatePlaylist() {
  const items = activePlaylistItems();
  if (items.length === 0) {
    return;
  }
  playlistEl.textContent = items[playlistIndex % items.length];
  playlistIndex += 1;
}

setInterval(updatePlaylist, 8000);

function parseServiceIds(value) {
  return value
    .split(",")
    .map((id) => id.trim())
    .filter((id) => id.length > 0);
}

function activePlaylistItems() {
  if (playlistSchedule.length === 0) {
    return playlistItems;
  }
  const now = new Date();
  const current = playlistSchedule.find((slot) => isWithinWindow(slot.start, slot.end, now));
  if (current && current.items.length > 0) {
    return current.items;
  }
  return playlistItems;
}

function isWithinWindow(start, end, now) {
  if (!start || !end) {
    return false;
  }
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
