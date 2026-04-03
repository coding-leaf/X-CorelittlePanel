// ===== WebSocket =====
let ws = null, wsRetryTimer = null, wsConnected = false;
let pollTimer = null, sysInfoTimer = null;

function connectWS() {
  if (ws && ws.readyState <= 1) return;
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  ws = new WebSocket(proto + '//' + location.host + '/ws');

  ws.onopen = function() {
    wsConnected = true;
    document.getElementById('wsBadge').className = 'ws-badge ws-connected';
    document.getElementById('wsBadge').textContent = 'WS';
    // Stop polling when WS is connected
    if (pollTimer) { clearInterval(pollTimer); pollTimer = null; }
    if (sysInfoTimer) { clearInterval(sysInfoTimer); sysInfoTimer = null; }
  };

  ws.onmessage = function(ev) {
    try {
      const msg = JSON.parse(ev.data);
      if (msg.type === 'stats') updateDashboard(msg.data);
      else if (msg.type === 'sysinfo') updateSysInfo(msg.data);
      else if (msg.type === 'speed') updateSpeed(msg.data);
    } catch(e) { console.error('WS parse error:', e); }
  };

  ws.onclose = function() {
    wsConnected = false;
    document.getElementById('wsBadge').className = 'ws-badge ws-disconnected';
    document.getElementById('wsBadge').textContent = 'WS断开';
    // Fallback to polling
    startPolling();
    // Retry WS after 5s
    if (wsRetryTimer) clearTimeout(wsRetryTimer);
    wsRetryTimer = setTimeout(connectWS, 5000);
  };

  ws.onerror = function() { ws.close(); };
}

function startPolling() {
  if (!pollTimer) pollTimer = setInterval(function() { fetchData(); fetchSpeed(); }, 10000);
  if (!sysInfoTimer) sysInfoTimer = setInterval(fetchSysInfo, 30000);
}

