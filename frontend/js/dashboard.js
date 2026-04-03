// ===== Dashboard update (shared by fetch and WebSocket) =====
function updateDashboard(data) {
  if (data.error) {
    document.getElementById('statusDot').className = 'status-dot offline';
    document.getElementById('userBody').innerHTML =
      '<tr><td colspan="6" class="error-msg">' + data.error + '</td></tr>';
    document.getElementById('inboundList').innerHTML =
      '<div class="error-msg">' + data.error + '</div>';
    document.getElementById('outboundList').innerHTML =
      '<div class="error-msg">' + data.error + '</div>';
    document.getElementById('sysGrid').innerHTML =
      '<div class="error-msg" style="grid-column:1/-1">' + data.error + '</div>';
    return;
  }

  document.getElementById('statusDot').className = 'status-dot';
  document.getElementById('updatedTime').textContent = data.updated_at;

  let totalUp = 0, totalDown = 0;
  (data.users || []).forEach(u => { totalUp += u.uplink; totalDown += u.downlink; });

  document.getElementById('totalUp').textContent = formatBytes(totalUp);
  document.getElementById('totalDown').textContent = formatBytes(totalDown);
  document.getElementById('totalAll').textContent = formatBytes(totalUp + totalDown);
  document.getElementById('uptime').textContent =
    data.sys_stats ? formatUptime(data.sys_stats.uptime) : '--';

  // Users
  const tbody = document.getElementById('userBody');
  if (!data.users || data.users.length === 0) {
    tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:var(--text-muted);padding:30px">暂无用户数据</td></tr>';
  } else {
    const maxTotal = Math.max(...data.users.map(u => u.total), 1);
    const grandTotal = data.users.reduce((s, u) => s + u.total, 0) || 1;
    tbody.innerHTML = data.users.map((u, i) => {
      const pct = ((u.total / grandTotal) * 100).toFixed(1);
      const barPct = ((u.total / maxTotal) * 100).toFixed(0);
      const c = colors[i % colors.length];
      const initial = u.email.charAt(0).toUpperCase();
      const h = histTrafficMap[u.email];
      const histStr = h ? formatBytes(h.hist_total) : '--';
      return '<tr>' +
        '<td><div class="user-email">' +
          '<div class="user-avatar" style="background:' + c + '">' + initial + '</div>' +
          '<span class="user-name">' + escHtml(u.email) + '</span>' +
        '</div></td>' +
        '<td style="color:var(--accent-green)">' + formatBytes(u.uplink) + '</td>' +
        '<td style="color:var(--accent-blue)">' + formatBytes(u.downlink) + '</td>' +
        '<td style="font-weight:600">' + formatBytes(u.total) + '</td>' +
        '<td class="hist-traffic">' + histStr + '</td>' +
        '<td><div class="traffic-bar"><div class="traffic-bar-fill" style="width:' + barPct + '%;background:' + c + '"></div></div><span style="font-size:12px;color:var(--text-muted)">' + pct + '%</span></td>' +
        '<td>' +
          '<button class="detail-btn" style="margin-right:6px" onclick="openDailyModal(\'' + escHtml(u.email).replace(/'/g, "\\'") + '\')">📊 详情</button>' +
          '<button class="detail-btn" style="margin-right:6px;color:var(--accent-purple);border-color:rgba(167,139,250,0.3);background:rgba(167,139,250,0.1)" onclick="openCycleModal(\'' + escHtml(u.email).replace(/'/g, "\\'") + '\')">⚙️ 周期</button>' +
          '<button class="detail-btn" style="color:var(--accent-red);border-color:rgba(248,113,113,0.3);background:rgba(248,113,113,0.1)" onclick="resetUser(\'' + escHtml(u.email).replace(/'/g, "\\'") + '\')">🔄 清零</button>' +
        '</td>' +
      '</tr>';
    }).join('');
  }

  // Inbounds
  const ibList = document.getElementById('inboundList');
  if (!data.inbounds || data.inbounds.length === 0) {
    ibList.innerHTML = '<div style="text-align:center;color:var(--text-muted);padding:20px">暂无数据</div>';
  } else {
    ibList.innerHTML = data.inbounds.map((ib, i) => {
      const c = colors[(i + 2) % colors.length];
      return '<div class="tag-item">' +
        '<div class="tag-name"><span class="tag-badge" style="background:' + c + '22;color:' + c + '">' + escHtml(ib.tag) + '</span></div>' +
        '<div class="tag-traffic"><span class="up">↑ ' + formatBytesShort(ib.uplink) + '</span> &nbsp; <span class="down">↓ ' + formatBytesShort(ib.downlink) + '</span></div>' +
      '</div>';
    }).join('');
  }

  // Outbounds
  const obList = document.getElementById('outboundList');
  if (!data.outbounds || data.outbounds.length === 0) {
    obList.innerHTML = '<div style="text-align:center;color:var(--text-muted);padding:20px">暂无数据</div>';
  } else {
    obList.innerHTML = data.outbounds.map((ob, i) => {
      const c = colors[(i + 4) % colors.length];
      return '<div class="tag-item">' +
        '<div class="tag-name"><span class="tag-badge" style="background:' + c + '22;color:' + c + '">' + escHtml(ob.tag) + '</span></div>' +
        '<div class="tag-traffic"><span class="up">↑ ' + formatBytesShort(ob.uplink) + '</span> &nbsp; <span class="down">↓ ' + formatBytesShort(ob.downlink) + '</span></div>' +
      '</div>';
    }).join('');
  }

  // System
  const sysGrid = document.getElementById('sysGrid');
  if (data.sys_stats) {
    const s = data.sys_stats;
    sysGrid.innerHTML = [
      {label: '协程数', value: s.num_goroutine},
      {label: '当前内存', value: formatBytes(s.alloc)},
      {label: '系统内存', value: formatBytes(s.sys)},
      {label: '累计分配', value: formatBytes(s.total_alloc)},
      {label: '存活对象', value: s.live_objects.toLocaleString()},
      {label: '运行时间', value: formatUptime(s.uptime)},
    ].map(item =>
      '<div class="sys-item"><div class="sys-label">' + item.label + '</div><div class="sys-value">' + item.value + '</div></div>'
    ).join('');
  }

  // Chart
  if (data.history && data.history.length > 0) {
    drawChart(data.history);
  }
}

function updateSysInfo(d) {
  if (d.error) return;
  const grid = document.getElementById('hostGrid');
  let items = [
    {label: '主机名', value: d.hostname || '--'},
    {label: '系统', value: (d.os + '/' + d.arch) || '--'},
    {label: 'CPU 核心', value: d.cpus || '--'},
  ];
  if (d.memory) {
    items.push({label: '内存使用', value: d.memory.usage_rate || '--'});
    items.push({label: '内存 已用/总量', value: (d.memory.used + ' / ' + d.memory.total)});
    items.push({label: '可用内存', value: d.memory.free || '--'});
  }
  if (d.load) {
    const parts = d.load.split(' ');
    items.push({label: '负载 (1/5/15分)', value: parts.slice(0,3).join(' / ')});
  }
  if (d.uptime) items.push({label: '系统运行', value: d.uptime.replace('up ', '')});
  grid.innerHTML = items.map(i =>
    '<div class="sys-item"><div class="sys-label">'+i.label+'</div><div class="sys-value">'+i.value+'</div></div>'
  ).join('');

  let extra = '';
  if (d.disk && d.disk.length) {
    extra += '<div style="margin-bottom:12px"><div style="font-size:12px;color:var(--text-muted);margin-bottom:8px;text-transform:uppercase;letter-spacing:0.5px">磁盘</div>';
    extra += '<table style="width:100%;font-size:13px;border-collapse:collapse">';
    extra += '<tr style="color:var(--text-muted);font-size:11px"><td>挂载点</td><td>总量</td><td>已用</td><td>可用</td><td>使用率</td></tr>';
    d.disk.forEach(dk => {
      extra += '<tr><td>'+escHtml(dk.mount)+'</td><td>'+dk.total+'</td><td>'+dk.used+'</td><td>'+dk.avail+'</td><td>'+dk.percent+'</td></tr>';
    });
    extra += '</table></div>';
  }
  if (d.processes && d.processes.length) {
    extra += '<div><div style="font-size:12px;color:var(--text-muted);margin-bottom:8px;text-transform:uppercase;letter-spacing:0.5px">进程 (TOP 10)</div>';
    extra += '<table style="width:100%;font-size:13px;border-collapse:collapse">';
    extra += '<tr style="color:var(--text-muted);font-size:11px"><td>PID</td><td>名称</td><td>CPU%</td><td>MEM%</td></tr>';
    d.processes.forEach(p => {
      extra += '<tr><td style="color:var(--text-muted)">'+p.pid+'</td><td>'+escHtml(p.name)+'</td><td style="color:var(--accent-cyan)">'+p.cpu+'</td><td style="color:var(--accent-purple)">'+p.mem+'</td></tr>';
    });
    extra += '</table></div>';
  }
  document.getElementById('hostExtra').innerHTML = extra;
}

function updateSpeed(d) {
  document.getElementById('speedUp').textContent = d.upload_str || '--';
  document.getElementById('speedDown').textContent = d.download_str || '--';
}


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

