let currentDailyEmail = '';
let currentDailyDays = 7;

function openDailyModal(email) {
  currentDailyEmail = email;
  currentDailyDays = 7;
  document.getElementById('dailyModalTitle').textContent = email + ' — 每日流量分析';
  document.getElementById('dailyModal').classList.add('active');
  // Reset tab active states
  document.querySelectorAll('.modal-tab').forEach((t, i) => {
    t.classList.toggle('active', i === 0);
  });
  fetchDailyTraffic(email, 7);
}

function closeDailyModal() {
  document.getElementById('dailyModal').classList.remove('active');
}

function switchDailyDays(days) {
  currentDailyDays = days;
  document.querySelectorAll('.modal-tab').forEach(t => {
    t.classList.toggle('active', t.textContent === days + '天');
  });
  fetchDailyTraffic(currentDailyEmail, days);
}

async function fetchDailyTraffic(email, days) {
  const tbody = document.getElementById('dailyTableBody');
  const summary = document.getElementById('dailySummary');
  tbody.innerHTML = '<tr><td colspan="4" class="loading">加载中</td></tr>';
  summary.innerHTML = '';

  try {
    const res = await panelFetch('/api/traffic-daily?email=' + encodeURIComponent(email) + '&days=' + days);
    const data = await res.json();
    if (data.error) {
      tbody.innerHTML = '<tr><td colspan="4" class="error-msg">' + escHtml(data.error) + '</td></tr>';
      return;
    }
    renderDailyData(data);
  } catch(e) {
    tbody.innerHTML = '<tr><td colspan="4" class="error-msg">加载失败</td></tr>';
  }
}

function renderDailyData(items) {
  // Summary
  let totalUp = 0, totalDown = 0;
  items.forEach(d => { totalUp += d.uplink; totalDown += d.downlink; });
  const totalAll = totalUp + totalDown;
  document.getElementById('dailySummary').innerHTML =
    '<div class="daily-summary-item"><div class="ds-label">周期上行</div><div class="ds-value" style="color:var(--accent-green)">' + formatBytes(totalUp) + '</div></div>' +
    '<div class="daily-summary-item"><div class="ds-label">周期下行</div><div class="ds-value" style="color:var(--accent-blue)">' + formatBytes(totalDown) + '</div></div>' +
    '<div class="daily-summary-item"><div class="ds-label">周期总量</div><div class="ds-value" style="color:var(--accent-cyan)">' + formatBytes(totalAll) + '</div></div>';

  // Table (newest first)
  const reversed = [...items].reverse();
  document.getElementById('dailyTableBody').innerHTML = reversed.map(d => {
    const dateStr = d.date.substring(5); // MM-DD
    return '<tr>' +
      '<td style="font-weight:500">' + d.date + '</td>' +
      '<td style="color:var(--accent-green)">' + formatBytes(d.uplink) + '</td>' +
      '<td style="color:var(--accent-blue)">' + formatBytes(d.downlink) + '</td>' +
      '<td style="font-weight:600">' + formatBytes(d.total) + '</td>' +
    '</tr>';
  }).join('');

  // Draw bar chart
  drawDailyChart(items);
}



// ESC to close modal
document.addEventListener('keydown', function(e) {
  if (e.key === 'Escape') {
    closeDailyModal();
    closeCycleModal();
  }
});

let currentCycleEmail = '';
async function openCycleModal(email) {
  currentCycleEmail = email;
  document.getElementById('cycleModalTitle').textContent = email + ' 周期';
  document.getElementById('cycleDayInput').value = '0';
  document.getElementById('cycleModal').classList.add('active');
  try {
    const res = await panelFetch('/api/cycle?email=' + encodeURIComponent(email));
    const d = await res.json();
    if (d && d.reset_day !== undefined) document.getElementById('cycleDayInput').value = d.reset_day;
  } catch(e) {}
}

function closeCycleModal() {
  document.getElementById('cycleModal').classList.remove('active');
}

async function saveCycle() {
  const day = parseInt(document.getElementById('cycleDayInput').value, 10);
  if (isNaN(day) || day < 0 || day > 31) return alert('请输入 0-31 之间的数字');
  try {
    const res = await panelFetch('/api/cycle', {
      method: 'POST',
      headers: {'Content-Type':'application/json'},
      body: JSON.stringify({email: currentCycleEmail, day: day})
    });
    const d = await res.json();
    if (d.error) alert(d.error); else { alert('设置成功'); closeCycleModal(); }
  } catch(e) { alert('设置失败: ' + e.message); }
}
