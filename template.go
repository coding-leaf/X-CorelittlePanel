package main

// ============================================================
// Frontend HTML Templates
// ============================================================
// Note: To deploy the Xray Panel as a single binary executable without
// external static assets, the HTML, CSS, and JS for the frontend interfaces
// (Dashboard, Admin Panel, Login etc.) are embedded here as raw string constants.
//
// While this is not conventional for large web applications, it ensures
// maximum portability and ease of installation for end users.

// indexHTML contains the main dashboard UI.
const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Xray Panel</title>
<style>
  @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');

  * { margin: 0; padding: 0; box-sizing: border-box; }

  :root {
    --bg-primary: #0a0e1a;
    --bg-card: rgba(17, 24, 45, 0.85);
    --bg-card-hover: rgba(25, 35, 60, 0.9);
    --border: rgba(99, 115, 168, 0.15);
    --text-primary: #e2e8f0;
    --text-secondary: #8892b0;
    --text-muted: #5a6480;
    --accent-blue: #60a5fa;
    --accent-purple: #a78bfa;
    --accent-green: #34d399;
    --accent-orange: #fb923c;
    --accent-red: #f87171;
    --accent-cyan: #22d3ee;
    --gradient-1: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    --gradient-2: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
    --gradient-3: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
    --gradient-4: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
    --shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  }

  body {
    font-family: 'Inter', -apple-system, sans-serif;
    background: var(--bg-primary);
    color: var(--text-primary);
    min-height: 100vh;
    overflow-x: hidden;
  }

  body::before {
    content: '';
    position: fixed;
    top: -50%; left: -50%;
    width: 200%; height: 200%;
    background:
      radial-gradient(ellipse at 20% 50%, rgba(99, 102, 241, 0.08) 0%, transparent 50%),
      radial-gradient(ellipse at 80% 20%, rgba(139, 92, 246, 0.06) 0%, transparent 50%),
      radial-gradient(ellipse at 50% 80%, rgba(59, 130, 246, 0.05) 0%, transparent 50%);
    z-index: 0;
    animation: bgShift 20s ease-in-out infinite alternate;
  }

  @keyframes bgShift {
    0% { transform: translate(0, 0) rotate(0deg); }
    100% { transform: translate(-2%, -2%) rotate(1deg); }
  }

  .container {
    position: relative;
    z-index: 1;
    max-width: 1280px;
    margin: 0 auto;
    padding: 24px 20px;
  }

  /* Header */
  .header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 28px;
    padding: 20px 28px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 16px;
    backdrop-filter: blur(20px);
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 14px;
  }

  .logo {
    width: 42px; height: 42px;
    background: var(--gradient-1);
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    font-weight: 700;
    box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
  }

  .header h1 {
    font-size: 22px;
    font-weight: 700;
    background: linear-gradient(135deg, #e2e8f0, #a78bfa);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .status-dot {
    width: 8px; height: 8px;
    border-radius: 50%;
    background: var(--accent-green);
    box-shadow: 0 0 8px var(--accent-green);
    animation: pulse 2s infinite;
  }

  .status-dot.offline {
    background: var(--accent-red);
    box-shadow: 0 0 8px var(--accent-red);
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
  }

  .updated-time {
    font-size: 13px;
    color: var(--text-muted);
  }

  .refresh-btn {
    padding: 8px 18px;
    background: rgba(96, 165, 250, 0.12);
    border: 1px solid rgba(96, 165, 250, 0.25);
    color: var(--accent-blue);
    border-radius: 10px;
    cursor: pointer;
    font-size: 13px;
    font-weight: 500;
    transition: all 0.25s;
  }

  .refresh-btn:hover {
    background: rgba(96, 165, 250, 0.2);
    transform: translateY(-1px);
  }

  /* Stats Cards Row */
  .stats-row {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    margin-bottom: 24px;
  }

  .stat-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 20px 22px;
    backdrop-filter: blur(20px);
    transition: all 0.3s;
    position: relative;
    overflow: hidden;
  }

  .stat-card::before {
    content: '';
    position: absolute;
    top: 0; left: 0; right: 0;
    height: 3px;
    border-radius: 14px 14px 0 0;
  }

  .stat-card:nth-child(1)::before { background: var(--gradient-3); }
  .stat-card:nth-child(2)::before { background: var(--gradient-1); }
  .stat-card:nth-child(3)::before { background: var(--gradient-4); }
  .stat-card:nth-child(4)::before { background: var(--gradient-2); }

  .stat-card:hover {
    transform: translateY(-3px);
    border-color: rgba(99, 115, 168, 0.3);
    box-shadow: var(--shadow);
  }

  .stat-card .label {
    font-size: 12px;
    font-weight: 500;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.8px;
    margin-bottom: 10px;
  }

  .stat-card .value {
    font-size: 28px;
    font-weight: 700;
    letter-spacing: -0.5px;
  }

  .stat-card:nth-child(1) .value { color: var(--accent-cyan); }
  .stat-card:nth-child(2) .value { color: var(--accent-purple); }
  .stat-card:nth-child(3) .value { color: var(--accent-green); }
  .stat-card:nth-child(4) .value { color: var(--accent-orange); }

  .stat-card .sub {
    font-size: 12px;
    color: var(--text-muted);
    margin-top: 6px;
  }

  /* Main Grid */
  .main-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
    margin-bottom: 24px;
  }

  .card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 14px;
    backdrop-filter: blur(20px);
    overflow: hidden;
    transition: border-color 0.3s;
  }

  .card:hover { border-color: rgba(99, 115, 168, 0.3); }

  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 18px 22px;
    border-bottom: 1px solid var(--border);
  }

  .card-header h2 {
    font-size: 15px;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .card-header .icon {
    width: 28px; height: 28px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 14px;
  }

  .card-body { padding: 16px 22px; }

  .card.full-width {
    grid-column: 1 / -1;
  }

  /* User Table */
  .user-table-wrap {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  .user-table {
    width: 100%;
    border-collapse: collapse;
    min-width: 640px;
  }

  .user-table th {
    text-align: left;
    padding: 10px 12px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.8px;
    border-bottom: 1px solid var(--border);
    white-space: nowrap;
  }

  .user-table td {
    padding: 14px 12px;
    font-size: 14px;
    border-bottom: 1px solid rgba(99, 115, 168, 0.08);
    vertical-align: middle;
    white-space: nowrap;
  }

  .user-table tr:last-child td { border-bottom: none; }

  .user-table tr:hover td {
    background: rgba(96, 165, 250, 0.04);
  }

  .hist-traffic {
    font-size: 13px;
    color: var(--accent-cyan);
    font-weight: 600;
  }

  .user-email {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .user-avatar {
    width: 32px; height: 32px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 13px;
    font-weight: 600;
    color: white;
  }

  .user-name {
    font-weight: 500;
  }

  .traffic-bar {
    height: 6px;
    background: rgba(255,255,255,0.06);
    border-radius: 3px;
    overflow: hidden;
    margin-top: 4px;
    min-width: 80px;
  }

  .traffic-bar-fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.6s ease;
  }



  /* Tag Items */
  .tag-list { display: flex; flex-direction: column; gap: 10px; }

  .tag-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 16px;
    background: rgba(255,255,255,0.02);
    border: 1px solid rgba(255,255,255,0.04);
    border-radius: 10px;
    transition: all 0.2s;
  }

  .tag-item:hover {
    background: rgba(255,255,255,0.04);
    border-color: rgba(255,255,255,0.08);
  }

  .tag-name {
    font-weight: 500;
    font-size: 14px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .tag-badge {
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
  }

  .tag-traffic {
    text-align: right;
    font-size: 13px;
    color: var(--text-secondary);
  }

  .tag-traffic .up { color: var(--accent-green); }
  .tag-traffic .down { color: var(--accent-blue); }

  /* System Stats */
  .sys-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
  }

  .sys-item {
    padding: 14px;
    background: rgba(255,255,255,0.02);
    border-radius: 10px;
    text-align: center;
  }

  .sys-item .sys-label {
    font-size: 11px;
    color: var(--text-muted);
    margin-bottom: 6px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .sys-item .sys-value {
    font-size: 18px;
    font-weight: 600;
    color: var(--text-primary);
  }

  /* Error state */
  .error-msg {
    text-align: center;
    padding: 40px 20px;
    color: var(--accent-red);
    font-size: 14px;
  }

  .loading {
    text-align: center;
    padding: 40px 20px;
    color: var(--text-muted);
  }

  .loading::after {
    content: '';
    display: inline-block;
    width: 16px; height: 16px;
    border: 2px solid var(--text-muted);
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin-left: 8px;
    vertical-align: middle;
  }

  @keyframes spin { to { transform: rotate(360deg); } }

  /* Chart */
  .chart-container {
    width: 100%;
    height: 200px;
    position: relative;
  }

  .chart-canvas {
    width: 100%;
    height: 100%;
  }

  /* Chart */
  .chart-container { width: 100%; height: 220px; position: relative; }
  .chart-canvas { width: 100%; height: 100%; }

  /* Speed indicator */
  .speed-value { font-size: 18px !important; }
  .speed-unit { font-size: 12px; color: var(--text-muted); }

  /* Network check */
  .ping-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; }
  .ping-item {
    display: flex; align-items: center; justify-content: space-between;
    padding: 12px 14px; background: rgba(255,255,255,0.02);
    border: 1px solid rgba(255,255,255,0.04); border-radius: 10px;
    transition: all 0.2s;
  }
  .ping-name { font-size: 13px; font-weight: 500; }
  .ping-latency { font-size: 13px; font-weight: 600; }
  .ping-ok { color: var(--accent-green); }
  .ping-fail { color: var(--accent-red); }
  .ping-loading { color: var(--text-muted); }
  .ping-btn {
    padding: 7px 16px; background: rgba(96,165,250,0.12);
    border: 1px solid rgba(96,165,250,0.25); color: var(--accent-blue);
    border-radius: 8px; cursor: pointer; font-size: 12px; font-weight: 500;
    font-family: inherit; transition: all 0.2s;
  }
  .ping-btn:hover { background: rgba(96,165,250,0.2); transform: translateY(-1px); }
  .ping-btn:disabled { opacity: 0.5; cursor: not-allowed; transform: none; }

  /* WS status */
  .ws-badge {
    font-size: 10px; padding: 2px 8px; border-radius: 4px;
    font-weight: 600; letter-spacing: 0.3px;
  }
  .ws-connected { background: rgba(52,211,153,0.15); color: var(--accent-green); }
  .ws-disconnected { background: rgba(248,113,113,0.15); color: var(--accent-red); }

  /* Responsive */
  @media (max-width: 768px) {
    .stats-row { grid-template-columns: repeat(2, 1fr); }
    .main-grid { grid-template-columns: 1fr; }
    .sys-grid { grid-template-columns: repeat(2, 1fr); }
    .stat-card .value { font-size: 22px; }
    .ping-grid { grid-template-columns: 1fr; }
    .chart-container { height: 180px; }
  }
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <div class="header-left">
      <div class="logo">X</div>
      <h1>Xray Panel</h1>
    </div>
    <div class="header-right">
      <div class="status-dot" id="statusDot"></div>
      <span class="ws-badge ws-disconnected" id="wsBadge">WS</span>
      <span class="updated-time" id="updatedTime">--</span>
      <button class="refresh-btn" onclick="fetchData()">刷新</button>
      <a href="/admin/panel" class="refresh-btn" style="text-decoration:none;background:rgba(167,139,250,0.12);border-color:rgba(167,139,250,0.25);color:#a78bfa">🔧 管理</a>
    </div>
  </div>

  <div class="stats-row" id="statsRow">
    <div class="stat-card">
      <div class="label">总上行</div>
      <div class="value" id="totalUp">--</div>
    </div>
    <div class="stat-card">
      <div class="label">总下行</div>
      <div class="value" id="totalDown">--</div>
    </div>
    <div class="stat-card">
      <div class="label">总流量</div>
      <div class="value" id="totalAll">--</div>
    </div>
    <div class="stat-card">
      <div class="label">运行时间</div>
      <div class="value" id="uptime">--</div>
    </div>
    <div class="stat-card">
      <div class="label">↑ 实时速率</div>
      <div class="value speed-value" id="speedUp">--</div>
    </div>
    <div class="stat-card">
      <div class="label">↓ 实时速率</div>
      <div class="value speed-value" id="speedDown">--</div>
    </div>
  </div>

  <!-- Traffic Chart -->
  <div class="card full-width" style="margin-bottom:20px">
    <div class="card-header">
      <h2>
        <span class="icon" style="background:rgba(96,165,250,0.15);color:var(--accent-blue)">📈</span>
        带宽趋势
      </h2>
    </div>
    <div class="card-body">
      <div class="chart-container">
        <canvas class="chart-canvas" id="trafficChart"></canvas>
      </div>
    </div>
  </div>

  <div class="main-grid">
    <!-- Users -->
    <div class="card full-width">
      <div class="card-header">
        <h2>
          <span class="icon" style="background:rgba(167,139,250,0.15);color:var(--accent-purple)">👤</span>
          用户流量
        </h2>
      </div>
      <div class="card-body">
        <div class="user-table-wrap">
        <table class="user-table" id="userTable">
          <thead>
            <tr>
              <th>用户</th>
              <th>上行</th>
              <th>下行</th>
              <th>当期流量</th>
              <th>历史总流量</th>
              <th>占比</th>
            </tr>
          </thead>
          <tbody id="userBody">
            <tr><td colspan="6" class="loading">加载中</td></tr>
          </tbody>
        </table>
        </div>
      </div>
    </div>

    <!-- Inbounds -->
    <div class="card">
      <div class="card-header">
        <h2>
          <span class="icon" style="background:rgba(52,211,153,0.15);color:var(--accent-green)">⬇</span>
          入站统计
        </h2>
      </div>
      <div class="card-body">
        <div class="tag-list" id="inboundList">
          <div class="loading">加载中</div>
        </div>
      </div>
    </div>

    <!-- Outbounds -->
    <div class="card">
      <div class="card-header">
        <h2>
          <span class="icon" style="background:rgba(96,165,250,0.15);color:var(--accent-blue)">⬆</span>
          出站统计
        </h2>
      </div>
      <div class="card-body">
        <div class="tag-list" id="outboundList">
          <div class="loading">加载中</div>
        </div>
      </div>
    </div>

    <!-- System -->
    <div class="card full-width">
      <div class="card-header">
        <h2>
          <span class="icon" style="background:rgba(251,146,60,0.15);color:var(--accent-orange)">⚙</span>
          系统状态
        </h2>
      </div>
      <div class="card-body">
        <div class="sys-grid" id="sysGrid">
          <div class="loading">加载中</div>
        </div>
    </div>

    <!-- Host System Info -->
    <div class="card full-width">
      <div class="card-header">
        <h2>
          <span class="icon" style="background:rgba(34,211,238,0.15);color:var(--accent-cyan)">🖥</span>
          主机信息
        </h2>
      </div>
      <div class="card-body">
        <div class="sys-grid" id="hostGrid">
          <div class="loading">加载中</div>
        </div>
        <div id="hostExtra" style="margin-top:16px"></div>
      </div>
    </div>
  </div>

  <!-- Network Check -->
  <div class="card full-width" style="margin-top:20px">
    <div class="card-header">
      <h2>
        <span class="icon" style="background:rgba(34,211,238,0.15);color:var(--accent-cyan)">🌐</span>
        网络检测
      </h2>
      <button class="ping-btn" id="pingBtn" onclick="runPing()">开始检测</button>
    </div>
    <div class="card-body">
      <div class="ping-grid" id="pingGrid">
        <div style="grid-column:1/-1;text-align:center;color:var(--text-muted);padding:20px">点击「开始检测」测试网络连通性</div>
      </div>
    </div>
  </div>
</div>

<script>
const colors = ['#667eea','#f093fb','#4facfe','#43e97b','#fb923c','#f87171','#22d3ee','#a78bfa'];
let histTrafficMap = {}; // email -> {hist_uplink, hist_downlink, hist_total}
function panelFetch(url, opts) {
  opts = opts || {};
  opts.headers = Object.assign({'X-Panel': '1'}, opts.headers || {});
  return fetch(url, opts);
}

function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i];
}

function formatUptime(seconds) {
  if (!seconds) return '--';
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return d + '天 ' + h + '时';
  if (h > 0) return h + '时 ' + m + '分';
  return m + '分';
}

function formatBytesShort(bytes) {
  if (bytes === 0) return '0B';
  const k = 1024;
  const sizes = ['B', 'K', 'M', 'G', 'T'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return (bytes / Math.pow(k, i)).toFixed(1) + sizes[i];
}

function escHtml(s) {
  const d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}

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
    tbody.innerHTML = '<tr><td colspan="6" style="text-align:center;color:var(--text-muted);padding:30px">暂无用户数据</td></tr>';
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

// ===== Chart Drawing =====
function drawChart(history) {
  const canvas = document.getElementById('trafficChart');
  if (!canvas) return;
  const ctx = canvas.getContext('2d');
  const dpr = window.devicePixelRatio || 1;
  const rect = canvas.parentElement.getBoundingClientRect();
  canvas.width = rect.width * dpr;
  canvas.height = rect.height * dpr;
  canvas.style.width = rect.width + 'px';
  canvas.style.height = rect.height + 'px';
  ctx.scale(dpr, dpr);

  const w = rect.width, h = rect.height;
  const pad = {top: 20, right: 20, bottom: 30, left: 60};
  const cw = w - pad.left - pad.right;
  const ch = h - pad.top - pad.bottom;

  ctx.clearRect(0, 0, w, h);

  if (history.length < 3) {
    ctx.fillStyle = '#5a6480';
    ctx.font = '13px Inter, sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText('数据收集中...', w/2, h/2);
    return;
  }

  // Compute deltas (rate) between consecutive points
  const ups = [], downs = [], times = [];
  for (let i = 1; i < history.length; i++) {
    const du = Math.max(0, history[i].uplink - history[i-1].uplink);
    const dd = Math.max(0, history[i].downlink - history[i-1].downlink);
    ups.push(du);
    downs.push(dd);
    times.push(history[i].time);
  }
  const allVals = [...ups, ...downs];
  const maxVal = Math.max(...allVals, 1);

  // Grid lines
  ctx.strokeStyle = 'rgba(99,115,168,0.1)';
  ctx.lineWidth = 1;
  const gridLines = 4;
  for (let i = 0; i <= gridLines; i++) {
    const y = pad.top + (ch / gridLines) * i;
    ctx.beginPath();
    ctx.moveTo(pad.left, y);
    ctx.lineTo(pad.left + cw, y);
    ctx.stroke();
    // Y label
    const val = maxVal - (maxVal / gridLines) * i;
    ctx.fillStyle = '#5a6480';
    ctx.font = '10px Inter, sans-serif';
    ctx.textAlign = 'right';
    ctx.fillText(formatBytesShort(val), pad.left - 8, y + 4);
  }

  // X labels
  const step = Math.max(1, Math.floor(times.length / 6));
  ctx.fillStyle = '#5a6480';
  ctx.font = '10px Inter, sans-serif';
  ctx.textAlign = 'center';
  for (let i = 0; i < times.length; i += step) {
    const x = pad.left + (i / (times.length - 1)) * cw;
    ctx.fillText(times[i], x, h - 8);
  }

  function drawLine(values, color, fillColor) {
    ctx.beginPath();
    for (let i = 0; i < values.length; i++) {
      const x = pad.left + (i / (values.length - 1)) * cw;
      const y = pad.top + ch - (values[i] / maxVal) * ch;
      if (i === 0) ctx.moveTo(x, y); else ctx.lineTo(x, y);
    }
    ctx.strokeStyle = color;
    ctx.lineWidth = 2;
    ctx.stroke();

    // Fill
    const lastX = pad.left + cw;
    ctx.lineTo(lastX, pad.top + ch);
    ctx.lineTo(pad.left, pad.top + ch);
    ctx.closePath();
    const grad = ctx.createLinearGradient(0, pad.top, 0, pad.top + ch);
    grad.addColorStop(0, fillColor);
    grad.addColorStop(1, 'rgba(0,0,0,0)');
    ctx.fillStyle = grad;
    ctx.fill();
  }

  drawLine(ups, '#34d399', 'rgba(52,211,153,0.15)');
  drawLine(downs, '#60a5fa', 'rgba(96,165,250,0.15)');

  // Legend
  const lx = pad.left + 10;
  ctx.font = '11px Inter, sans-serif';
  ctx.fillStyle = '#34d399';
  ctx.fillRect(lx, pad.top + 2, 12, 3);
  ctx.fillText('上行', lx + 16, pad.top + 8);
  ctx.fillStyle = '#60a5fa';
  ctx.fillRect(lx + 60, pad.top + 2, 12, 3);
  ctx.fillText('下行', lx + 76, pad.top + 8);
}

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
</script>
</body>
</html>`

const logsHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Xray Logs</title>
<style>
  @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');
  * { margin: 0; padding: 0; box-sizing: border-box; }
  :root {
    --bg-primary: #0a0e1a;
    --bg-card: rgba(17, 24, 45, 0.85);
    --border: rgba(99, 115, 168, 0.15);
    --text-primary: #e2e8f0;
    --text-secondary: #8892b0;
    --text-muted: #5a6480;
    --accent-blue: #60a5fa;
    --accent-purple: #a78bfa;
    --accent-green: #34d399;
    --accent-red: #f87171;
    --accent-cyan: #22d3ee;
    --accent-yellow: #fbbf24;
  }
  body {
    font-family: 'Inter', -apple-system, sans-serif;
    background: var(--bg-primary);
    color: var(--text-primary);
    min-height: 100vh;
  }
  body::before {
    content: '';
    position: fixed;
    top: -50%; left: -50%;
    width: 200%; height: 200%;
    background:
      radial-gradient(ellipse at 20% 50%, rgba(99,102,241,0.08) 0%, transparent 50%),
      radial-gradient(ellipse at 80% 20%, rgba(139,92,246,0.06) 0%, transparent 50%);
    z-index: 0;
  }
  .layout { position: relative; z-index: 1; display: flex; height: 100vh; }

  .sidebar {
    width: 260px; min-width: 260px;
    background: var(--bg-card);
    border-right: 1px solid var(--border);
    display: flex; flex-direction: column;
    backdrop-filter: blur(20px);
  }
  .sidebar-header { padding: 20px; border-bottom: 1px solid var(--border); }
  .sidebar-header h2 { font-size: 15px; font-weight: 600; display: flex; align-items: center; gap: 8px; }
  .sidebar-header .back-link {
    font-size: 12px; color: var(--text-muted); text-decoration: none;
    margin-top: 8px; display: inline-block; transition: color 0.2s;
  }
  .sidebar-header .back-link:hover { color: var(--accent-blue); }
  .user-list { flex: 1; overflow-y: auto; padding: 8px; }
  .user-item {
    display: flex; align-items: center; gap: 10px;
    padding: 10px 12px; border-radius: 10px; cursor: pointer;
    transition: all 0.2s; margin-bottom: 2px;
  }
  .user-item:hover { background: rgba(255,255,255,0.04); }
  .user-item.active { background: rgba(96,165,250,0.12); border: 1px solid rgba(96,165,250,0.25); }
  .user-item .avatar {
    width: 32px; height: 32px; border-radius: 8px;
    display: flex; align-items: center; justify-content: center;
    font-size: 13px; font-weight: 600; color: white; flex-shrink: 0;
  }
  .user-item .name { font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .all-users-item { border-bottom: 1px solid var(--border); margin-bottom: 4px; padding-bottom: 12px; }

  .main { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
  .toolbar {
    display: flex; align-items: center; gap: 12px;
    padding: 16px 24px; border-bottom: 1px solid var(--border);
    background: var(--bg-card); backdrop-filter: blur(20px);
  }
  .tabs { display: flex; gap: 4px; background: rgba(255,255,255,0.04); border-radius: 10px; padding: 3px; }
  .tab {
    padding: 7px 16px; border-radius: 8px; font-size: 13px; font-weight: 500;
    cursor: pointer; transition: all 0.2s; color: var(--text-muted);
    border: none; background: none; font-family: inherit;
  }
  .tab:hover { color: var(--text-secondary); }
  .tab.active { background: rgba(96,165,250,0.15); color: var(--accent-blue); }
  .search-box {
    flex: 1; max-width: 300px; padding: 7px 14px;
    background: rgba(255,255,255,0.04); border: 1px solid var(--border);
    border-radius: 8px; color: var(--text-primary); font-size: 13px;
    font-family: inherit; outline: none; transition: border-color 0.2s;
  }
  .search-box:focus { border-color: rgba(96,165,250,0.4); }
  .search-box::placeholder { color: var(--text-muted); }
  .route-filter {
    padding: 7px 12px; background: rgba(255,255,255,0.04);
    border: 1px solid var(--border); border-radius: 8px;
    color: var(--text-primary); font-size: 13px; font-family: inherit;
    outline: none; transition: border-color 0.2s; cursor: pointer;
    -webkit-appearance: none; appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%238892b0' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
    background-repeat: no-repeat; background-position: right 10px center;
    padding-right: 30px; min-width: 100px;
  }
  .route-filter:focus { border-color: rgba(96,165,250,0.4); }
  .route-filter option { background: #11182d; color: var(--text-primary); }
  .toolbar-right { margin-left: auto; display: flex; align-items: center; gap: 10px; }
  .count-info { font-size: 12px; color: var(--text-muted); }
  .refresh-btn {
    padding: 7px 14px; background: rgba(96,165,250,0.12);
    border: 1px solid rgba(96,165,250,0.25); color: var(--accent-blue);
    border-radius: 8px; cursor: pointer; font-size: 12px; font-weight: 500;
    font-family: inherit; transition: all 0.2s;
  }
  .refresh-btn:hover { background: rgba(96,165,250,0.2); transform: translateY(-1px); }

  .log-container { flex: 1; overflow-y: auto; }
  .log-table { width: 100%; border-collapse: collapse; font-size: 13px; }
  .log-table thead { position: sticky; top: 0; z-index: 5; }
  .log-table th {
    text-align: left; padding: 10px 14px; font-size: 11px; font-weight: 600;
    color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.6px;
    border-bottom: 1px solid var(--border); background: var(--bg-primary);
  }
  .log-table td {
    padding: 8px 14px; border-bottom: 1px solid rgba(99,115,168,0.06);
    vertical-align: middle; font-size: 12.5px;
  }
  .log-table tr:hover td { background: rgba(96,165,250,0.03); }
  .tag-direct { color: var(--accent-green); }
  .tag-block { color: var(--accent-red); }
  .tag-warp { color: var(--accent-cyan); }
  .level-Error { color: var(--accent-red); }
  .level-Warning { color: var(--accent-yellow); }
  .level-Info { color: var(--accent-green); }
  .domain-cell { font-family: 'JetBrains Mono', 'Fira Code', monospace; font-size: 12px; }
  .error-detail {
    font-size: 11px; color: var(--text-muted);
    max-width: 400px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  }
  .empty-state { text-align: center; padding: 60px 20px; color: var(--text-muted); font-size: 14px; }
  .loading { text-align: center; padding: 40px 20px; color: var(--text-muted); }
  .loading::after {
    content: ''; display: inline-block; width: 14px; height: 14px;
    border: 2px solid var(--text-muted); border-top-color: transparent;
    border-radius: 50%; animation: spin 0.8s linear infinite;
    margin-left: 8px; vertical-align: middle;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  @media (max-width: 768px) {
    .layout { flex-direction: column; }
    .sidebar {
      width: 100%; min-width: 100%;
      border-right: none; border-bottom: 1px solid var(--border);
      max-height: 140px;
    }
    .sidebar-header { padding: 12px 16px; }
    .sidebar-header h2 { font-size: 14px; }
    .user-list {
      display: flex; flex-wrap: wrap; gap: 4px;
      padding: 4px 8px; overflow-x: auto; overflow-y: hidden;
    }
    .user-item { padding: 6px 10px; margin-bottom: 0; white-space: nowrap; flex-shrink: 0; }
    .user-item .avatar { width: 24px; height: 24px; font-size: 11px; }
    .user-item .name { font-size: 12px; }
    .all-users-item { border-bottom: none; padding-bottom: 6px; margin-bottom: 0; border-right: 1px solid var(--border); padding-right: 8px; margin-right: 4px; }
    .toolbar { flex-wrap: wrap; padding: 10px 14px; gap: 8px; }
    .search-box { max-width: 100%; flex: 1 1 150px; }
    .toolbar-right { margin-left: 0; width: 100%; justify-content: space-between; }
    .log-container { overflow-x: auto; }
    .log-table { min-width: 600px; font-size: 12px; }
    .log-table th, .log-table td { padding: 6px 8px; }
    .domain-cell { font-size: 11px; }
    .error-detail { max-width: 200px; }
  }
  ::-webkit-scrollbar { width: 6px; }
  ::-webkit-scrollbar-track { background: transparent; }
  ::-webkit-scrollbar-thumb { background: rgba(99,115,168,0.2); border-radius: 3px; }
  ::-webkit-scrollbar-thumb:hover { background: rgba(99,115,168,0.35); }
</style>
</head>
<body>
<div class="layout">
  <div class="sidebar">
    <div class="sidebar-header">
      <h2>📋 用户日志</h2>
      <a href="/" class="back-link">← 返回面板</a>
    </div>
    <div class="user-list" id="userList"><div class="loading">加载中</div></div>
  </div>
  <div class="main">
    <div class="toolbar">
      <div class="tabs">
        <button class="tab active" onclick="switchTab('access',this)">访问日志</button>
        <button class="tab" onclick="switchTab('error',this)">错误日志</button>
      </div>
      <select class="route-filter" id="routeFilter" onchange="filterLogs()" style="display:none">
        <option value="">全部路由</option>
      </select>
      <input type="text" class="search-box" id="searchBox" placeholder="搜索域名..." oninput="filterLogs()">
      <div class="toolbar-right">
        <span class="count-info" id="countInfo"></span>
        <button class="refresh-btn" onclick="clearLogs()" style="background:rgba(248,113,113,0.1);border-color:rgba(248,113,113,0.25);color:#f87171">清除日志</button>
        <button class="refresh-btn" onclick="loadLogs()">刷新</button>
      </div>
    </div>
    <div class="log-container" id="logContainer">
      <div class="empty-state">选择用户查看访问日志</div>
    </div>
  </div>
</div>
<script>
const colors = ['#667eea','#f093fb','#4facfe','#43e97b','#fb923c','#f87171','#22d3ee','#a78bfa'];
function panelFetch(url, opts) {
  opts = opts || {};
  opts.headers = Object.assign({'X-Panel': '1'}, opts.headers || {});
  return fetch(url, opts);
}
let currentTab = 'access', currentUser = '', accessData = [], errorData = [], currentRoute = '';

async function loadUsers() {
  try {
    const res = await panelFetch('/api/users');
    const data = await res.json();
    const list = document.getElementById('userList');
    if (data.error) { list.innerHTML = '<div class="empty-state">' + esc(data.error) + '</div>'; return; }
    let h = '<div class="user-item all-users-item' + (currentUser===''?' active':'') + '" onclick="sel(\'\')">' +
      '<div class="avatar" style="background:var(--accent-purple)">A</div><div class="name">全部用户</div></div>';
    (data||[]).forEach((e,i) => {
      const c = colors[i%colors.length], a = currentUser===e?' active':'';
      h += '<div class="user-item'+a+'" onclick="sel(\''+esc(e)+'\')">' +
        '<div class="avatar" style="background:'+c+'">'+e.charAt(0).toUpperCase()+'</div>' +
        '<div class="name">'+esc(e)+'</div></div>';
    });
    list.innerHTML = h;
  } catch(e) { document.getElementById('userList').innerHTML = '<div class="empty-state">加载失败</div>'; }
}

function sel(email) { currentUser = email; loadUsers(); loadLogs(); }

function switchTab(tab, el) {
  currentTab = tab;
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  el.classList.add('active');
  document.getElementById('searchBox').placeholder = tab==='access'?'搜索域名...':'搜索域名或错误...';
  // Show/hide route filter dropdown (only for access logs)
  document.getElementById('routeFilter').style.display = tab==='access'?'':'none';
  renderLogs();
}

async function loadLogs() {
  document.getElementById('logContainer').innerHTML = '<div class="loading">加载中</div>';
  try {
    const [ar, er] = await Promise.all([
      panelFetch('/api/logs?count=3000' + (currentUser?'&email='+encodeURIComponent(currentUser):'')),
      panelFetch('/api/errors?count=1000')
    ]);
    const aj = await ar.json(), ej = await er.json();
    accessData = aj.error ? [] : (aj||[]);
    errorData = ej.error ? [] : (ej||[]);
    buildRouteFilter();
    renderLogs();
  } catch(e) {
    document.getElementById('logContainer').innerHTML = '<div class="empty-state">加载失败</div>';
  }
}

function filterLogs() {
  currentRoute = document.getElementById('routeFilter').value;
  renderLogs();
}

function buildRouteFilter() {
  const sel = document.getElementById('routeFilter');
  const routes = new Set();
  accessData.forEach(e => { if(e.route) routes.add(e.route); });
  let h = '<option value="">全部路由</option>';
  [...routes].sort().forEach(r => {
    h += '<option value="'+r+'"'+(currentRoute===r?' selected':'')+'>' + r + '</option>';
  });
  sel.innerHTML = h;
  sel.style.display = currentTab==='access'?'':'none';
}

function renderLogs() {
  const c = document.getElementById('logContainer'), s = document.getElementById('searchBox').value.toLowerCase();
  if (currentTab==='access') renderAccess(c,s); else renderErrors(c,s);
}

function renderAccess(container, search) {
  let f = accessData;
  if (currentRoute) f = f.filter(e => e.route === currentRoute);
  if (search) f = f.filter(e => e.target.toLowerCase().includes(search) || e.from_ip.includes(search));
  document.getElementById('countInfo').textContent = f.length + ' 条记录';
  if (!f.length) { container.innerHTML = '<div class="empty-state">暂无访问记录</div>'; return; }
  const r = [...f].reverse();
  let h = '<table class="log-table"><thead><tr><th>时间</th><th>来源IP</th><th>目标</th><th>路由</th>' +
    (currentUser===''?'<th>用户</th>':'') + '</tr></thead><tbody>';
  r.forEach(e => {
    h += '<tr><td style="white-space:nowrap;color:var(--text-muted)">'+esc(e.time)+'</td>' +
      '<td style="font-family:monospace;font-size:12px">'+esc(e.from_ip)+'</td>' +
      '<td class="domain-cell">'+esc(e.target)+'</td>' +
      '<td><span class="tag-'+e.route+'">'+esc(e.route)+'</span></td>' +
      (currentUser===''?'<td style="font-size:12px;color:var(--text-secondary)">'+esc(e.email)+'</td>':'') +
      '</tr>';
  });
  container.innerHTML = h + '</tbody></table>';
}

function renderErrors(container, search) {
  let f = errorData;
  if (search) f = f.filter(e => (e.domain&&e.domain.toLowerCase().includes(search)) ||
    e.message.toLowerCase().includes(search) || (e.error&&e.error.toLowerCase().includes(search)));
  document.getElementById('countInfo').textContent = f.length + ' 条记录';
  if (!f.length) { container.innerHTML = '<div class="empty-state">暂无错误日志</div>'; return; }
  const r = [...f].reverse();
  let h = '<table class="log-table"><thead><tr><th>时间</th><th>级别</th><th>模块</th><th>信息</th><th>域名</th><th>详情</th></tr></thead><tbody>';
  r.forEach(e => {
    h += '<tr><td style="white-space:nowrap;color:var(--text-muted)">'+esc(e.time)+'</td>' +
      '<td><span class="level-'+e.level+'">'+esc(e.level)+'</span></td>' +
      '<td style="font-size:12px;color:var(--text-secondary)">'+esc(e.module)+'</td>' +
      '<td>'+esc(e.message)+'</td>' +
      '<td class="domain-cell">'+esc(e.domain||'')+'</td>' +
      '<td class="error-detail" title="'+escA(e.error||'')+'">'+esc(e.error||'')+'</td></tr>';
  });
  container.innerHTML = h + '</tbody></table>';
}

function esc(s) { if(!s)return''; const d=document.createElement('div'); d.textContent=s; return d.innerHTML; }
function escA(s) { return esc(s).replace(/"/g,'&quot;'); }

loadUsers(); loadLogs();

async function clearLogs() {
  const type = currentTab === 'access' ? 'access' : 'error';
  const name = type === 'access' ? '访问日志' : '错误日志';
  if (!confirm('确认清除' + name + '？此操作不可恢复。')) return;
  try {
    const res = await panelFetch('/api/clear-logs', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({type: type})
    });
    const data = await res.json();
    if (data.error) { alert('清除失败: ' + data.error); return; }
    loadLogs();
  } catch(e) { alert('清除失败'); }
}
</script>
</body>
</html>`

const loginHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Xray Panel - Login</title>
<style>
  @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: 'Inter', sans-serif;
    background: #0a0e1a; color: #e2e8f0;
    min-height: 100vh;
    display: flex; align-items: center; justify-content: center;
  }
  body::before {
    content: ''; position: fixed;
    top: -50%; left: -50%; width: 200%; height: 200%;
    background:
      radial-gradient(ellipse at 30% 50%, rgba(99,102,241,0.1) 0%, transparent 50%),
      radial-gradient(ellipse at 70% 30%, rgba(139,92,246,0.08) 0%, transparent 50%);
    z-index: 0;
  }
  .login-card {
    position: relative; z-index: 1;
    background: rgba(17,24,45,0.9);
    border: 1px solid rgba(99,115,168,0.2);
    border-radius: 20px; padding: 40px; width: 360px;
    backdrop-filter: blur(20px);
    box-shadow: 0 20px 60px rgba(0,0,0,0.4);
  }
  .login-card .logo {
    width: 50px; height: 50px;
    background: linear-gradient(135deg, #667eea, #764ba2);
    border-radius: 14px;
    display: flex; align-items: center; justify-content: center;
    font-size: 24px; font-weight: 700;
    margin: 0 auto 20px;
    box-shadow: 0 4px 15px rgba(102,126,234,0.4);
  }
  .login-card h1 {
    text-align: center; font-size: 20px; font-weight: 600; margin-bottom: 8px;
    background: linear-gradient(135deg, #e2e8f0, #a78bfa);
    -webkit-background-clip: text; -webkit-text-fill-color: transparent;
  }
  .login-card .sub { text-align: center; font-size: 13px; color: #5a6480; margin-bottom: 28px; }
  .login-card input {
    width: 100%; padding: 12px 16px;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(99,115,168,0.2);
    border-radius: 10px; color: #e2e8f0;
    font-size: 14px; font-family: inherit;
    outline: none; transition: border-color 0.2s; margin-bottom: 16px;
  }
  .login-card input:focus { border-color: rgba(96,165,250,0.5); }
  .login-card input::placeholder { color: #5a6480; }
  .login-card button {
    width: 100%; padding: 12px;
    background: linear-gradient(135deg, #667eea, #764ba2);
    border: none; border-radius: 10px;
    color: white; font-size: 14px; font-weight: 600;
    cursor: pointer; font-family: inherit; transition: all 0.2s;
  }
  .login-card button:hover { transform: translateY(-1px); box-shadow: 0 4px 20px rgba(102,126,234,0.4); }
  .err { color: #f87171; font-size: 13px; text-align: center; margin-top: 12px; display: none; }
</style>
</head>
<body>
<div class="login-card">
  <div class="logo">X</div>
  <h1>Xray Panel</h1>
  <div class="sub">请输入管理密码</div>
  <input type="password" id="pwd" placeholder="密码" onkeydown="if(event.key==='Enter')doLogin()">
  <button onclick="doLogin()">登 录</button>
  <div class="err" id="err"></div>
</div>
<script>
async function doLogin() {
  const pwd = document.getElementById('pwd').value;
  const err = document.getElementById('err');
  err.style.display = 'none';
  try {
    const res = await fetch('/login', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({password: pwd})
    });
    const data = await res.json();
    if (data.error) { err.textContent = data.error; err.style.display = 'block'; }
    else {
      let nextUrl = '/';
      try {
        const u = new URL(window.location.href);
        if (u.searchParams.get('redirect')) {
            nextUrl = u.searchParams.get('redirect');
        }
      } catch(e) {}
      window.location.replace(nextUrl);
    }
  } catch(e) { err.textContent = '连接失败'; err.style.display = 'block'; }
}
</script>
</body>
</html>`
