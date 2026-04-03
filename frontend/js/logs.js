
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
    const limit = document.getElementById('limitFilter').value || '3000';
    const [ar, er] = await Promise.all([
      panelFetch('/api/logs?count=' + limit + (currentUser?'&email='+encodeURIComponent(currentUser):'')),
      panelFetch('/api/errors?count=' + limit)
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

function exportLogs() {
  const isAccess = currentTab === 'access';
  let data = isAccess ? accessData : errorData;
  if (!data || data.length === 0) {
    alert('没有数据可导出');
    return;
  }
  
  if (isAccess && currentRoute) {
    data = data.filter(e => e.route === currentRoute);
  }
  const search = document.getElementById('searchBox').value.toLowerCase();
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
