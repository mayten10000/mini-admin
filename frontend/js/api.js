const API = (() => {
  const TOKEN_KEY = 'access_token';
  const REFRESH_KEY = 'refresh_token';

  function getAccessToken() { return localStorage.getItem(TOKEN_KEY); }
  function getRefreshToken() { return localStorage.getItem(REFRESH_KEY); }

  function setTokens(access, refresh) {
    localStorage.setItem(TOKEN_KEY, access);
    localStorage.setItem(REFRESH_KEY, refresh);
  }

  function clearTokens() {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_KEY);
  }

  function isLoggedIn() {
    return !!getAccessToken();
  }

  async function refreshAccessToken() {
    const rt = getRefreshToken();
    if (!rt) throw new Error('No refresh token');

    const res = await fetch('/api/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: rt }),
    });

    if (!res.ok) {
      clearTokens();
      throw new Error('Refresh failed');
    }

    const data = await res.json();
    setTokens(data.access_token, data.refresh_token);
    return data.access_token;
  }

  async function request(method, url, body = null, retry = true) {
    const headers = { 'Content-Type': 'application/json' };
    const token = getAccessToken();
    if (token) headers['Authorization'] = `Bearer ${token}`;

    const opts = { method, headers };
    if (body) opts.body = JSON.stringify(body);

    let res = await fetch(url, opts);

    if (res.status === 401 && retry) {
      try {
        await refreshAccessToken();
        return request(method, url, body, false);
      } catch {
        clearTokens();
        window.dispatchEvent(new Event('auth:logout'));
        throw new Error('Session expired');
      }
    }

    const data = await res.json();

    if (!res.ok) {
      const err = new Error(data.error || 'Request failed');
      err.status = res.status;
      err.details = data.details || null;
      throw err;
    }

    return data;
  }

  return {
    get:    (url)       => request('GET', url),
    post:   (url, body) => request('POST', url, body),
    put:    (url, body) => request('PUT', url, body),
    delete: (url)       => request('DELETE', url),
    setTokens,
    clearTokens,
    isLoggedIn,
    getRefreshToken,
  };
})();
