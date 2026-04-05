let apiKey = '';
let allTokens = [];
let currentFilter = 'all';

const byId = (id) => document.getElementById(id);

function setText(id, text) {
  const el = byId(id);
  if (el) el.innerText = text;
}

function openModal(id) {
  const modal = byId(id);
  if (!modal) return;
  modal.classList.remove('hidden');
  requestAnimationFrame(() => modal.classList.add('is-open'));
}

function closeModal(id) {
  const modal = byId(id);
  if (!modal) return;
  modal.classList.remove('is-open');
  setTimeout(() => modal.classList.add('hidden'), 200);
}

async function init() {
  apiKey = await ensureAdminKey();
  if (!apiKey) return;
  loadApiKeyConfig();
  loadData();
}

// API Key Settings
let apiKeyRevealed = false;
let currentMaskedKey = '';
let currentRawKey = '';

async function loadApiKeyConfig() {
  try {
    const res = await fetch('/v1/admin/config', {
      headers: buildAuthHeaders(apiKey)
    });
    if (!res.ok) return;
    const data = await res.json();
    currentMaskedKey = data.api_key || '';
    const statusEl = byId('api-key-status');
    const input = byId('api-key-input');
    if (data.api_key_set) {
      if (statusEl) {
        statusEl.innerHTML = '<span class="badge badge-green">已设置</span>';
      }
      if (input) {
        input.value = '';
        input.placeholder = currentMaskedKey + '（已设置，输入新值覆盖，提交空值清除）';
      }
    } else {
      if (statusEl) {
        statusEl.innerHTML = '<span class="badge badge-orange">未设置</span>';
      }
      if (input) {
        input.value = '';
        input.placeholder = '留空表示无需认证（支持逗号分隔多个 key）';
      }
    }
    apiKeyRevealed = false;
  } catch (e) {
    // ignore
  }
}

async function saveApiKey() {
  const input = byId('api-key-input');
  const newKey = input ? input.value.trim() : '';
  try {
    const res = await fetch('/v1/admin/config', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        ...buildAuthHeaders(apiKey)
      },
      body: JSON.stringify({ api_key: newKey })
    });
    if (res.ok) {
      showToast(newKey ? 'API Key 已更新' : 'API Key 已清除');
      loadApiKeyConfig();
    } else {
      const data = await res.json();
      showToast(data.error?.message || '保存失败', 'error');
    }
  } catch (e) {
    showToast('保存失败: ' + e.message, 'error');
  }
}

function toggleApiKeyVisibility() {
  // This is a simple toggle - since we only have masked key from server,
  // it just toggles between password and text type for user-entered values
  const input = byId('api-key-input');
  if (!input) return;
  if (input.type === 'password') {
    input.type = 'text';
  } else {
    input.type = 'password';
  }
}

async function loadData() {
  try {
    const res = await fetch('/v1/admin/tokens', {
      headers: buildAuthHeaders(apiKey)
    });
    if (res.ok) {
      const data = await res.json();
      processTokens(data.tokens);
      updateStats(data.stats);
      renderTable();
    } else if (res.status === 401) {
      logout();
    }
  } catch (e) {
    showToast('加载失败: ' + e.message, 'error');
  }
}

function processTokens(data) {
  allTokens = [];
  if (typeof data === 'object' && !Array.isArray(data)) {
    Object.keys(data).forEach(pool => {
      const tokens = data[pool];
      if (Array.isArray(tokens)) {
        tokens.forEach(t => {
          allTokens.push({
            token: t.token,
            status: t.status || 'active',
            note: t.note || '',
            use_count: t.use_count || 0,
            fail_count: t.fail_count || 0,
            _selected: false
          });
        });
      }
    });
  } else if (Array.isArray(data)) {
    data.forEach(t => {
      allTokens.push({
        token: t.token,
        status: t.status || 'active',
        note: t.note || '',
        use_count: t.use_count || 0,
        fail_count: t.fail_count || 0,
        _selected: false
      });
    });
  }
}

function updateStats(stats) {
  if (!stats) return;
  setText('stat-total', (stats.total || 0).toLocaleString());
  setText('stat-active', (stats.active || 0).toLocaleString());
  setText('stat-cooling', (stats.cooling || 0).toLocaleString());
  setText('stat-invalid', ((stats.disabled || 0) + (stats.expired || 0)).toLocaleString());
  updateTabCounts();
}

function updateTabCounts() {
  const counts = {
    all: allTokens.length,
    active: allTokens.filter(t => t.status === 'active').length,
    cooling: allTokens.filter(t => t.status === 'cooling').length,
    expired: allTokens.filter(t => t.status !== 'active' && t.status !== 'cooling').length,
  };
  Object.entries(counts).forEach(([key, count]) => {
    const el = byId(`tab-count-${key}`);
    if (el) el.textContent = count;
  });
}

function getFilteredTokens() {
  if (currentFilter === 'all') return allTokens;
  if (currentFilter === 'active') return allTokens.filter(t => t.status === 'active');
  if (currentFilter === 'cooling') return allTokens.filter(t => t.status === 'cooling');
  if (currentFilter === 'expired') return allTokens.filter(t => t.status !== 'active' && t.status !== 'cooling');
  return allTokens;
}

function filterByStatus(status) {
  currentFilter = status;
  document.querySelectorAll('.tab-item').forEach(tab => {
    const isActive = tab.dataset.filter === status;
    tab.classList.toggle('active', isActive);
  });
  renderTable();
}

function renderTable() {
  const tbody = byId('token-table-body');
  const loading = byId('loading');
  const emptyState = byId('empty-state');
  if (loading) loading.classList.add('hidden');

  const filtered = getFilteredTokens();

  if (filtered.length === 0) {
    tbody.replaceChildren();
    if (emptyState) emptyState.classList.remove('hidden');
    updateSelectionState();
    return;
  }
  if (emptyState) emptyState.classList.add('hidden');

  const fragment = document.createDocumentFragment();
  filtered.forEach((item) => {
    const originalIndex = allTokens.indexOf(item);
    const tr = document.createElement('tr');
    tr.dataset.index = originalIndex;
    if (item._selected) tr.classList.add('row-selected');

    // Checkbox
    const tdCheck = document.createElement('td');
    tdCheck.className = 'text-center';
    tdCheck.innerHTML = `<input type="checkbox" class="checkbox" ${item._selected ? 'checked' : ''} onchange="toggleSelect(${originalIndex})">`;

    // Token (masked)
    const tdToken = document.createElement('td');
    tdToken.className = 'text-left';
    const tokenShort = item.token.length > 16
      ? '***' + item.token.substring(item.token.length - 8)
      : item.token;
    tdToken.innerHTML = `
      <div class="flex items-center gap-2">
        <span class="font-mono text-xs text-gray-500" title="${escapeHtml(item.token)}">${escapeHtml(tokenShort)}</span>
      </div>`;

    // Status badge
    const tdStatus = document.createElement('td');
    tdStatus.className = 'text-center';
    let statusClass = 'badge-gray';
    if (item.status === 'active') statusClass = 'badge-green';
    else if (item.status === 'cooling') statusClass = 'badge-orange';
    else statusClass = 'badge-red';
    tdStatus.innerHTML = `<span class="badge ${statusClass}">${escapeHtml(item.status)}</span>`;

    // Use count
    const tdUseCount = document.createElement('td');
    tdUseCount.className = 'text-center font-mono text-xs';
    tdUseCount.innerText = item.use_count || 0;

    // Note
    const tdNote = document.createElement('td');
    tdNote.className = 'text-left text-gray-500 text-xs truncate max-w-[150px]';
    tdNote.innerText = item.note || '-';

    // Actions (delete only)
    const tdActions = document.createElement('td');
    tdActions.className = 'text-center';
    tdActions.innerHTML = `
      <button onclick="deleteToken(${originalIndex})" class="p-1 text-gray-400 hover:text-red-600 rounded" title="删除">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
      </button>`;

    tr.appendChild(tdCheck);
    tr.appendChild(tdToken);
    tr.appendChild(tdStatus);
    tr.appendChild(tdUseCount);
    tr.appendChild(tdNote);
    tr.appendChild(tdActions);
    fragment.appendChild(tr);
  });

  tbody.replaceChildren(fragment);
  updateSelectionState();
}

// Selection
function toggleSelectAll() {
  const checkbox = byId('select-all');
  const checked = !!(checkbox && checkbox.checked);
  getFilteredTokens().forEach(t => t._selected = checked);
  renderTable();
}

function toggleSelect(index) {
  allTokens[index]._selected = !allTokens[index]._selected;
  const row = document.querySelector(`#token-table-body tr[data-index="${index}"]`);
  if (row) row.classList.toggle('row-selected', allTokens[index]._selected);
  updateSelectionState();
}

function updateSelectionState() {
  const selectedCount = allTokens.filter(t => t._selected).length;
  const filtered = getFilteredTokens();
  const filteredSelected = filtered.filter(t => t._selected).length;

  const selectAll = byId('select-all');
  if (selectAll) {
    selectAll.checked = filtered.length > 0 && filteredSelected === filtered.length;
    selectAll.indeterminate = filteredSelected > 0 && filteredSelected < filtered.length;
  }

  const countEl = byId('selected-count');
  if (countEl) countEl.innerText = selectedCount;

  const batchBar = byId('batch-actions');
  if (batchBar) {
    batchBar.style.display = selectedCount > 0 ? 'flex' : 'none';
  }
}

// Add single token
function openAddModal() {
  byId('add-token-input').value = '';
  byId('add-note-input').value = '';
  openModal('add-modal');
}

function closeAddModal() {
  closeModal('add-modal');
}

async function submitAdd() {
  const token = byId('add-token-input').value.trim();
  const note = byId('add-note-input').value.trim();
  if (!token) {
    showToast('请输入 Token', 'error');
    return;
  }
  try {
    const res = await fetch('/v1/admin/tokens', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...buildAuthHeaders(apiKey)
      },
      body: JSON.stringify({ token, note })
    });
    if (res.ok) {
      showToast('添加成功');
      closeAddModal();
      loadData();
    } else {
      const data = await res.json();
      showToast(data.error?.message || '添加失败', 'error');
    }
  } catch (e) {
    showToast('添加失败: ' + e.message, 'error');
  }
}

// Bulk import
function openImportModal() {
  byId('import-text').value = '';
  byId('import-note-input').value = '';
  openModal('import-modal');
}

function closeImportModal() {
  closeModal('import-modal');
}

async function submitImport() {
  const text = byId('import-text').value.trim();
  const note = byId('import-note-input').value.trim();
  if (!text) {
    showToast('请输入 Token', 'error');
    return;
  }
  const tokens = text.split('\n').map(s => s.trim()).filter(Boolean);
  if (tokens.length === 0) {
    showToast('没有有效的 Token', 'error');
    return;
  }

  let success = 0, fail = 0;
  for (const token of tokens) {
    try {
      const res = await fetch('/v1/admin/tokens', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...buildAuthHeaders(apiKey)
        },
        body: JSON.stringify({ token, note })
      });
      if (res.ok) success++;
      else fail++;
    } catch (e) {
      fail++;
    }
  }

  showToast(`导入完成: 成功 ${success}, 失败 ${fail}`);
  closeImportModal();
  loadData();
}

// Delete token
let confirmResolver = null;

function confirmAction(message) {
  return new Promise(resolve => {
    confirmResolver = resolve;
    byId('confirm-message').textContent = message;
    openModal('confirm-dialog');
  });
}

function closeConfirm(ok) {
  closeModal('confirm-dialog');
  if (confirmResolver) {
    confirmResolver(ok);
    confirmResolver = null;
  }
}

async function deleteToken(index) {
  const token = allTokens[index];
  if (!token) return;
  const ok = await confirmAction('确定要删除此 Token 吗？');
  if (!ok) return;

  try {
    const res = await fetch('/v1/admin/tokens', {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        ...buildAuthHeaders(apiKey)
      },
      body: JSON.stringify({ token: token.token })
    });
    if (res.ok) {
      showToast('删除成功');
      loadData();
    } else {
      showToast('删除失败', 'error');
    }
  } catch (e) {
    showToast('删除失败: ' + e.message, 'error');
  }
}

async function batchDelete() {
  const selected = allTokens.filter(t => t._selected);
  if (selected.length === 0) {
    showToast('请先选择 Token', 'error');
    return;
  }
  const ok = await confirmAction(`确定要删除 ${selected.length} 个 Token 吗？`);
  if (!ok) return;

  let success = 0, fail = 0;
  for (const item of selected) {
    try {
      const res = await fetch('/v1/admin/tokens', {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          ...buildAuthHeaders(apiKey)
        },
        body: JSON.stringify({ token: item.token })
      });
      if (res.ok) success++;
      else fail++;
    } catch (e) {
      fail++;
    }
  }
  showToast(`删除完成: 成功 ${success}, 失败 ${fail}`);
  loadData();
}

function escapeHtml(text) {
  if (!text) return '';
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

window.onload = init;
