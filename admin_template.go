package main

// Admin panel HTML — full management interface with logs
const adminHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Xray Admin Panel</title>
<script src="https://cdn.staticfile.net/qrcodejs/1.0.0/qrcode.min.js"></script>
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
    --accent-orange: #fb923c;
    --accent-red: #f87171;
    --accent-cyan: #22d3ee;
    --accent-yellow: #fbbf24;
    --gradient-1: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    --shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  }
  body {
    font-family: 'Inter', -apple-system, sans-serif;
    background: var(--bg-primary);
    color: var(--text-primary);
    min-height: 100vh;
  }
  body::before {
    content: ''; position: fixed; top: -50%; left: -50%;
    width: 200%; height: 200%;
    background:
      radial-gradient(ellipse at 20% 50%, rgba(99,102,241,0.08) 0%, transparent 50%),
      radial-gradient(ellipse at 80% 20%, rgba(139,92,246,0.06) 0%, transparent 50%);
    z-index: 0;
  }
  .container { position: relative; z-index: 1; max-width: 1200px; margin: 0 auto; padding: 24px 20px; }
  .header {
    display: flex; align-items: center; justify-content: space-between;
    margin-bottom: 24px; padding: 20px 28px;
    background: var(--bg-card); border: 1px solid var(--border);
    border-radius: 16px; backdrop-filter: blur(20px);
  }
  .header-left { display: flex; align-items: center; gap: 14px; }
  .logo {
    width: 42px; height: 42px; background: var(--gradient-1);
    border-radius: 12px; display: flex; align-items: center; justify-content: center;
    font-size: 20px; font-weight: 700; box-shadow: 0 4px 15px rgba(102,126,234,0.4);
  }
  .header h1 {
    font-size: 22px; font-weight: 700;
    background: linear-gradient(135deg, #e2e8f0, #a78bfa);
    -webkit-background-clip: text; -webkit-text-fill-color: transparent;
  }
  .back-link {
    color: var(--text-muted); text-decoration: none; font-size: 13px;
    padding: 8px 16px; border: 1px solid var(--border); border-radius: 8px;
    transition: all 0.2s;
  }
  .back-link:hover { color: var(--accent-blue); border-color: rgba(96,165,250,0.3); }
  .tabs-bar {
    display: flex; gap: 4px; margin-bottom: 24px; background: var(--bg-card);
    border: 1px solid var(--border); border-radius: 14px; padding: 6px;
    backdrop-filter: blur(20px); overflow-x: auto;
  }
  .tab-btn {
    padding: 10px 20px; border-radius: 10px; font-size: 13px; font-weight: 500;
    cursor: pointer; transition: all 0.2s; color: var(--text-muted);
    border: none; background: none; font-family: inherit; white-space: nowrap;
  }
  .tab-btn:hover { color: var(--text-secondary); }
  .tab-btn.active { background: rgba(96,165,250,0.15); color: var(--accent-blue); }
  .card {
    background: var(--bg-card); border: 1px solid var(--border);
    border-radius: 14px; backdrop-filter: blur(20px); margin-bottom: 20px; overflow: hidden;
  }
  .card-header {
    display: flex; align-items: center; justify-content: space-between;
    padding: 18px 22px; border-bottom: 1px solid var(--border); flex-wrap: wrap; gap: 10px;
  }
  .card-header h2 { font-size: 15px; font-weight: 600; display: flex; align-items: center; gap: 8px; }
  .card-body { padding: 20px 22px; }
  .data-table { width: 100%; border-collapse: collapse; }
  .data-table th {
    text-align: left; padding: 10px 12px; font-size: 11px; font-weight: 600;
    color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.8px;
    border-bottom: 1px solid var(--border);
  }
  .data-table td { padding: 12px; font-size: 13px; border-bottom: 1px solid rgba(99,115,168,0.08); }
  .data-table tr:hover td { background: rgba(96,165,250,0.04); }
  .btn {
    padding: 8px 18px; border-radius: 8px; font-size: 13px; font-weight: 500;
    cursor: pointer; border: none; font-family: inherit; transition: all 0.2s;
    display: inline-flex; align-items: center; gap: 6px;
  }
  .btn:hover { transform: translateY(-1px); }
  .btn-primary { background: rgba(96,165,250,0.15); color: var(--accent-blue); border: 1px solid rgba(96,165,250,0.25); }
  .btn-primary:hover { background: rgba(96,165,250,0.25); }
  .btn-danger { background: rgba(248,113,113,0.1); color: var(--accent-red); border: 1px solid rgba(248,113,113,0.25); }
  .btn-danger:hover { background: rgba(248,113,113,0.2); }
  .btn-success { background: rgba(52,211,153,0.12); color: var(--accent-green); border: 1px solid rgba(52,211,153,0.25); }
  .btn-success:hover { background: rgba(52,211,153,0.2); }
  .btn-orange { background: rgba(251,146,60,0.12); color: var(--accent-orange); border: 1px solid rgba(251,146,60,0.25); }
  .btn-orange:hover { background: rgba(251,146,60,0.2); }
  .btn:disabled { opacity: 0.5; cursor: not-allowed; transform: none; }
  .btn-group { display: flex; gap: 8px; flex-wrap: wrap; }
  .form-group { margin-bottom: 16px; }
  .form-label { font-size: 12px; color: var(--text-muted); margin-bottom: 6px; display: block; text-transform: uppercase; letter-spacing: 0.5px; }
  .form-input {
    width: 100%; padding: 10px 14px;
    background: rgba(255,255,255,0.04); border: 1px solid var(--border);
    border-radius: 8px; color: var(--text-primary); font-size: 14px;
    font-family: inherit; outline: none; transition: border-color 0.2s;
  }
  .form-input:focus { border-color: rgba(96,165,250,0.4); }
  .config-editor {
    width: 100%; min-height: 500px; padding: 16px;
    background: rgba(0,0,0,0.3); border: 1px solid var(--border);
    border-radius: 10px; color: var(--accent-green); font-size: 13px;
    font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
    line-height: 1.6; outline: none; resize: vertical; tab-size: 2;
  }
  .status-badge { display: inline-flex; align-items: center; gap: 6px; padding: 4px 12px; border-radius: 6px; font-size: 12px; font-weight: 600; }
  .status-active { background: rgba(52,211,153,0.15); color: var(--accent-green); }
  .status-inactive { background: rgba(248,113,113,0.15); color: var(--accent-red); }
  .cert-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 12px; }
  .cert-item { padding: 16px; background: rgba(255,255,255,0.02); border: 1px solid rgba(255,255,255,0.04); border-radius: 10px; }
  .cert-label { font-size: 11px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 6px; }
  .cert-value { font-size: 15px; font-weight: 600; }
  .cert-bar { height: 6px; background: rgba(255,255,255,0.06); border-radius: 3px; margin-top: 12px; overflow: hidden; }
  .cert-bar-fill { height: 100%; border-radius: 3px; transition: width 0.6s ease; }
  .toast {
    position: fixed; bottom: 24px; right: 24px; padding: 14px 22px;
    border-radius: 12px; font-size: 14px; font-weight: 500;
    z-index: 1000; opacity: 0; transform: translateY(20px);
    transition: all 0.3s; backdrop-filter: blur(20px);
  }
  .toast.show { opacity: 1; transform: translateY(0); }
  .toast-success { background: rgba(52,211,153,0.2); border: 1px solid rgba(52,211,153,0.3); color: var(--accent-green); }
  .toast-error { background: rgba(248,113,113,0.2); border: 1px solid rgba(248,113,113,0.3); color: var(--accent-red); }
  .modal-overlay {
    position: fixed; top: 0; left: 0; right: 0; bottom: 0;
    background: rgba(0,0,0,0.7); z-index: 100;
    display: none; align-items: center; justify-content: center;
    backdrop-filter: blur(4px);
  }
  .modal-overlay.show { display: flex; }
  .modal {
    background: var(--bg-card); border: 1px solid var(--border);
    border-radius: 16px; padding: 28px; width: 90%; max-width: 560px;
    max-height: 80vh; overflow-y: auto;
  }
  .modal h3 { font-size: 16px; margin-bottom: 16px; }
  .sub-link {
    padding: 12px; background: rgba(0,0,0,0.3); border-radius: 8px;
    font-family: monospace; font-size: 11px; word-break: break-all;
    color: var(--accent-cyan); margin-bottom: 12px; cursor: pointer;
    border: 1px solid var(--border); transition: border-color 0.2s;
  }
  .sub-link:hover { border-color: var(--accent-cyan); }
  .qr-container { display: flex; justify-content: center; margin: 16px 0; }
  .tab-panel { display: none; }
  .tab-panel.active { display: block; }
  .result-box { padding: 14px; border-radius: 8px; font-size: 13px; margin-top: 12px; font-family: monospace; white-space: pre-wrap; word-break: break-all; }
  .result-success { background: rgba(52,211,153,0.1); border: 1px solid rgba(52,211,153,0.2); color: var(--accent-green); }
  .result-error { background: rgba(248,113,113,0.1); border: 1px solid rgba(248,113,113,0.2); color: var(--accent-red); }
  /* Logs styles */
  .log-toolbar { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
  .log-tabs { display: flex; gap: 4px; background: rgba(255,255,255,0.04); border-radius: 10px; padding: 3px; }
  .log-tab { padding: 7px 16px; border-radius: 8px; font-size: 13px; font-weight: 500; cursor: pointer; transition: all 0.2s; color: var(--text-muted); border: none; background: none; font-family: inherit; }
  .log-tab:hover { color: var(--text-secondary); }
  .log-tab.active { background: rgba(96,165,250,0.15); color: var(--accent-blue); }
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
    outline: none; cursor: pointer;
    -webkit-appearance: none; appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%238892b0' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
    background-repeat: no-repeat; background-position: right 10px center;
    padding-right: 30px;
  }
  .route-filter:focus { border-color: rgba(96,165,250,0.4); }
  .route-filter option { background: #11182d; color: var(--text-primary); }
  .log-table { width: 100%; border-collapse: collapse; font-size: 13px; }
  .log-table thead { position: sticky; top: 0; z-index: 5; }
  .log-table th {
    text-align: left; padding: 10px 14px; font-size: 11px; font-weight: 600;
    color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.6px;
    border-bottom: 1px solid var(--border); background: var(--bg-primary);
  }
  .log-table td { padding: 8px 14px; border-bottom: 1px solid rgba(99,115,168,0.06); font-size: 12.5px; }
  .log-table tr:hover td { background: rgba(96,165,250,0.03); }
  .tag-direct { color: var(--accent-green); } .tag-block { color: var(--accent-red); } .tag-warp { color: var(--accent-cyan); }
  .level-Error { color: var(--accent-red); } .level-Warning { color: var(--accent-yellow); } .level-Info { color: var(--accent-green); }
  .domain-cell { font-family: 'JetBrains Mono', 'Fira Code', monospace; font-size: 12px; }
  .error-detail { font-size: 11px; color: var(--text-muted); max-width: 400px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .log-container { max-height: 600px; overflow-y: auto; overflow-x: auto; }
  .user-chips { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 12px; }
  .user-chip {
    padding: 6px 14px; border-radius: 8px; font-size: 12px; font-weight: 500;
    cursor: pointer; border: 1px solid var(--border); transition: all 0.2s;
    background: rgba(255,255,255,0.02); color: var(--text-secondary);
  }
  .user-chip:hover { border-color: rgba(96,165,250,0.3); }
  .user-chip.active { background: rgba(96,165,250,0.12); border-color: rgba(96,165,250,0.3); color: var(--accent-blue); }
  .loading-msg { text-align: center; padding: 40px; color: var(--text-muted); }
  .loading-msg::after {
    content: ''; display: inline-block; width: 14px; height: 14px;
    border: 2px solid var(--text-muted); border-top-color: transparent;
    border-radius: 50%; animation: spin 0.8s linear infinite;
    margin-left: 8px; vertical-align: middle;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  @media (max-width: 768px) {
    .tabs-bar { flex-wrap: nowrap; overflow-x: auto; }
    .tab-btn { padding: 8px 14px; font-size: 12px; }
    .cert-grid { grid-template-columns: 1fr 1fr; }
    .config-editor { min-height: 300px; font-size: 12px; }
    .log-table { min-width: 600px; }
  }
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <div class="header-left">
      <div class="logo">⚙</div>
      <h1>Admin Panel</h1>
    </div>
    <a href="/" class="back-link">← 返回面板</a>
  </div>

  <div class="tabs-bar">
    <button class="tab-btn active" onclick="switchAdminTab('users',this)">👤 用户</button>
    <button class="tab-btn" onclick="switchAdminTab('xray',this)">🔧 服务</button>
    <button class="tab-btn" onclick="switchAdminTab('config',this)">📝 配置</button>
    <button class="tab-btn" onclick="switchAdminTab('cert',this)">🔐 证书</button>
    <button class="tab-btn" onclick="switchAdminTab('telegram',this)">🤖 TG</button>
    <button class="tab-btn" onclick="switchAdminTab('subscribe',this)">🔗 订阅</button>
    <button class="tab-btn" onclick="switchAdminTab('logs',this)">📋 日志</button>
  </div>

  <!-- Users -->
  <div class="tab-panel active" id="panel-users">
    <div class="card">
      <div class="card-header">
        <h2>👤 用户管理</h2>
        <div class="btn-group">
          <button class="btn btn-primary" onclick="showAddUser()">+ 添加用户</button>
          <button class="btn btn-primary" onclick="loadAdminUsers()">刷新</button>
        </div>
      </div>
      <div class="card-body">
        <div id="addUserForm" style="display:none;margin-bottom:20px;padding:16px;background:rgba(255,255,255,0.02);border-radius:10px;border:1px solid var(--border)">
          <div class="form-group">
            <label class="form-label">Email</label>
            <input type="text" class="form-input" id="newEmail" placeholder="user@example.com">
          </div>
          <div class="btn-group">
            <button class="btn btn-success" onclick="addUser()">确认添加</button>
            <button class="btn btn-danger" onclick="hideAddUser()">取消</button>
          </div>
        </div>
        <div style="overflow-x:auto">
        <table class="data-table"><thead><tr><th>Email</th><th>UUID</th><th>操作</th></tr></thead>
        <tbody id="usersBody"><tr><td colspan="3" class="loading-msg">加载中</td></tr></tbody></table>
        </div>
      </div>
    </div>
  </div>

  <!-- Xray -->
  <div class="tab-panel" id="panel-xray">
    <div class="card">
      <div class="card-header">
        <h2>🔧 Xray 服务控制</h2>
        <span id="xrayStatus" class="status-badge status-inactive">检查中...</span>
      </div>
      <div class="card-body">
        <div class="btn-group" style="margin-bottom:16px">
          <button class="btn btn-orange" onclick="xrayAction('reload')">♻️ 重载配置</button>
          <button class="btn btn-danger" onclick="xrayAction('restart')">🔄 重启服务</button>
          <button class="btn btn-primary" onclick="checkXrayStatus()">📊 刷新状态</button>
        </div>
        <div id="xrayResult"></div>
      </div>
    </div>
  </div>

  <!-- Config -->
  <div class="tab-panel" id="panel-config">
    <div class="card">
      <div class="card-header">
        <h2>📝 配置文件编辑器</h2>
        <div class="btn-group">
          <button class="btn btn-success" onclick="saveConfig()">💾 保存</button>
          <button class="btn btn-primary" onclick="validateConfig()">✅ 验证</button>
          <button class="btn btn-orange" onclick="restoreConfig()">↩️ 还原</button>
          <button class="btn btn-primary" onclick="loadConfig()">🔄 刷新</button>
        </div>
      </div>
      <div class="card-body">
        <textarea class="config-editor" id="configEditor" spellcheck="false"></textarea>
        <div id="configResult"></div>
      </div>
    </div>
  </div>

  <!-- Cert -->
  <div class="tab-panel" id="panel-cert">
    <div class="card">
      <div class="card-header"><h2>🔐 证书状态</h2><button class="btn btn-primary" onclick="loadCert()">刷新</button></div>
      <div class="card-body"><div id="certInfo"><div style="color:var(--text-muted)">点击刷新加载</div></div></div>
    </div>
  </div>

  <!-- Telegram -->
  <div class="tab-panel" id="panel-telegram">
    <div class="card">
      <div class="card-header"><h2>🤖 Telegram Bot</h2></div>
      <div class="card-body">
        <div id="tgStatus"><div style="color:var(--text-muted)">点击 Tab 后加载</div></div>
        <div style="margin-top:16px"><button class="btn btn-primary" onclick="testTelegram()">📤 发送测试消息</button></div>
        <div id="tgResult"></div>
      </div>
    </div>
  </div>

  <!-- Subscribe -->
  <div class="tab-panel" id="panel-subscribe">
    <div class="card">
      <div class="card-header"><h2>🔗 订阅链接生成</h2><button class="btn btn-primary" onclick="loadSubUsers()">刷新</button></div>
      <div class="card-body">
        <table class="data-table"><thead><tr><th>用户</th><th>操作</th></tr></thead>
        <tbody id="subUsersBody"><tr><td colspan="2" style="color:var(--text-muted);text-align:center;padding:30px">切换到此 Tab 后加载</td></tr></tbody></table>
      </div>
    </div>
  </div>

  <!-- Logs -->
  <div class="tab-panel" id="panel-logs">
    <div class="card">
      <div class="card-header">
        <h2>📋 日志查看</h2>
        <div class="btn-group">
          <button class="btn btn-danger" onclick="clearLogs()">清除日志</button>
          <button class="btn btn-primary" onclick="loadLogData()">刷新</button>
        </div>
      </div>
      <div class="card-body">
        <div class="user-chips" id="logUserChips"></div>
        <div style="margin-top:-2px;margin-bottom:6px;display:none" id="logRouteWrap">
          <select class="route-filter" id="logRouteSelect" onchange="selLogRoute(this.value)">
            <option value="">全部路由</option>
          </select>
        </div>
        <div class="log-toolbar" style="margin-bottom:12px">
          <div class="log-tabs">
            <button class="log-tab active" onclick="switchLogTab('access',this)">访问日志</button>
            <button class="log-tab" onclick="switchLogTab('error',this)">错误日志</button>
          </div>
          <input type="text" class="search-box" id="logSearch" placeholder="搜索..." oninput="renderLogData()">
          <span style="font-size:12px;color:var(--text-muted)" id="logCount"></span>
        </div>
        <div class="log-container" id="logContainer"><div style="color:var(--text-muted);text-align:center;padding:30px">切换到此 Tab 后加载</div></div>
      </div>
    </div>
  </div>
</div>

<!-- Subscribe Modal -->
<div class="modal-overlay" id="subModal">
  <div class="modal">
    <h3 id="subModalTitle">订阅链接</h3>
    <div id="subModalContent"></div>
    <div style="margin-top:16px;text-align:right"><button class="btn btn-primary" onclick="closeSubModal()">关闭</button></div>
  </div>
</div>
<div class="toast" id="toast"></div>

<script>
function pf(url, opts) {
  opts = opts || {};
  if (!opts.headers) opts.headers = {};
  opts.headers['X-Panel'] = '1';
  opts.credentials = 'include';
  return fetch(url, opts);
}
function esc(s) { if(!s) return ''; const d=document.createElement('div'); d.textContent=s; return d.innerHTML; }
function escA(s) { return esc(s).replace(/"/g,'&quot;'); }
function showToast(msg, type) {
  const t = document.getElementById('toast');
  t.textContent = msg;
  t.className = 'toast toast-' + (type||'success') + ' show';
  setTimeout(() => t.className = 'toast', 3000);
}

function switchAdminTab(name, el) {
  document.querySelectorAll('.tab-btn').forEach(t => t.classList.remove('active'));
  document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
  el.classList.add('active');
  document.getElementById('panel-' + name).classList.add('active');
  if (name === 'users') loadAdminUsers();
  else if (name === 'xray') checkXrayStatus();
  else if (name === 'config') loadConfig();
  else if (name === 'cert') loadCert();
  else if (name === 'telegram') loadTelegramStatus();
  else if (name === 'subscribe') loadSubUsers();
  else if (name === 'logs') { loadLogUsers(); loadLogData(); }
}

// ===== Users =====
function showAddUser() { document.getElementById('addUserForm').style.display = 'block'; }
function hideAddUser() { document.getElementById('addUserForm').style.display = 'none'; document.getElementById('newEmail').value = ''; }

async function loadAdminUsers() {
  const body = document.getElementById('usersBody');
  body.innerHTML = '<tr><td colspan="3" class="loading-msg">加载中</td></tr>';
  try {
    const res = await pf('/admin/api/users');
    const data = await res.json();
    if (data && data.error) { body.innerHTML = '<tr><td colspan="3" style="color:var(--accent-red)">' + esc(data.error) + '</td></tr>'; return; }
    if (!data || !Array.isArray(data) || data.length === 0) { body.innerHTML = '<tr><td colspan="3" style="color:var(--text-muted);text-align:center;padding:30px">暂无用户</td></tr>'; return; }
    body.innerHTML = data.map(u =>
      '<tr><td style="font-weight:500">' + esc(u.email) + '</td>' +
      '<td style="font-family:monospace;font-size:12px;color:var(--text-secondary)">' + esc(u.id) + '</td>' +
      '<td><button class="btn btn-danger" onclick="deleteUser(\'' + esc(u.email) + '\')">删除</button></td></tr>'
    ).join('');
  } catch(e) { body.innerHTML = '<tr><td colspan="3" style="color:var(--accent-red)">加载失败: '+esc(e.message)+'</td></tr>'; console.error('loadUsers', e); }
}

async function addUser() {
  const email = document.getElementById('newEmail').value.trim();
  if (!email) { showToast('请输入 Email', 'error'); return; }
  const btn = event.target; btn.disabled = true; btn.textContent = '添加中...';
  hideAddUser();
  try {
    const res = await pf('/admin/api/users/add', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({email: email}) });
    const data = await res.json();
    if (data && data.error) { showToast(data.error, 'error'); } else { showToast('用户已添加'); }
  } catch(e) { showToast('操作失败: ' + e.message, 'error'); }
  btn.disabled = false; btn.textContent = '确认添加';
  loadAdminUsers();
}

async function deleteUser(email) {
  if (!confirm('确认删除用户 ' + email + '？')) return;
  showToast('删除中...');
  try {
    const res = await pf('/admin/api/users/delete', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({email: email}) });
    const data = await res.json();
    if (data && data.error) { showToast(data.error, 'error'); } else { showToast('用户 ' + email + ' 已删除'); }
  } catch(e) { showToast('操作失败: ' + e.message, 'error'); }
  loadAdminUsers();
}

// ===== Xray =====
async function checkXrayStatus() {
  try {
    const res = await pf('/admin/api/xray/status');
    const data = await res.json();
    const el = document.getElementById('xrayStatus');
    if (data.status === 'active') { el.className = 'status-badge status-active'; el.textContent = '● 运行中'; }
    else { el.className = 'status-badge status-inactive'; el.textContent = '● ' + (data.status || '未知'); }
  } catch(e) { console.error('xrayStatus', e); }
}
async function xrayAction(action) {
  const name = action === 'restart' ? '重启' : '重载';
  if (!confirm('确认' + name + ' Xray？')) return;
  document.getElementById('xrayResult').innerHTML = '<div class="result-box" style="background:rgba(96,165,250,0.1);border:1px solid rgba(96,165,250,0.2);color:var(--accent-blue)">正在' + name + '...</div>';
  try {
    const res = await pf('/admin/api/xray/' + action, { method: 'POST' });
    const data = await res.json();
    if (data && data.error) { document.getElementById('xrayResult').innerHTML = '<div class="result-box result-error">' + esc(data.error) + '</div>'; }
    else { document.getElementById('xrayResult').innerHTML = '<div class="result-box result-success">✅ ' + name + '成功</div>'; }
  } catch(e) { document.getElementById('xrayResult').innerHTML = '<div class="result-box result-error">操作失败: ' + esc(e.message) + '</div>'; }
  setTimeout(checkXrayStatus, 1000);
}

// ===== Config =====
async function loadConfig() {
  document.getElementById('configEditor').value = '加载中...';
  document.getElementById('configResult').innerHTML = '';
  try {
    const res = await pf('/admin/api/xconf');
    const data = await res.json();
    if (!data || data.error) { document.getElementById('configEditor').value = '错误: ' + ((data&&data.error)||'无响应'); return; }
    document.getElementById('configEditor').value = data.content || '';
  } catch(e) { document.getElementById('configEditor').value = '加载失败: ' + e.message; console.error('loadConfig', e); }
}
async function saveConfig() {
  if (!confirm('确认保存？（已自动备份为 .bak）')) return;
  try {
    const res = await pf('/admin/api/xconf/save', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({content: document.getElementById('configEditor').value}) });
    const data = await res.json();
    if (!data || data.error) { showToast((data&&data.error)||'保存失败', 'error'); return; }
    showToast('配置已保存');
  } catch(e) { showToast('保存失败: ' + e.message, 'error'); }
}
async function validateConfig() {
  const content = document.getElementById('configEditor').value;
  if (!content || !content.trim()) { showToast('配置内容为空', 'error'); return; }
  document.getElementById('configResult').innerHTML = '<div class="result-box" style="background:rgba(96,165,250,0.1);border:1px solid rgba(96,165,250,0.2);color:var(--accent-blue)">验证中...</div>';
  try {
    const res = await pf('/admin/api/xconf/validate', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({content: content}) });
    const data = await res.json();
    document.getElementById('configResult').innerHTML = data.valid
      ? '<div class="result-box result-success">✅ 语法正确\n' + esc(data.message) + '</div>'
      : '<div class="result-box result-error">❌ 语法错误\n' + esc(data.message) + '</div>';
  } catch(e) { document.getElementById('configResult').innerHTML = '<div class="result-box result-error">验证失败: ' + esc(e.message) + '</div>'; }
}
async function restoreConfig() {
  if (!confirm('确认从备份还原？')) return;
  try {
    const res = await pf('/admin/api/xconf/restore', { method: 'POST' });
    const data = await res.json();
    if (data.error) { showToast(data.error, 'error'); return; }
    showToast('已还原'); loadConfig();
  } catch(e) { showToast('还原失败', 'error'); }
}

// ===== Cert =====
async function loadCert() {
  document.getElementById('certInfo').innerHTML = '<div class="loading-msg">加载中</div>';
  try {
    const res = await pf('/admin/api/cert');
    const data = await res.json();
    if (data.error) { document.getElementById('certInfo').innerHTML = '<div style="color:var(--accent-red)">' + esc(data.error) + '</div>'; return; }
    let color = 'var(--accent-green)', statusText = '正常';
    if (data.is_expired) { color = 'var(--accent-red)'; statusText = '已过期'; }
    else if (data.is_expiring) { color = 'var(--accent-orange)'; statusText = '即将过期'; }
    const barPct = Math.min(100, Math.max(0, (data.days_left / 365) * 100));
    let h = '<div class="cert-grid">';
    h += '<div class="cert-item"><div class="cert-label">域名</div><div class="cert-value">' + esc(data.subject) + '</div></div>';
    h += '<div class="cert-item"><div class="cert-label">颁发者</div><div class="cert-value">' + esc(data.issuer) + '</div></div>';
    h += '<div class="cert-item"><div class="cert-label">生效</div><div class="cert-value" style="font-size:13px">' + esc(data.not_before) + '</div></div>';
    h += '<div class="cert-item"><div class="cert-label">到期</div><div class="cert-value" style="font-size:13px">' + esc(data.not_after) + '</div></div>';
    h += '<div class="cert-item"><div class="cert-label">剩余</div><div class="cert-value" style="color:'+color+'">' + data.days_left + ' 天 ('+statusText+')</div></div>';
    h += '<div class="cert-item"><div class="cert-label">SAN</div><div class="cert-value" style="font-size:12px">' + esc((data.dns_names||[]).join(', ')) + '</div></div>';
    h += '</div><div class="cert-bar"><div class="cert-bar-fill" style="width:'+barPct+'%;background:'+color+'"></div></div>';
    document.getElementById('certInfo').innerHTML = h;
  } catch(e) { document.getElementById('certInfo').innerHTML = '<div style="color:var(--accent-red)">加载失败: '+esc(e.message)+'</div>'; console.error('loadCert', e); }
}

// ===== Telegram =====
async function loadTelegramStatus() {
  document.getElementById('tgStatus').innerHTML = '<div class="loading-msg">加载中</div>';
  try {
    const res = await pf('/admin/api/telegram/status');
    const data = await res.json();
    if (data.configured) {
      document.getElementById('tgStatus').innerHTML = '<span class="status-badge status-active">● 已配置</span> Chat ID: <code>' + data.chat_id + '</code>';
    } else {
      document.getElementById('tgStatus').innerHTML = '<span class="status-badge status-inactive">● 未配置</span><p style="margin-top:8px;color:var(--text-muted);font-size:13px">在 config.json 中填入 telegram_token 和 telegram_chat_id 后重启服务</p>';
    }
  } catch(e) { document.getElementById('tgStatus').innerHTML = '<div style="color:var(--accent-red)">加载失败: '+esc(e.message)+'</div>'; console.error('tgStatus', e); }
}
async function testTelegram() {
  try {
    const res = await pf('/admin/api/telegram/test', { method: 'POST' });
    const data = await res.json();
    document.getElementById('tgResult').innerHTML = data.error
      ? '<div class="result-box result-error">' + esc(data.error) + '</div>'
      : '<div class="result-box result-success">✅ 测试消息已发送</div>';
  } catch(e) { document.getElementById('tgResult').innerHTML = '<div class="result-box result-error">发送失败</div>'; }
}

// ===== Subscribe =====
async function loadSubUsers() {
  const body = document.getElementById('subUsersBody');
  body.innerHTML = '<tr><td colspan="2" class="loading-msg">加载中</td></tr>';
  try {
    const res = await pf('/admin/api/users');
    const data = await res.json();
    if (data && data.error) { body.innerHTML = '<tr><td colspan="2" style="color:var(--accent-red)">' + esc(data.error) + '</td></tr>'; return; }
    if (!data || !Array.isArray(data) || data.length === 0) { body.innerHTML = '<tr><td colspan="2" style="color:var(--text-muted);text-align:center;padding:30px">暂无用户</td></tr>'; return; }
    body.innerHTML = data.map(u =>
      '<tr><td style="font-weight:500">' + esc(u.email) + '</td>' +
      '<td><button class="btn btn-primary" onclick="genSubscribe(\'' + esc(u.email) + '\')">🔗 生成链接</button></td></tr>'
    ).join('');
  } catch(e) { body.innerHTML = '<tr><td colspan="2" style="color:var(--accent-red)">加载失败</td></tr>'; }
}
async function genSubscribe(email) {
  document.getElementById('subModalTitle').textContent = '🔗 订阅 — ' + email;
  document.getElementById('subModalContent').innerHTML = '<div class="loading-msg">生成中</div>';
  document.getElementById('subModal').classList.add('show');
  try {
    const res = await pf('/admin/api/subscribe?email=' + encodeURIComponent(email));
    const data = await res.json();
    if (data.error) { document.getElementById('subModalContent').innerHTML = '<div style="color:var(--accent-red)">' + esc(data.error) + '</div>'; return; }
    let h = '';
    (data.links || []).forEach((link, idx) => {
      h += '<div style="margin-bottom:20px"><div style="font-size:13px;font-weight:600;margin-bottom:8px;color:var(--accent-purple)">' + esc(link.name) + '</div>';
      h += '<div class="sub-link" onclick="copyText(this.textContent)" title="点击复制">' + esc(link.uri) + '</div>';
      h += '<div class="qr-container"><div id="qr_' + idx + '"></div></div></div>';
    });
    if (data.combined_base64) {
      h += '<div style="margin-top:16px;padding-top:16px;border-top:1px solid var(--border)"><div style="font-size:12px;color:var(--text-muted);margin-bottom:8px">合并 Base64</div>';
      h += '<div class="sub-link" onclick="copyText(this.textContent)" title="点击复制" style="font-size:10px">' + esc(data.combined_base64) + '</div></div>';
    }
    document.getElementById('subModalContent').innerHTML = h;
    (data.links || []).forEach((link, idx) => {
      const el = document.getElementById('qr_' + idx);
      if (el && typeof QRCode !== 'undefined') { new QRCode(el, { text: link.uri, width: 160, height: 160, colorDark: '#e2e8f0', colorLight: '#0a0e1a', correctLevel: QRCode.CorrectLevel.M }); }
    });
  } catch(e) { document.getElementById('subModalContent').innerHTML = '<div style="color:var(--accent-red)">生成失败</div>'; }
}
function closeSubModal() { document.getElementById('subModal').classList.remove('show'); }
function copyText(text) { navigator.clipboard.writeText(text).then(() => showToast('已复制到剪贴板')); }

// ===== Logs =====
let logTab = 'access', logUser = '', logRoute = '', logAccessData = [], logErrorData = [];

async function loadLogUsers() {
  try {
    const res = await pf('/admin/api/users');
    const data = await res.json();
    const el = document.getElementById('logUserChips');
    if ((data && data.error) || !data || !Array.isArray(data)) { el.innerHTML = ''; return; }
    let h = '<div class="user-chip' + (logUser===''?' active':'') + '" data-loguser="">全部</div>';
    data.forEach(u => { const e = u.email || u; h += '<div class="user-chip' + (logUser===e?' active':'') + '" data-loguser="' + esc(e) + '">' + esc(String(e).split('@')[0]) + '</div>'; });
    el.innerHTML = h;
    el.querySelectorAll('.user-chip').forEach(chip => { chip.onclick = function() { selLogUser(this.getAttribute('data-loguser')); }; });
  } catch(e) { console.error('loadLogUsers', e); }
}
function selLogUser(email) { logUser = email; loadLogUsers(); loadLogData(); }

function switchLogTab(tab, el) {
  logTab = tab;
  document.querySelectorAll('.log-tab').forEach(t => t.classList.remove('active'));
  el.classList.add('active');
  loadLogRoutes();
  renderLogData();
}

function loadLogRoutes() {
  const wrap = document.getElementById('logRouteWrap');
  const sel = document.getElementById('logRouteSelect');
  if (logTab !== 'access' || !logAccessData || logAccessData.length === 0) { wrap.style.display = 'none'; return; }
  wrap.style.display = 'block';
  const routes = new Set(logAccessData.map(d => d.route).filter(Boolean));
  let h = '<option value="">全部路由</option>';
  Array.from(routes).sort().forEach(r => { h += '<option value="' + esc(r) + '"' + (logRoute===r?' selected':'') + '>' + esc(r) + '</option>'; });
  sel.innerHTML = h;
}
function selLogRoute(r) { logRoute = r; renderLogData(); }

async function loadLogData() {
  document.getElementById('logContainer').innerHTML = '<div class="loading-msg">加载中</div>';
  try {
    const [ar, er] = await Promise.all([
      pf('/api/logs?count=3000' + (logUser ? '&email=' + encodeURIComponent(logUser) : '')),
      pf('/api/errors?count=1000')
    ]);
    const aj = await ar.json(), ej = await er.json();
    let logErr = '';
    if (aj && aj.error) { logErr += aj.error; logAccessData = []; }
    else { logAccessData = Array.isArray(aj) ? aj : []; }
    if (ej && ej.error) { logErr += (logErr ? ' | ' : '') + ej.error; logErrorData = []; }
    else { logErrorData = Array.isArray(ej) ? ej : []; }
    if (logErr) { showToast(logErr, 'error'); }
    loadLogRoutes();
    renderLogData();
  } catch(e) { document.getElementById('logContainer').innerHTML = '<div style="color:var(--accent-red);text-align:center;padding:30px">加载失败: '+esc(e.message)+'</div>'; console.error('loadLogData', e); }
}

function renderLogData() {
  const c = document.getElementById('logContainer'), s = (document.getElementById('logSearch').value||'').toLowerCase();
  if (logTab === 'access') renderAccessLogs(c, s); else renderErrorLogs(c, s);
}

function renderAccessLogs(container, search) {
  let f = logAccessData;
  if (logRoute) f = f.filter(e => e.route === logRoute);
  if (search) f = f.filter(e => e.target.toLowerCase().includes(search) || e.from_ip.includes(search));
  document.getElementById('logCount').textContent = f.length + ' 条';
  if (!f.length) { container.innerHTML = '<div style="text-align:center;padding:30px;color:var(--text-muted)">暂无访问记录</div>'; return; }
  const r = [...f].reverse();
  let h = '<table class="log-table"><thead><tr><th>时间</th><th>来源IP</th><th>目标</th><th>路由</th>' + (logUser===''?'<th>用户</th>':'') + '</tr></thead><tbody>';
  r.forEach(e => {
    h += '<tr><td style="white-space:nowrap;color:var(--text-muted)">'+esc(e.time)+'</td>' +
      '<td style="font-family:monospace;font-size:12px">'+esc(e.from_ip)+'</td>' +
      '<td class="domain-cell">'+esc(e.target)+'</td>' +
      '<td><span class="tag-'+e.route+'">'+esc(e.route)+'</span></td>' +
      (logUser===''?'<td style="font-size:12px;color:var(--text-secondary)">'+esc(e.email)+'</td>':'') + '</tr>';
  });
  container.innerHTML = h + '</tbody></table>';
}

function renderErrorLogs(container, search) {
  let f = logErrorData;
  if (search) f = f.filter(e => (e.domain&&e.domain.toLowerCase().includes(search)) || e.message.toLowerCase().includes(search) || (e.error&&e.error.toLowerCase().includes(search)));
  document.getElementById('logCount').textContent = f.length + ' 条';
  if (!f.length) { container.innerHTML = '<div style="text-align:center;padding:30px;color:var(--text-muted)">暂无错误日志</div>'; return; }
  const r = [...f].reverse();
  let h = '<table class="log-table"><thead><tr><th>时间</th><th>级别</th><th>模块</th><th>信息</th><th>域名</th><th>详情</th></tr></thead><tbody>';
  r.forEach(e => {
    h += '<tr><td style="white-space:nowrap;color:var(--text-muted)">'+esc(e.time)+'</td>' +
      '<td><span class="level-'+e.level+'">'+esc(e.level)+'</span></td>' +
      '<td style="font-size:12px;color:var(--text-secondary)">'+esc(e.module)+'</td>' +
      '<td>'+esc(e.message)+'</td><td class="domain-cell">'+esc(e.domain||'')+'</td>' +
      '<td class="error-detail" title="'+escA(e.error||'')+'">'+esc(e.error||'')+'</td></tr>';
  });
  container.innerHTML = h + '</tbody></table>';
}

async function clearLogs() {
  const type = logTab === 'access' ? 'access' : 'error';
  if (!confirm('确认清除' + (type==='access'?'访问':'错误') + '日志？')) return;
  try {
    const res = await pf('/api/clear-logs', { method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify({type: type}) });
    const data = await res.json();
    if (data.error) { showToast('清除失败: ' + data.error, 'error'); return; }
    showToast('日志已清除'); loadLogData();
  } catch(e) { showToast('清除失败', 'error'); }
}

// ===== Init =====
loadAdminUsers();
</script>
</body>
</html>` + "\n"
