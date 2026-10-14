// avatax.eu – appointment booking (pages /book/, /book/confirm/, /book/cancel/ and translations).
// Talks only to our own /api/booking/* (formsvc), never to EGroupware directly.

const T = (() => {
  const el = document.getElementById('booking-i18n');
  return el ? JSON.parse(el.textContent) : {};
})();

async function api(path, options = {}) {
  const res = await fetch(path, { cache: 'no-store', ...options, headers: { Accept: 'application/json', ...(options.headers || {}) } });
  const data = await res.json().catch(() => ({}));
  if (!res.ok || data.ok === false) {
    const err = new Error(data.error || 'server');
    err.code = data.error || (res.status >= 500 ? 'server' : 'invalid');
    err.fields = data.fields || [];
    throw err;
  }
  return data;
}

function showStatus(box, kind, text) {
  box.className = 'form-status ' + kind;
  box.textContent = text;
  box.hidden = false;
  box.scrollIntoView({ behavior: 'smooth', block: 'center' });
}

function errorText(err) {
  if (err instanceof TypeError) return T.network;
  return T[err.code] || T.server;
}

// --- dates: the booking timezone decides what "a day" is -------------------

function todayIn(tz) {
  return new Intl.DateTimeFormat('en-CA', { timeZone: tz, year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date());
}

function addDays(ymd, n) {
  const d = new Date(ymd + 'T12:00:00Z');
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}

function dayLabel(lang, ymd) {
  return new Intl.DateTimeFormat(lang, { timeZone: 'UTC', weekday: 'short', day: 'numeric', month: 'short' }).format(new Date(ymd + 'T12:00:00Z'));
}

function longTime(lang, tz, iso) {
  return new Intl.DateTimeFormat(lang, {
    timeZone: tz, weekday: 'long', day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit', hourCycle: 'h23',
  }).format(new Date(iso));
}

function shortTime(lang, tz, iso) {
  return new Intl.DateTimeFormat(lang, { timeZone: tz, hour: '2-digit', minute: '2-digit', hourCycle: 'h23' }).format(new Date(iso));
}

function money(lang, amount, currency) {
  try {
    return new Intl.NumberFormat(lang, { style: 'currency', currency }).format(amount);
  } catch (_) {
    return amount + ' ' + currency;
  }
}

function el(tag, attrs = {}, ...children) {
  const e = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === 'class') e.className = v;
    else if (k.startsWith('data-')) e.setAttribute(k, v);
    else e[k] = v;
  }
  for (const c of children) if (c !== null && c !== undefined && c !== '') e.append(c);
  return e;
}

// --- the booking page ------------------------------------------------------

const root = document.getElementById('booking');
if (root) bookingPage(root);

function bookingPage(root) {
  const lang = root.dataset.lang;
  const steps = Object.fromEntries([...root.querySelectorAll('.booking-step')].map((s) => [s.dataset.step, s]));
  const form = root.querySelector('.booking-form');
  const status = form.querySelector('.form-status');
  const state = { catalog: null, service: null, person: null, week: null, slot: null };
  root.hidden = false;

  function choose(container, value) {
    container.querySelectorAll('[role=radio]').forEach((b) => b.setAttribute('aria-checked', String(b.dataset.value === String(value))));
  }

  function showFrom(step) {
    // a choice in a step resets everything after it
    const order = ['service', 'person', 'slot', 'details'];
    order.slice(order.indexOf(step) + 1).forEach((s) => { steps[s].hidden = true; });
  }

  function option(value, title, ...lines) {
    return el('button', { type: 'button', class: 'booking-option', role: 'radio', 'data-value': String(value), ariaChecked: 'false' },
      el('strong', {}, title), ...lines.filter(Boolean).map((l) => el('span', {}, l)));
  }

  async function loadCatalog() {
    const box = steps.service.querySelector('.booking-options');
    box.textContent = T.loading;
    try {
      state.catalog = await api('/api/booking/catalog?lang=' + encodeURIComponent(lang));
    } catch (err) {
      box.textContent = '';
      box.append(el('p', { class: 'form-status error' }, errorText(err)));
      return;
    }
    box.textContent = '';
    if (!state.catalog.services.length) {
      box.append(el('p', { class: 'form-note' }, T.unavailable));
      return;
    }
    for (const s of state.catalog.services) {
      const price = s.price === null ? T.free : money(lang, s.price, s.currency);
      const b = option(s.id, s.title, s.durationMinutes + ' ' + T.minutes + ' · ' + price, s.description);
      b.addEventListener('click', () => selectService(s));
      box.append(b);
    }
    if (state.catalog.services.length === 1) selectService(state.catalog.services[0]);
  }

  function selectService(service) {
    state.service = service;
    choose(steps.service, service.id);
    showFrom('service');
    const box = steps.person.querySelector('.booking-options');
    box.textContent = '';
    const persons = state.catalog.persons.filter((p) => service.persons.includes(p.id));
    if (persons.length > 1) {
      const any = option('any', T.anyPerson);
      any.addEventListener('click', () => selectPerson('any'));
      box.append(any);
    }
    for (const p of persons) {
      const b = option(p.id, p.name, p.bio);
      b.addEventListener('click', () => selectPerson(p.id));
      box.append(b);
    }
    steps.person.hidden = false;
    if (persons.length === 1) selectPerson(persons[0].id);
    else steps.person.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }

  function selectPerson(id) {
    state.person = id;
    choose(steps.person, id);
    showFrom('person');
    state.week = todayIn(state.catalog.timeZone);
    steps.slot.hidden = false;
    loadWeek();
  }

  async function loadWeek() {
    const tz = state.catalog.timeZone;
    const week = steps.slot.querySelector('.booking-week');
    const none = steps.slot.querySelector('.booking-none');
    const from = state.week;
    const to = addDays(from, 7);
    steps.slot.querySelector('.booking-week-label').textContent = dayLabel(lang, from) + ' – ' + dayLabel(lang, addDays(from, 6));
    steps.slot.querySelector('[data-week="-1"]').disabled = from <= todayIn(tz);
    week.textContent = T.loading;
    none.hidden = true;
    let data;
    try {
      data = await api('/api/booking/slots?' + new URLSearchParams({ service: state.service.id, person: state.person, from, to }));
    } catch (err) {
      week.textContent = '';
      showStatus(none, 'error', errorText(err));
      return;
    }
    week.textContent = '';
    const byDay = {};
    for (const s of data.slots) (byDay[s.start.slice(0, 10)] ||= []).push(s);
    for (let i = 0; i < 7; i++) {
      const day = addDays(from, i);
      const slots = byDay[day] || [];
      const col = el('div', { class: 'booking-day' + (slots.length ? '' : ' empty') }, el('h3', {}, dayLabel(lang, day)));
      for (const s of slots) {
        const b = el('button', { type: 'button', class: 'booking-slot', 'data-start': s.start }, shortTime(lang, tz, s.start));
        b.addEventListener('click', () => selectSlot(s, b));
        col.append(b);
      }
      week.append(col);
    }
    none.className = 'form-note booking-none';
    none.textContent = T.noSlots;
    none.hidden = data.slots.length > 0;
  }

  steps.slot.querySelectorAll('[data-week]').forEach((b) => b.addEventListener('click', () => {
    state.week = addDays(state.week, 7 * Number(b.dataset.week));
    showFrom('slot');
    loadWeek();
  }));

  function selectSlot(slot, button) {
    state.slot = slot;
    steps.slot.querySelectorAll('.booking-slot').forEach((b) => b.setAttribute('aria-pressed', String(b === button)));
    const tz = state.catalog.timeZone;
    const person = state.person === 'any' ? T.anyPerson : state.catalog.persons.find((p) => p.id === state.person).name;
    steps.details.querySelector('.booking-summary').textContent =
      state.service.title + ' · ' + T.with + ' ' + person + ' · ' + longTime(lang, tz, slot.start);
    form.elements.service.value = state.service.id;
    form.elements.person.value = state.person;
    form.elements.start.value = slot.start;
    form.hidden = false;
    form.classList.remove('sent');
    status.hidden = true;
    steps.details.hidden = false;
    steps.details.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }

  async function fetchToken() {
    try {
      form.elements.token.value = (await api('/api/token')).token;
    } catch (_) { /* reported on submit */ }
  }
  fetchToken();

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const bad = [];
    form.querySelectorAll('.field-invalid').forEach((f) => f.classList.remove('field-invalid'));
    for (const name of ['name', 'email', 'consent']) {
      const input = form.elements[name];
      const v = input.type === 'checkbox' ? input.checked : input.value.trim();
      if (!v || (name === 'email' && !/^[^\s@<>,;"]+@[^\s@<>,;"]+\.[^\s@<>,;"]+$/.test(input.value.trim()))) bad.push(input);
    }
    if (bad.length) {
      bad.forEach((i) => i.closest('.field').classList.add('field-invalid'));
      showStatus(status, 'error', T.invalid);
      bad[0].focus();
      return;
    }
    if (!form.elements.token.value) await fetchToken();
    const button = form.querySelector('button[type=submit]');
    const label = button.textContent;
    button.disabled = true;
    button.textContent = T.sending;
    try {
      const data = await api(form.action, { method: 'POST', body: new FormData(form) });
      form.classList.add('sent');
      const until = data.booking && data.booking.expires ? shortTime(lang, state.catalog.timeZone, data.booking.expires) : '';
      showStatus(status, 'ok', T.held.replace('%s', until));
      steps.service.hidden = steps.person.hidden = steps.slot.hidden = true;
      root.querySelector('.booking-intro').hidden = true;
    } catch (err) {
      if (err.code === 'slot_taken') {
        showStatus(status, 'error', T.slot_taken);
        loadWeek();
        steps.slot.scrollIntoView({ behavior: 'smooth', block: 'start' });
      } else {
        (err.fields || []).forEach((n) => form.elements[n] && form.elements[n].closest('.field') && form.elements[n].closest('.field').classList.add('field-invalid'));
        showStatus(status, 'error', errorText(err));
        if (err.code === 'spam') { form.elements.token.value = ''; fetchToken(); }
      }
    } finally {
      button.disabled = false;
      button.textContent = label;
    }
  });

  loadCatalog();
}

// --- the confirm / cancel page ---------------------------------------------

const tokenRoot = document.getElementById('booking-token');
if (tokenRoot) tokenPage(tokenRoot);

function tokenPage(root) {
  const lang = root.dataset.lang;
  const mode = root.dataset.mode;
  const card = root.querySelector('.booking-card');
  const status = root.querySelector('.form-status');
  const again = root.querySelector('.booking-again');
  const token = new URLSearchParams(location.hash.slice(1)).get('t');
  root.hidden = false;

  if (!token) {
    showStatus(status, 'error', T.noToken);
    return;
  }

  function details(b) {
    const dl = card.querySelector('.booking-details');
    dl.textContent = '';
    const row = (label, value) => { if (value) dl.append(el('dt', {}, label), el('dd', {}, value)); };
    row(T.service, b.service);
    row(T.person, b.person);
    row(T.when, longTime(lang, b.timeZone, b.start));
    row(T.location, b.video ? T.video : b.location);
    row(T.status, T['st_' + b.status] || b.status);
    if (b.cancelUntil && b.status === 'confirmed') row(T.cancelUntil, longTime(lang, b.timeZone, b.cancelUntil));
    card.hidden = false;
  }

  function finished(b) {
    card.querySelector('button').hidden = true;
    const reason = card.querySelector('.field');
    if (reason) reason.hidden = true;
    if (b.status === 'cancelled' || b.status === 'expired') again.hidden = false;
  }

  (async () => {
    let b;
    try {
      b = (await api('/api/booking/booking', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ token }) })).booking;
    } catch (err) {
      showStatus(status, 'error', errorText(err));
      return;
    }
    details(b);
    const actionable = mode === 'confirm' ? b.status === 'held' : b.status === 'confirmed' || b.status === 'held';
    if (!actionable) {
      finished(b);
      if (mode === 'confirm' && b.status === 'confirmed') showStatus(status, 'ok', T.confirmed);
      if (b.status === 'cancelled') showStatus(status, 'ok', T.cancelled);
      if (b.status === 'expired') showStatus(status, 'error', T.expired);
    }
  })();

  const button = card.querySelector('button[data-action]');
  button.addEventListener('click', async () => {
    const label = button.textContent;
    button.disabled = true;
    button.textContent = T.sending;
    try {
      const body = mode === 'confirm'
        ? { token, cancel_page: root.dataset.cancelPage }
        : { token, reason: (root.querySelector('#t-reason') || {}).value || '' };
      const data = await api('/api/booking/' + mode, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
      details(data.booking);
      finished(data.booking);
      showStatus(status, 'ok', mode === 'confirm' ? T.confirmed : T.cancelled);
      // the token has done its job on this page; keep it out of the history
      history.replaceState(null, '', location.pathname);
    } catch (err) {
      showStatus(status, 'error', errorText(err));
      if (['expired', 'cancelled', 'slot_taken'].includes(err.code)) again.hidden = false;
    } finally {
      button.disabled = false;
      button.textContent = label;
    }
  });
}
