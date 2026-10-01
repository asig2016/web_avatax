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
