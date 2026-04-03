
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
      '<td><div class="action-group">' +
        '<button class="btn-sm btn-sm-danger" onclick="deleteUser(\'' + esc(u.email) + '\')">删除</button>' +
        '<button class="btn-sm btn-sm-purple" onclick="openCycleModal(\'' + esc(u.email) + '\')">⚙ 周期</button>' +
        '<button class="btn-sm btn-sm-orange" onclick="resetUser(\'' + esc(u.email) + '\')">↻ 清零</button>' +
      '</div></td></tr>'
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

async function resetUser(email) {
  if (!confirm('确认清零用户 ' + email + ' 的流量统计？')) return;
  try {
    const res = await pf('/api/reset', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({email: email})
    });
    const data = await res.json();
    if (data && data.error) { showToast(data.error, 'error'); } else { showToast('用户 ' + email + ' 流量已清零'); }
  } catch(e) { showToast('功能执行失败: ' + e.message, 'error'); }
}

let currentCycleEmail = '';
async function openCycleModal(email) {
  currentCycleEmail = email;
  document.getElementById('cycleModalTitle').textContent = email + ' 周期';
  document.getElementById('cycleDayInput').value = '0';
  document.getElementById('cycleModal').classList.add('show');
  try {
    const res = await pf('/api/cycle?email=' + encodeURIComponent(email));
    const d = await res.json();
    if (d && d.reset_day !== undefined) document.getElementById('cycleDayInput').value = d.reset_day;
  } catch(e) {}
}

function closeCycleModal() {
  document.getElementById('cycleModal').classList.remove('show');
}

async function saveCycle() {
  const day = parseInt(document.getElementById('cycleDayInput').value, 10);
  if (isNaN(day) || day < 0 || day > 31) return showToast('请输入 0-31 之间的数字', 'error');
  try {
    const res = await pf('/api/cycle', {
      method: 'POST',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify({email: currentCycleEmail, day: day})
    });
    const d = await res.json();
    if (d.error) showToast(d.error, 'error'); else { showToast('周期设置成功'); closeCycleModal(); }
  } catch(e) { showToast('设置失败: ' + e.message, 'error'); }
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
  // 同时获取版本信息
  checkVersion();
}
async function checkVersion(refresh) {
  try {
    const url = '/admin/api/version' + (refresh ? '?refresh=1' : '');
    const res = await pf(url);
    const v = await res.json();
    const el = document.getElementById('xrayVersion');
    if (!el) return;
    if (v.error && !v.current) { el.innerHTML = '<span style="color:var(--text-muted)">版本信息不可用</span>'; return; }
    let html = '<div style="display:flex;gap:16px;align-items:center;flex-wrap:wrap">';
    html += '<span>当前: <strong>' + esc(v.current || '未知') + '</strong></span>';
    html += '<span>最新: <strong>' + esc(v.latest || '检查中...') + '</strong></span>';
    if (v.has_update) {
      html += '<button class="btn-sm btn-sm-blue" onclick="xrayUpdate()" id="updateBtn">⬆ 更新到 ' + esc(v.latest) + '</button>';
    } else if (v.latest) {
      html += '<span class="status-badge status-active">✓ 已是最新</span>';
    }
    html += '<button class="btn-sm" style="font-size:11px" onclick="checkVersion(true)">↻ 刷新</button>';
    html += '</div>';
    if (v.checked_at) html += '<div style="font-size:11px;color:var(--text-muted);margin-top:4px">上次检查: ' + esc(v.checked_at) + '</div>';
    el.innerHTML = html;
  } catch(e) { console.error('checkVersion', e); }
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
async function xrayUpdate() {
  if (!confirm('确认更新 Xray-core 到最新版？\n更新将执行官方安装脚本并自动重启服务。')) return;
  const btn = document.getElementById('updateBtn');
  if (btn) { btn.disabled = true; btn.textContent = '更新中...'; }
  document.getElementById('xrayResult').innerHTML = '<div class="result-box" style="background:rgba(96,165,250,0.1);border:1px solid rgba(96,165,250,0.2);color:var(--accent-blue)">⏳ 正在更新，请稍候 (可能需要1-2分钟)...</div>';
  try {
    const res = await pf('/admin/api/xray-update', { method: 'POST' });
    const data = await res.json();
    if (data.success) {
      document.getElementById('xrayResult').innerHTML = '<div class="result-box result-success">✅ 更新完成<pre style="margin-top:8px;font-size:11px;white-space:pre-wrap;max-height:200px;overflow:auto">' + esc(data.output || '') + '</pre></div>';
      showToast('Xray 更新成功');
    } else {
      document.getElementById('xrayResult').innerHTML = '<div class="result-box result-error">❌ 更新失败: ' + esc(data.error || '') + '<pre style="margin-top:8px;font-size:11px;white-space:pre-wrap;max-height:200px;overflow:auto">' + esc(data.output || '') + '</pre></div>';
    }
  } catch(e) {
    document.getElementById('xrayResult').innerHTML = '<div class="result-box result-error">操作失败: ' + esc(e.message) + '</div>';
  }
  if (btn) { btn.disabled = false; btn.textContent = '⬆ 更新'; }
  setTimeout(() => { checkXrayStatus(); checkVersion(true); }, 2000);
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
      '<td><button class="btn-sm btn-sm-blue" onclick="genSubscribe(\'' + esc(u.email) + '\')">🔗 生成链接</button></td></tr>'
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
    const limit = document.getElementById('logLimitSelect').value || '3000';
    const [ar, er] = await Promise.all([
      pf('/api/logs?count=' + limit + (logUser ? '&email=' + encodeURIComponent(logUser) : '')),
      pf('/api/errors?count=' + limit)
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

function exportLogs() {
  const isAccess = logTab === 'access';
  let data = isAccess ? logAccessData : logErrorData;
  if (!data || data.length === 0) {
    showToast('没有数据可导出', 'error');
    return;
  }
  
  if (isAccess && logRoute) {
    data = data.filter(e => e.route === logRoute);
  }
  const search = (document.getElementById('logSearch').value||'').toLowerCase();
  if (search) {
    if (isAccess) {
      data = data.filter(e => e.target.toLowerCase().includes(search) || e.from_ip.includes(search));
    } else {
      data = data.filter(e => (e.domain&&e.domain.toLowerCase().includes(search)) || e.message.toLowerCase().includes(search) || (e.error&&e.error.toLowerCase().includes(search)));
    }
  }

  let csvContent = "";
  let keys = [];
  if (isAccess) {
    keys = ["Time", "FromIP", "Target", "Route", "Email"];
  } else {
    keys = ["Time", "Level", "Module", "Message", "Domain", "Error"];
  }
  csvContent += keys.join(",") + "\r\n";
  
  data.forEach(function(rowArray) {
    let row = keys.map(k => {
      let keyToVal = isAccess ? {
        "Time": rowArray.time, "FromIP": rowArray.from_ip, "Target": rowArray.target, "Route": rowArray.route, "Email": (rowArray.email || "")
      } : {
        "Time": rowArray.time, "Level": rowArray.level, "Module": rowArray.module, "Message": rowArray.message, "Domain": (rowArray.domain || ""), "Error": (rowArray.error || "")
      };
      let cell = keyToVal[k] === null || keyToVal[k] === undefined ? "" : String(keyToVal[k]);
      return `"${cell.replace(/"/g, '""')}"`;
    });
    csvContent += row.join(",") + "\r\n";
  });
  
  const blob = new Blob(["\uFEFF" + csvContent], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.setAttribute("href", url);
  link.setAttribute("download", (isAccess ? "access_logs_" : "error_logs_") + new Date().getTime() + ".csv");
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

// ===== Init =====
loadAdminUsers();
