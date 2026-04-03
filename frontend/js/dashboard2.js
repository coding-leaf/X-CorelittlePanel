async function fetchData() {
  try {
    const res = await panelFetch('/api/stats');
    const data = await res.json();
    updateDashboard(data);
  } catch(e) {
    document.getElementById('statusDot').className = 'status-dot offline';
    console.error('Fetch error:', e);
  }
}

async function fetchSysInfo() {
  try {
    const res = await panelFetch('/api/sysinfo');
    const d = await res.json();
    updateSysInfo(d);
  } catch(e) { console.error('SysInfo error:', e); }
}

async function fetchSpeed() {
  try {
    const res = await panelFetch('/api/speed');
    const d = await res.json();
    updateSpeed(d);
  } catch(e) {}
}

async function resetUser(email) {
  if (!confirm('确认清零用户 ' + email + ' 的流量统计？')) return;
  try {
    await panelFetch('/api/reset', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({email: email})
    });
    fetchData();
  } catch(e) { alert('清零失败: ' + e.message); }
}

// ===== Network Ping =====
async function runPing() {
  const btn = document.getElementById('pingBtn');
  const grid = document.getElementById('pingGrid');
  btn.disabled = true;
  btn.textContent = '检测中...';
  grid.innerHTML = '<div style="grid-column:1/-1" class="loading">正在检测</div>';

  try {
    const res = await panelFetch('/api/ping', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: '{}'
    });
    const data = await res.json();
    if (data.error) {
      grid.innerHTML = '<div style="grid-column:1/-1" class="error-msg">' + escHtml(data.error) + '</div>';
      return;
    }
    grid.innerHTML = data.map(r => {
      const ok = r.status === 'ok';
      return '<div class="ping-item">' +
        '<span class="ping-name">' + escHtml(r.name) + '</span>' +
        '<span class="ping-latency ' + (ok ? 'ping-ok' : 'ping-fail') + '">' +
          (ok ? r.latency_ms + ' ms' : '超时') +
        '</span></div>';
    }).join('');
  } catch(e) {
    grid.innerHTML = '<div style="grid-column:1/-1" class="error-msg">检测失败</div>';
  } finally {
    btn.disabled = false;
    btn.textContent = '开始检测';
  }
}

// ===== Init =====
// Fetch historical traffic data
async function fetchTrafficHistory() {
  try {
    const res = await panelFetch('/api/traffic-history');
    const data = await res.json();
    if (Array.isArray(data)) {
      histTrafficMap = {};
      data.forEach(item => { histTrafficMap[item.email] = item; });
    }
  } catch(e) { console.error('TrafficHistory error:', e); }
}

fetchTrafficHistory();
setInterval(fetchTrafficHistory, 60000);
fetchData();
fetchSysInfo();
startPolling();
connectWS();

window.addEventListener('resize', function() {
  const canvas = document.getElementById('trafficChart');
  if (canvas && canvas._lastHistory) drawChart(canvas._lastHistory);
});

