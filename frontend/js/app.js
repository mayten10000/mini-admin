document.addEventListener('DOMContentLoaded', () => {
  const loginPage = document.getElementById('login-page');
  const appPage = document.getElementById('app-page');
  const headerUser = document.getElementById('header-user');
  const toastContainer = document.getElementById('toast-container');

  let currentUser = null;
  let currentView = 'list';
  let listParams = { search: '', status: '', sort_by: 'id', order: 'asc', page: 1, per_page: 20 };
  let debounceTimer = null;

  function toast(msg, type = 'success') {
    const el = document.createElement('div');
    el.className = `toast toast-${type}`;
    el.textContent = msg;
    toastContainer.appendChild(el);
    setTimeout(() => el.remove(), 3500);
  }

  function showLogin() {
    loginPage.classList.remove('hidden');
    appPage.classList.remove('active');
    document.getElementById('login-email').value = '';
    document.getElementById('login-password').value = '';
    document.getElementById('login-error').textContent = '';
  }

  function showApp() {
    loginPage.classList.add('hidden');
    appPage.classList.add('active');
  }

  window.addEventListener('auth:logout', () => {
    currentUser = null;
    showLogin();
    toast('Session expired, please log in again', 'error');
  });

  document.getElementById('login-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const email = document.getElementById('login-email').value.trim();
    const password = document.getElementById('login-password').value;
    const errorEl = document.getElementById('login-error');
    const btn = document.getElementById('login-btn');

    errorEl.textContent = '';
    btn.disabled = true;
    btn.textContent = 'Signing in…';

    try {
      const data = await API.post('/api/auth/login', { email, password });
      API.setTokens(data.access_token, data.refresh_token);
      currentUser = await API.get('/api/auth/me');
      headerUser.textContent = currentUser.name;
      showApp();
      navigateTo('list');
    } catch (err) {
      errorEl.textContent = err.message || 'Login failed';
    } finally {
      btn.disabled = false;
      btn.textContent = 'Sign In';
    }
  });

  document.getElementById('logout-btn').addEventListener('click', async () => {
    try {
      await API.post('/api/auth/logout', { refresh_token: API.getRefreshToken() });
    } catch {}
    API.clearTokens();
    currentUser = null;
    showLogin();
  });

  function navigateTo(view, data) {
    currentView = view;
    const content = document.getElementById('content');
    switch (view) {
      case 'list':   renderList(content); break;
      case 'view':   renderView(content, data); break;
      case 'create': renderForm(content, null); break;
      case 'edit':   renderForm(content, data); break;
      case 'analyze': renderAnalyze(content); break;
    }
  }

  async function renderAnalyze(container) {
    container.innerHTML = `
      <div class="toolbar">
        <div class="toolbar-filters">
          <a class="back-link" id="ai-back">← Back to list</a>
        </div>
        <div class="toolbar-actions">
          <button class="btn btn-primary" id="ai-run">Проанализировать пользователей</button>
        </div>
      </div>
      <div class="ai-meta" id="ai-meta"></div>
      <div class="card">
        <div class="table-wrapper">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Имя</th>
                <th>Email</th>
                <th>Статус</th>
                <th>Создан</th>
                <th>Обновлён</th>
                <th>Уровень риска</th>
                <th>Комментарий ИИ</th>
                <th>Рекомендуемое действие</th>
              </tr>
            </thead>
            <tbody id="ai-tbody">
              <tr><td colspan="9" style="text-align:center;padding:40px;color:var(--text-muted)">Нажмите «Проанализировать пользователей», чтобы запустить ИИ.</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    `;

    document.getElementById('ai-back').onclick = (e) => { e.preventDefault(); navigateTo('list'); };

    const runBtn = document.getElementById('ai-run');
    const tbody = document.getElementById('ai-tbody');
    const meta = document.getElementById('ai-meta');

    runBtn.onclick = async () => {
      runBtn.disabled = true;
      const oldLabel = runBtn.textContent;
      runBtn.textContent = 'Анализирую…';
      meta.textContent = '';
      tbody.innerHTML = `<tr><td colspan="9" style="text-align:center;padding:40px;color:var(--text-muted)">Идёт анализ, подождите…</td></tr>`;

      try {
        const data = await API.post('/api/v1/ai/users/analyze', {});
        const results = data.results || [];
        if (results.length === 0) {
          tbody.innerHTML = `<tr><td colspan="9" style="text-align:center;padding:40px;color:var(--text-muted)">Нет данных для анализа</td></tr>`;
        } else {
          tbody.innerHTML = results.map(u => `
            <tr>
              <td>${u.id}</td>
              <td>${esc(u.name)}</td>
              <td>${esc(u.email)}</td>
              <td><span class="badge badge-${u.status}">${u.status}</span></td>
              <td>${formatDate(u.created_at)}</td>
              <td>${formatDate(u.updated_at)}</td>
              <td><span class="badge badge-${esc(u.risk_level)}">${esc(u.risk_level)}</span></td>
              <td>${esc(u.comment)}</td>
              <td>${esc(u.recommended_action)}</td>
            </tr>
          `).join('');
        }
        meta.textContent = `Проанализировано: ${data.total} • модель: ${data.model || '—'}`;
      } catch (err) {
        tbody.innerHTML = `<tr><td colspan="9" style="text-align:center;padding:40px;color:var(--danger)">${esc(err.message)}</td></tr>`;
      } finally {
        runBtn.disabled = false;
        runBtn.textContent = oldLabel;
      }
    };
  }

  async function renderList(container) {
    container.innerHTML = `
      <div class="toolbar">
        <div class="toolbar-filters">
          <input type="text" class="form-input" id="filter-search" placeholder="Search by name or email…" value="${listParams.search}">
          <select class="form-input" id="filter-status">
            <option value="">All statuses</option>
            <option value="active" ${listParams.status === 'active' ? 'selected' : ''}>Active</option>
            <option value="disabled" ${listParams.status === 'disabled' ? 'selected' : ''}>Disabled</option>
          </select>
        </div>
        <div class="toolbar-actions">
          <button class="btn btn-outline" id="btn-ai-analyze">AI: Проанализировать пользователей</button>
          <button class="btn btn-primary" id="btn-create-user">+ New User</button>
        </div>
      </div>
      <div class="card">
        <div class="table-wrapper">
          <table>
            <thead>
              <tr>
                ${tableHeader('id', 'ID')}
                ${tableHeader('name', 'Name')}
                ${tableHeader('email', 'Email')}
                ${tableHeader('status', 'Status')}
                ${tableHeader('created_at', 'Created')}
                <th style="text-align:right">Actions</th>
              </tr>
            </thead>
            <tbody id="user-tbody">
              <tr><td colspan="6" style="text-align:center;padding:40px;color:var(--text-muted)">Loading…</td></tr>
            </tbody>
          </table>
        </div>
        <div class="pagination" id="pagination"></div>
      </div>
    `;

    document.getElementById('btn-create-user').onclick = () => navigateTo('create');
    document.getElementById('btn-ai-analyze').onclick = () => navigateTo('analyze');

    document.getElementById('filter-search').addEventListener('input', (e) => {
      listParams.search = e.target.value;
      listParams.page = 1;
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(loadUsers, 300);
    });

    document.getElementById('filter-status').addEventListener('change', (e) => {
      listParams.status = e.target.value;
      listParams.page = 1;
      loadUsers();
    });

    document.querySelectorAll('th[data-sort]').forEach(th => {
      th.onclick = () => {
        const field = th.dataset.sort;
        if (listParams.sort_by === field) {
          listParams.order = listParams.order === 'asc' ? 'desc' : 'asc';
        } else {
          listParams.sort_by = field;
          listParams.order = 'asc';
        }
        listParams.page = 1;
        loadUsers();
      };
    });

    loadUsers();
  }

  function tableHeader(field, label) {
    const active = listParams.sort_by === field;
    const arrow = active ? (listParams.order === 'asc' ? '▲' : '▼') : '▲';
    return `<th data-sort="${field}" class="${active ? 'sorted' : ''}">${label}<span class="sort-icon">${arrow}</span></th>`;
  }

  async function loadUsers() {
    const tbody = document.getElementById('user-tbody');
    if (!tbody) return;

    const qs = new URLSearchParams({
      page: listParams.page,
      per_page: listParams.per_page,
      sort_by: listParams.sort_by,
      order: listParams.order,
    });
    if (listParams.search) qs.set('search', listParams.search);
    if (listParams.status) qs.set('status', listParams.status);

    try {
      const data = await API.get(`/api/users?${qs}`);
      const users = data.users || [];

      if (users.length === 0) {
        tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;padding:40px;color:var(--text-muted)">No users found</td></tr>`;
      } else {
        tbody.innerHTML = users.map(u => `
          <tr>
            <td>${u.id}</td>
            <td>${esc(u.name)}</td>
            <td>${esc(u.email)}</td>
            <td><span class="badge badge-${u.status}">${u.status}</span></td>
            <td>${formatDate(u.created_at)}</td>
            <td class="actions">
              <button class="btn btn-ghost btn-sm" data-action="view" data-id="${u.id}">View</button>
              <button class="btn btn-outline btn-sm" data-action="edit" data-id="${u.id}">Edit</button>
              <button class="btn btn-danger btn-sm" data-action="delete" data-id="${u.id}">Delete</button>
            </td>
          </tr>
        `).join('');
      }

      tbody.querySelectorAll('[data-action]').forEach(btn => {
        btn.onclick = () => handleRowAction(btn.dataset.action, parseInt(btn.dataset.id));
      });

      renderPagination(data);

      document.querySelectorAll('th[data-sort]').forEach(th => {
        const field = th.dataset.sort;
        const active = listParams.sort_by === field;
        th.className = active ? 'sorted' : '';
        const icon = th.querySelector('.sort-icon');
        if (icon) icon.textContent = active ? (listParams.order === 'asc' ? '▲' : '▼') : '▲';
        th.onclick = () => {
          if (listParams.sort_by === field) {
            listParams.order = listParams.order === 'asc' ? 'desc' : 'asc';
          } else {
            listParams.sort_by = field;
            listParams.order = 'asc';
          }
          listParams.page = 1;
          loadUsers();
        };
      });
    } catch (err) {
      tbody.innerHTML = `<tr><td colspan="6" style="text-align:center;padding:40px;color:var(--danger)">${esc(err.message)}</td></tr>`;
    }
  }

  function renderPagination(data) {
    const el = document.getElementById('pagination');
    if (!el) return;
    if (data.total_pages <= 1) {
      el.innerHTML = `<span>Showing ${data.total} user${data.total !== 1 ? 's' : ''}</span><span></span>`;
      return;
    }

    const from = (data.page - 1) * data.per_page + 1;
    const to = Math.min(data.page * data.per_page, data.total);

    let btns = '';
    const maxVisible = 7;
    let start = Math.max(1, data.page - Math.floor(maxVisible / 2));
    let end = Math.min(data.total_pages, start + maxVisible - 1);
    if (end - start < maxVisible - 1) start = Math.max(1, end - maxVisible + 1);

    btns += `<button class="page-btn" ${data.page <= 1 ? 'disabled' : ''} data-page="${data.page - 1}">‹</button>`;
    for (let i = start; i <= end; i++) {
      btns += `<button class="page-btn ${i === data.page ? 'active' : ''}" data-page="${i}">${i}</button>`;
    }
    btns += `<button class="page-btn" ${data.page >= data.total_pages ? 'disabled' : ''} data-page="${data.page + 1}">›</button>`;

    el.innerHTML = `<span>Showing ${from}–${to} of ${data.total}</span><div class="pagination-btns">${btns}</div>`;

    el.querySelectorAll('.page-btn').forEach(btn => {
      btn.onclick = () => {
        if (btn.disabled) return;
        listParams.page = parseInt(btn.dataset.page);
        loadUsers();
      };
    });
  }

  async function handleRowAction(action, id) {
    if (action === 'view') {
      navigateTo('view', id);
    } else if (action === 'edit') {
      navigateTo('edit', id);
    } else if (action === 'delete') {
      showConfirm(`Are you sure you want to delete user #${id}? This action cannot be undone.`, async () => {
        try {
          await API.delete(`/api/users/${id}`);
          toast('User deleted');
          loadUsers();
        } catch (err) {
          toast(err.message, 'error');
        }
      });
    }
  }

  async function renderView(container, userId) {
    container.innerHTML = '<div class="detail-card"><p style="color:var(--text-muted)">Loading…</p></div>';
    try {
      const u = await API.get(`/api/users/${userId}`);
      container.innerHTML = `
        <div class="detail-card">
          <a class="back-link" id="back-to-list">← Back to list</a>
          <h2>${esc(u.name)}</h2>
          <div class="detail-row"><div class="detail-label">ID</div><div class="detail-value">${u.id}</div></div>
          <div class="detail-row"><div class="detail-label">Email</div><div class="detail-value">${esc(u.email)}</div></div>
          <div class="detail-row"><div class="detail-label">Status</div><div class="detail-value"><span class="badge badge-${u.status}">${u.status}</span></div></div>
          <div class="detail-row"><div class="detail-label">Created</div><div class="detail-value">${formatDate(u.created_at)}</div></div>
          <div class="detail-row"><div class="detail-label">Updated</div><div class="detail-value">${formatDate(u.updated_at)}</div></div>
          <div class="detail-actions">
            <button class="btn btn-primary" id="detail-edit">Edit</button>
            <button class="btn btn-danger" id="detail-delete">Delete</button>
          </div>
        </div>
      `;
      document.getElementById('back-to-list').onclick = () => navigateTo('list');
      document.getElementById('detail-edit').onclick = () => navigateTo('edit', userId);
      document.getElementById('detail-delete').onclick = () => {
        showConfirm(`Delete user "${esc(u.name)}"?`, async () => {
          try {
            await API.delete(`/api/users/${userId}`);
            toast('User deleted');
            navigateTo('list');
          } catch (err) { toast(err.message, 'error'); }
        });
      };
    } catch (err) {
      container.innerHTML = `<div class="detail-card"><p style="color:var(--danger)">${esc(err.message)}</p><a class="back-link" id="back-to-list">← Back</a></div>`;
      document.getElementById('back-to-list').onclick = () => navigateTo('list');
    }
  }

  async function renderForm(container, userId) {
    const isEdit = userId !== null && userId !== undefined;
    let user = { name: '', email: '', status: 'active' };

    if (isEdit) {
      container.innerHTML = '<div class="detail-card"><p style="color:var(--text-muted)">Loading…</p></div>';
      try { user = await API.get(`/api/users/${userId}`); }
      catch (err) {
        container.innerHTML = `<div class="detail-card"><p style="color:var(--danger)">${esc(err.message)}</p></div>`;
        return;
      }
    }

    container.innerHTML = `
      <div class="detail-card" style="max-width:480px">
        <a class="back-link" id="back-to-list">← Back to list</a>
        <h2>${isEdit ? 'Edit User' : 'New User'}</h2>
        <form id="user-form" novalidate>
          <div class="form-group">
            <label for="f-name">Name</label>
            <input type="text" class="form-input" id="f-name" value="${esc(user.name)}" autocomplete="off">
            <div class="form-error" id="err-name"></div>
          </div>
          <div class="form-group">
            <label for="f-email">Email</label>
            <input type="email" class="form-input" id="f-email" value="${esc(user.email)}" autocomplete="off">
            <div class="form-error" id="err-email"></div>
          </div>
          <div class="form-group">
            <label for="f-password">Password${isEdit ? ' (leave blank to keep)' : ''}</label>
            <input type="password" class="form-input" id="f-password" autocomplete="new-password">
            <div class="form-error" id="err-password"></div>
          </div>
          <div class="form-group">
            <label for="f-status">Status</label>
            <select class="form-input" id="f-status">
              <option value="active" ${user.status === 'active' ? 'selected' : ''}>Active</option>
              <option value="disabled" ${user.status === 'disabled' ? 'selected' : ''}>Disabled</option>
            </select>
            <div class="form-error" id="err-status"></div>
          </div>
          <div class="detail-actions">
            <button type="submit" class="btn btn-primary" id="form-submit">${isEdit ? 'Save Changes' : 'Create User'}</button>
            <button type="button" class="btn btn-outline" id="form-cancel">Cancel</button>
          </div>
        </form>
      </div>
    `;

    document.getElementById('back-to-list').onclick = () => navigateTo('list');
    document.getElementById('form-cancel').onclick = () => navigateTo('list');

    document.getElementById('user-form').addEventListener('submit', async (e) => {
      e.preventDefault();
      clearFormErrors();

      const payload = {
        name: document.getElementById('f-name').value.trim(),
        email: document.getElementById('f-email').value.trim(),
        status: document.getElementById('f-status').value,
      };
      const password = document.getElementById('f-password').value;
      if (password || !isEdit) payload.password = password;

      const btn = document.getElementById('form-submit');
      btn.disabled = true;

      try {
        if (isEdit) {
          await API.put(`/api/users/${userId}`, payload);
          toast('User updated');
        } else {
          await API.post('/api/users', payload);
          toast('User created');
        }
        navigateTo('list');
      } catch (err) {
        if (err.details) {
          Object.entries(err.details).forEach(([field, msg]) => {
            const errEl = document.getElementById(`err-${field}`);
            const input = document.getElementById(`f-${field}`);
            if (errEl) errEl.textContent = msg;
            if (input) input.classList.add('error');
          });
        } else {
          toast(err.message, 'error');
        }
      } finally {
        btn.disabled = false;
      }
    });
  }

  function clearFormErrors() {
    document.querySelectorAll('.form-error').forEach(el => el.textContent = '');
    document.querySelectorAll('.form-input.error').forEach(el => el.classList.remove('error'));
  }

  function showConfirm(text, onConfirm) {
    const overlay = document.getElementById('modal-overlay');
    document.getElementById('confirm-text').textContent = text;
    overlay.classList.add('active');

    const btnOk = document.getElementById('confirm-ok');
    const btnCancel = document.getElementById('confirm-cancel');
    const closeBtn = document.getElementById('modal-close');

    function close() {
      overlay.classList.remove('active');
      btnOk.onclick = null;
      btnCancel.onclick = null;
      closeBtn.onclick = null;
    }

    btnCancel.onclick = close;
    closeBtn.onclick = close;
    btnOk.onclick = async () => {
      close();
      await onConfirm();
    };
  }

  function esc(str) {
    const d = document.createElement('div');
    d.textContent = str || '';
    return d.innerHTML;
  }

  function formatDate(iso) {
    if (!iso) return '—';
    const d = new Date(iso);
    return d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' }) +
      ' ' + d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
  }

  async function init() {
    if (API.isLoggedIn()) {
      try {
        currentUser = await API.get('/api/auth/me');
        headerUser.textContent = currentUser.name;
        showApp();
        navigateTo('list');
      } catch {
        API.clearTokens();
        showLogin();
      }
    } else {
      showLogin();
    }
  }

  init();
});
