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
// The buttons are shown only while an agent is available. Availability comes from our own
// /api/chat-status (formsvc asks the chat server, cached), so the visitor's browser contacts the
// chat server only after the visitor clicks "Start chat". Besides the corner button, pages can
// contain inline buttons ([data-chat-open], shortcode {{< chat >}}) that carry a topic.
const chat = document.querySelector('.chat[data-chat-url]');

function chatStore(key, value) {
  try {
    if (value === undefined) return sessionStorage.getItem('chat-' + key) || '';
    sessionStorage.setItem('chat-' + key, value);
  } catch (_) { /* storage blocked: the visitor clicks again on the next page */ }
  return '';
}

async function chatStatus() {
  try {
    const res = await fetch('/api/chat-status', { headers: { Accept: 'application/json' }, cache: 'no-store' });
    return res.ok ? await res.json() : { online: false };
  } catch (_) { return { online: false }; }
}

// Custom fields of the chat (scope "Room" in Rocket.Chat): website, language and, if known, topic.
function chatFields(lang) {
  const fields = [['website', location.host], ['language', lang]];
  const topic = chatStore('topic');
  if (topic) fields.push(['topic', topic]);
  return fields;
}

// the loader API only has setCustomField (singular); setCustomFields exists only as an initialize() option
function sendChatFields(api, lang) {
  chatFields(lang).forEach(([key, value]) => api.setCustomField(key, value, true));
}

function loadChat(base, lang, title, department, guest) {
  if (window.RocketChat) return;
  window.RocketChat = function (c) { window.RocketChat._.push(c); };
  window.RocketChat._ = [];
  window.RocketChat.url = base + '/livechat';
  window.RocketChat(function () {
    const api = this;
    // Room fields can only be stored once the chat exists. The widget creates it with the first
    // message and then reports chat-started, assign-agent or queue-position-change (depending on
    // the flow); the fields are sent on each of these.
    const send = () => sendChatFields(api, lang);
    // The Rocket.Chat widget has no Greek translation; without a supported language it keeps whatever
    // language it used last, so Greek pages get the English widget texts.
    api.setLanguage(lang === 'el' ? 'en' : lang);
    if (department) api.setDepartment(department);
    api.setTheme({ title, color: '#6c5020', position: 'right' });
    api.onChatStarted(send);
    api.onAssignAgent(send);
    api.onQueuePositionChange(send);
  });
  // Commands that reach the widget before its app is ready are lost (custom fields, guest data, maximize),
  // so these are sent shortly after its "ready" message: fields for a chat that is already running (page
  // change); for a new chat the visitor's name and e-mail from our form (the widget then skips its own form)
  // and maximize.
  const onReady = (e) => {
    if (e.origin !== base || !e.data || e.data.src !== 'rocketchat' || e.data.fn !== 'ready') return;
    window.removeEventListener('message', onReady);
    setTimeout(() => window.RocketChat(function () {
      if (guest) this.registerGuest({ token: guest.token, name: guest.name, email: guest.email, department: department || undefined });
      sendChatFields(this, lang);
      if (guest) this.maximizeWidget();
    }), 1000);
  };
  window.addEventListener('message', onReady);
  const s = document.createElement('script');
  s.async = true;
  s.src = base + '/livechat/rocketchat-livechat.min.js';
  document.body.appendChild(s);
}

// Visitor id for Rocket.Chat, kept in this browser so that a returning visitor stays the same contact.
function chatToken() {
  let token = '';
  try { token = localStorage.getItem('chat-token') || ''; } catch (_) { /* storage blocked */ }
  if (!token) {
    token = Array.from(crypto.getRandomValues(new Uint8Array(16)), (b) => b.toString(16).padStart(2, '0')).join('');
    try { localStorage.setItem('chat-token', token); } catch (_) { /* a new id next time */ }
  }
  return token;
}

if (chat) {
  const base = chat.dataset.chatUrl;
  const lang = chat.dataset.chatLang;
  const title = chat.dataset.chatTitle;
  const launch = chat.querySelector('.chat-launch');
  const panel = chat.querySelector('.chat-panel');
  const error = panel.querySelector('.chat-error');
  const inline = document.querySelectorAll('.chat-inline');
  let department = '';
  let topic = chat.dataset.chatTopic || '';
  let poll = null;

  const closePanel = () => { panel.hidden = true; launch.setAttribute('aria-expanded', 'false'); };
  const openPanel = () => { panel.hidden = false; launch.setAttribute('aria-expanded', 'true'); panel.elements.name.focus(); };
  const showInline = (show) => inline.forEach((el) => { el.hidden = !show; });
  const begin = (guest) => {
    clearInterval(poll);
    if (!chatStore('started')) chatStore('topic', topic);
    chatStore('started', '1');
    chat.hidden = true;
    loadChat(base, lang, title, department, guest);
  };
  const refresh = async () => {
    const status = await chatStatus();
    department = status.department || '';
    showInline(status.online);
    if (!status.online) { chat.hidden = true; closePanel(); return; }
    // chat already started in this browser session: restore the widget (minimised) on every page
    if (chatStore('started')) begin(null);
    // pages with chatButton: false (e.g. the contact page) show no chat button; a running chat is still restored
    else chat.hidden = chat.dataset.chatButton === 'off';
  };

  refresh();
  // agents come and go: re-check every minute until the visitor starts a chat
  poll = setInterval(refresh, 60000);
  panel.querySelector('.chat-close').addEventListener('click', () => { closePanel(); launch.focus(); });
  launch.addEventListener('click', () => {
    if (panel.hidden) { topic = chat.dataset.chatTopic || ''; openPanel(); } else closePanel();
  });
  panel.addEventListener('submit', (e) => {
    e.preventDefault();
    const name = panel.elements.name.value.trim();
    const email = panel.elements.email.value.trim();
    const bad = [];
    if (!name) bad.push('name');
    if (!EMAIL_RE.test(email)) bad.push('email');
    ['name', 'email'].forEach((n) => {
      panel.elements[n].closest('.field').classList.toggle('field-invalid', bad.includes(n));
      panel.elements[n].toggleAttribute('aria-invalid', bad.includes(n));
    });
    error.hidden = !bad.length;
    if (bad.length) { panel.elements[bad[0]].focus(); return; }
    begin({ token: chatToken(), name, email });
  });
  chat.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && !panel.hidden) { closePanel(); launch.focus(); }
  });
  document.querySelectorAll('[data-chat-open]').forEach((btn) => btn.addEventListener('click', () => {
    topic = btn.dataset.chatTopic || chat.dataset.chatTopic || '';
    if (window.RocketChat) {
      // chat already running: switch its topic and open the widget
      chatStore('topic', topic);
      window.RocketChat(function () { sendChatFields(this, lang); this.maximizeWidget(); });
      return;
    }
    openPanel();
  }));
}
