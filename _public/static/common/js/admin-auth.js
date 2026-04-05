const APP_KEY_STORAGE = 'chatgpt2api_app_key';

function getStoredAppKey() {
  return localStorage.getItem(APP_KEY_STORAGE) || '';
}

function storeAppKey(key) {
  localStorage.setItem(APP_KEY_STORAGE, key);
}

function clearStoredAppKey() {
  localStorage.removeItem(APP_KEY_STORAGE);
}

async function ensureAdminKey() {
  const key = getStoredAppKey();
  if (!key) {
    window.location.href = '/admin/login';
    return null;
  }
  try {
    const res = await fetch('/v1/admin/verify', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${key}`
      }
    });
    if (!res.ok) {
      clearStoredAppKey();
      window.location.href = '/admin/login';
      return null;
    }
    return `Bearer ${key}`;
  } catch (e) {
    clearStoredAppKey();
    window.location.href = '/admin/login';
    return null;
  }
}

function buildAuthHeaders(apiKey) {
  if (!apiKey) return {};
  return { 'Authorization': apiKey };
}

function logout() {
  clearStoredAppKey();
  window.location.href = '/admin/login';
}
