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


function drawDailyChart(items) {
  const canvas = document.getElementById('dailyChart');
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
  const pad = {top: 20, right: 20, bottom: 40, left: 60};
  const cw = w - pad.left - pad.right;
  const ch = h - pad.top - pad.bottom;

  ctx.clearRect(0, 0, w, h);

  if (items.length === 0) {
    ctx.fillStyle = '#5a6480';
    ctx.font = '13px Inter, sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText('暂无数据', w/2, h/2);
    return;
  }

  const maxVal = Math.max(...items.map(d => d.total), 1);
  const barGroupWidth = cw / items.length;
  const barWidth = Math.min(barGroupWidth * 0.7, 30);
  const halfBar = barWidth / 2;

  // Grid lines
  ctx.strokeStyle = 'rgba(99,115,168,0.1)';
  ctx.lineWidth = 1;
  for (let i = 0; i <= 4; i++) {
    const y = pad.top + (ch / 4) * i;
    ctx.beginPath();
    ctx.moveTo(pad.left, y);
    ctx.lineTo(pad.left + cw, y);
    ctx.stroke();
    const val = maxVal - (maxVal / 4) * i;
    ctx.fillStyle = '#5a6480';
    ctx.font = '10px Inter, sans-serif';
    ctx.textAlign = 'right';
    ctx.fillText(formatBytesShort(val), pad.left - 8, y + 4);
  }

  // Bars
  items.forEach((d, i) => {
    const cx = pad.left + barGroupWidth * i + barGroupWidth / 2;
    const upH = (d.uplink / maxVal) * ch;
    const downH = (d.downlink / maxVal) * ch;

    // Download bar (left half)
    const dlGrad = ctx.createLinearGradient(0, pad.top + ch - downH, 0, pad.top + ch);
    dlGrad.addColorStop(0, 'rgba(96,165,250,0.9)');
    dlGrad.addColorStop(1, 'rgba(96,165,250,0.3)');
    ctx.fillStyle = dlGrad;
    ctx.beginPath();
    const dlX = cx - halfBar;
    const dlY = pad.top + ch - downH;
    ctx.roundRect(dlX, dlY, halfBar - 1, downH, [3, 3, 0, 0]);
    ctx.fill();

    // Upload bar (right half)
    const ulGrad = ctx.createLinearGradient(0, pad.top + ch - upH, 0, pad.top + ch);
    ulGrad.addColorStop(0, 'rgba(52,211,153,0.9)');
    ulGrad.addColorStop(1, 'rgba(52,211,153,0.3)');
    ctx.fillStyle = ulGrad;
    ctx.beginPath();
    const ulX = cx + 1;
    ctx.roundRect(ulX, pad.top + ch - upH, halfBar - 1, upH, [3, 3, 0, 0]);
    ctx.fill();

    // X label
    if (items.length <= 14 || i % Math.ceil(items.length / 10) === 0) {
      ctx.fillStyle = '#5a6480';
      ctx.font = '9px Inter, sans-serif';
      ctx.textAlign = 'center';
      ctx.save();
      ctx.translate(cx, pad.top + ch + 14);
      ctx.rotate(-0.4);
      ctx.fillText(d.date.substring(5), 0, 0);
      ctx.restore();
    }
  });

  // Legend
  ctx.font = '11px Inter, sans-serif';
  ctx.textAlign = 'left';
  ctx.fillStyle = '#60a5fa';
  ctx.fillRect(pad.left + 10, pad.top + 2, 12, 3);
  ctx.fillText('下行', pad.left + 26, pad.top + 8);
  ctx.fillStyle = '#34d399';
  ctx.fillRect(pad.left + 70, pad.top + 2, 12, 3);
  ctx.fillText('上行', pad.left + 86, pad.top + 8);
}
