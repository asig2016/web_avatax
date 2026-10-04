// avatax.eu – small progressive enhancements. The site works without it,
// except the forms, which need the anti-spam token fetched here.

// ---------- mobile navigation ----------
const header = document.querySelector('.site-header');
const toggle = document.querySelector('.nav-toggle');
if (header && toggle) {
  toggle.addEventListener('click', () => {
    const open = header.classList.toggle('nav-open');
    toggle.setAttribute('aria-expanded', String(open));
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && header.classList.contains('nav-open')) {
      header.classList.remove('nav-open');
      toggle.setAttribute('aria-expanded', 'false');
      toggle.focus();
    }
  });
}

// ---------- forms (contact + CV) ----------
const i18nEl = document.getElementById('form-i18n');
const msg = i18nEl ? JSON.parse(i18nEl.textContent) : {};
const EMAIL_RE = /^[^\s@<>,;"]+@[^\s@<>,;"]+\.[^\s@<>,;"]+$/;

async function fetchToken(form) {
  try {
    const res = await fetch('/api/token', { headers: { Accept: 'application/json' }, cache: 'no-store' });
    if (res.ok) form.elements.token.value = (await res.json()).token;
  } catch (_) { /* reported on submit */ }
}

function showStatus(form, kind, text) {
  const box = form.querySelector('.form-status');
  box.className = 'form-status ' + kind;
  box.textContent = text;
  box.hidden = false;
  box.scrollIntoView({ behavior: 'smooth', block: 'center' });
}

function markInvalid(form, names) {
  form.querySelectorAll('.field-invalid').forEach((f) => f.classList.remove('field-invalid'));
  let first = null;
  for (const name of names) {
    const el = form.elements[name];
    const field = el && (el.closest ? el.closest('.field') : null);
    if (field) {
      field.classList.add('field-invalid');
      el.setAttribute('aria-invalid', 'true');
      first = first || el;
    }
  }
  return first;
}

function clientValidate(form) {
  const bad = [];
  for (const el of form.elements) {
    if (!el.name || el.type === 'hidden') continue;
    el.removeAttribute('aria-invalid');
    const v = el.type === 'checkbox' ? el.checked : el.type === 'file' ? el.files.length : el.value.trim();
    if (el.required && !v) bad.push(el.name);
    else if (el.type === 'email' && v && !EMAIL_RE.test(el.value.trim())) bad.push(el.name);
    else if (el.type === 'file' && el.files.length && el.files[0].size > Number(el.dataset.maxBytes || Infinity)) {
      return { names: [el.name], code: 'too_large' };
    }
  }
  return bad.length ? { names: bad, code: 'invalid' } : null;
}

document.querySelectorAll('form.js-form').forEach((form) => {
  fetchToken(form);
  const button = form.querySelector('button[type=submit]');
  const label = button.textContent;

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const problem = clientValidate(form);
    if (problem) {
      const first = markInvalid(form, problem.names);
      showStatus(form, 'error', msg[problem.code] || msg.invalid);
      if (first) first.focus();
      return;
    }
    if (!form.elements.token.value) await fetchToken(form);

    button.disabled = true;
    button.textContent = msg.sending || label;
    try {
      const res = await fetch(form.action, { method: 'POST', body: new FormData(form), headers: { Accept: 'application/json' } });
      const data = await res.json().catch(() => ({}));
      if (res.ok && data.ok) {
        form.classList.add('sent');
        showStatus(form, 'ok', msg.ok);
        return;
      }
      const first = markInvalid(form, data.fields || []);
      showStatus(form, 'error', msg[data.error] || msg.server);
      if (first) first.focus();
      if (data.error === 'spam') { form.elements.token.value = ''; fetchToken(form); }
    } catch (_) {
      showStatus(form, 'error', msg.network);
    } finally {
      button.disabled = false;
      button.textContent = label;
    }
  });
});

// ---------- cookie consent + Google Analytics 4 (only if configured) ----------
const gaMeta = document.querySelector('meta[name="ga4-id"]');
const banner = document.querySelector('.cookie-banner');
const KEY = 'cookie-consent';

function storage(action, value) {
  try {
    if (action === 'get') return localStorage.getItem(KEY);
    if (action === 'set') localStorage.setItem(KEY, value);
  } catch (_) { return null; }
  return null;
}

function loadGA(id) {
  if (window.gaLoaded) return;
  window.gaLoaded = true;
  const s = document.createElement('script');
  s.async = true;
  s.src = 'https://www.googletagmanager.com/gtag/js?id=' + encodeURIComponent(id);
  document.head.appendChild(s);
  window.dataLayer = window.dataLayer || [];
  window.gtag = function () { window.dataLayer.push(arguments); };
  window.gtag('js', new Date());
  window.gtag('config', id, { anonymize_ip: true });
}

function removeGACookies() {
  document.cookie.split(';').map((c) => c.split('=')[0].trim()).filter((n) => n.startsWith('_ga')).forEach((n) => {
    const host = location.hostname.replace(/^www\./, '');
    for (const domain of ['', '; domain=' + host, '; domain=.' + host]) {
      document.cookie = n + '=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/' + domain;
    }
  });
}

if (gaMeta && banner) {
  const id = gaMeta.content;
  const choice = storage('get');
  if (choice === 'granted') loadGA(id);
  else if (choice !== 'denied') banner.hidden = false;

  banner.addEventListener('click', (e) => {
    const btn = e.target.closest('[data-consent]');
    if (!btn) return;
    const value = btn.dataset.consent;
    const before = storage('get');
    storage('set', value);
    banner.hidden = true;
    if (value === 'granted') loadGA(id);
    else if (before === 'granted') { removeGACookies(); location.reload(); }
  });
  document.querySelectorAll('[data-cookie-settings]').forEach((b) =>
    b.addEventListener('click', () => { banner.hidden = false; banner.querySelector('button').focus(); }));
}

// ---------- live chat (Rocket.Chat Livechat, only if configured) ----------
// The button is shown only while an agent is available. Availability comes from our own
// /api/chat-status (formsvc asks the chat server, cached), so the visitor's browser contacts the
// chat server only after the visitor clicks "Start chat".
const chat = document.querySelector('.chat[data-chat-url]');
const CHAT_KEY = 'chat-started';

function chatSession(action) {
  try {
    if (action === 'get') return sessionStorage.getItem(CHAT_KEY) === '1';
    sessionStorage.setItem(CHAT_KEY, '1');
  } catch (_) { /* storage blocked: the visitor clicks again on the next page */ }
  return false;
}

async function chatStatus() {
  try {
    const res = await fetch('/api/chat-status', { headers: { Accept: 'application/json' }, cache: 'no-store' });
    return res.ok ? await res.json() : { online: false };
  } catch (_) { return { online: false }; }
}

function loadChat(base, lang, department, open) {
  if (window.RocketChat) return;
  window.RocketChat = function (c) { window.RocketChat._.push(c); };
  window.RocketChat._ = [];
  window.RocketChat.url = base + '/livechat';
  const fields = [['website', location.host, true], ['language', lang, true]];
  window.RocketChat(function () {
    const api = this;
    // Custom fields have scope "Room" in Rocket.Chat: they can only be stored once the chat exists.
    // The widget creates it with the first message and then reports chat-started, assign-agent or
    // queue-position-change (depending on the flow); the fields are sent on each of these.
    // the loader API only has setCustomField (singular); setCustomFields exists only as an initialize() option
    const sendFields = () => fields.forEach(([key, value, overwrite]) => api.setCustomField(key, value, overwrite));
    api.setLanguage(lang);
    if (department) api.setDepartment(department);
    api.setTheme({ title: 'AVATAX A.E.', color: '#6c5020', position: 'right' });
    api.onChatStarted(sendFields);
    api.onAssignAgent(sendFields);
    api.onQueuePositionChange(sendFields);
  });
  // Commands that reach the widget before its app is ready are lost (custom fields, maximize), so
  // these are sent shortly after its "ready" message: fields for a chat that is already running
  // (page change), maximize when the visitor has just clicked "Start chat".
  const onReady = (e) => {
    if (e.origin !== base || !e.data || e.data.src !== 'rocketchat' || e.data.fn !== 'ready') return;
    window.removeEventListener('message', onReady);
    setTimeout(() => window.RocketChat(function () {
      fields.forEach(([key, value, overwrite]) => this.setCustomField(key, value, overwrite));
      if (open) this.maximizeWidget();
    }), 1000);
  };
  window.addEventListener('message', onReady);
  const s = document.createElement('script');
  s.async = true;
  s.src = base + '/livechat/rocketchat-livechat.min.js';
  document.body.appendChild(s);
}

if (chat) {
  const base = chat.dataset.chatUrl;
  const lang = chat.dataset.chatLang;
  const launch = chat.querySelector('.chat-launch');
  const panel = chat.querySelector('.chat-panel');
  const start = chat.querySelector('.chat-start');
  let department = '';
  let poll = null;

  const closePanel = () => { panel.hidden = true; launch.setAttribute('aria-expanded', 'false'); };
  const begin = (open) => {
    clearInterval(poll);
    chatSession('set');
    chat.hidden = true;
    loadChat(base, lang, department, open);
  };
  const refresh = async () => {
    const status = await chatStatus();
    department = status.department || '';
    if (!status.online) { chat.hidden = true; closePanel(); return; }
    // chat already started in this browser session: restore the widget (minimised) on every page
    if (chatSession('get')) begin(false);
    else chat.hidden = false;
  };

  refresh();
  // agents come and go: re-check every minute until the visitor starts a chat
  poll = setInterval(refresh, 60000);
  launch.addEventListener('click', () => {
    const show = panel.hidden;
    panel.hidden = !show;
    launch.setAttribute('aria-expanded', String(show));
    if (show) start.focus();
  });
  start.addEventListener('click', () => begin(true));
  chat.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && !panel.hidden) { closePanel(); launch.focus(); }
  });
}
