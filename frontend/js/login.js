
async function doLogin(e) {
  if (e) e.preventDefault();
  const pwd = document.getElementById('password').value;
  const errEl = document.getElementById('errorMsg');
  errEl.textContent = '';
  const btn = document.getElementById('loginBtn');
  btn.disabled = true;
  btn.textContent = '登录中...';
  try {
    const res = await fetch('/login', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({password: pwd})
    });
    const data = await res.json();
    if (data.error) {
      errEl.textContent = data.error;
      btn.disabled = false;
      btn.textContent = '登 录';
    } else {
      let nextUrl = '/';
      try {
        const u = new URL(window.location.href);
        if (u.searchParams.get('redirect')) {
          nextUrl = u.searchParams.get('redirect');
        }
      } catch(ex) {}
      window.location.replace(nextUrl);
    }
  } catch(ex) {
    errEl.textContent = '连接失败';
    btn.disabled = false;
    btn.textContent = '登 录';
  }
}
