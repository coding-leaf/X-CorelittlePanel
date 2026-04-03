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
