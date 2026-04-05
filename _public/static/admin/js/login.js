const apiKeyInput = document.getElementById('api-key-input');
if (apiKeyInput) {
  apiKeyInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') login();
  });
}

async function requestLogin(key) {
  const res = await fetch('/v1/admin/verify', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${key}`
    }
  });
  return res.ok;
}

async function login() {
  const input = (apiKeyInput ? apiKeyInput.value : '').trim();
  if (!input) return;
  try {
    const ok = await requestLogin(input);
    if (ok) {
      storeAppKey(input);
      window.location.href = '/admin/token';
    } else {
      showToast('密码错误', 'error');
    }
  } catch (e) {
    showToast('连接失败', 'error');
  }
}

(async () => {
  const existingKey = getStoredAppKey();
  if (!existingKey) return;
  try {
    const ok = await requestLogin(existingKey);
    if (ok) window.location.href = '/admin/token';
  } catch (e) {}
})();
