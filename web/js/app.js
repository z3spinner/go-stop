// web/js/app.js
'use strict';

const app = document.getElementById('app');

// ── i18n ─────────────────────────────────────────────────────────────────────

const STRINGS = {
  en: {
    tagline:        'Local rides, direct contact',
    btnDriver:      "I'm driving",
    btnSearcher:    'I need a ride',
    postRideTitle:  'Post a ride',
    postReqTitle:   'Post a waiting request',
    findTitle:      'Find a ride',
    labelName:      'Your name',
    labelPhone:     'Phone number',
    labelFrom:      'From',
    labelTo:        'To',
    labelDatetime:  'Date & departure time',
    labelFlex:      'Flexibility',
    flexExact:      'Exact',
    flex30:         '±30 minutes',
    flex60:         '±60 minutes',
    btnPostRide:    'Post ride',
    btnPostReq:     'Post request',
    btnSearch:      'Search',
    btnBack:        '← Back',
    noRides:        'No rides found.',
    btnWaitingReq:  'Post a waiting request',
    flexLabel:      { 0: 'Exact', 30: '±30 min', 60: '±60 min' },
    at:             'at',
    locale:         'en-GB',
  },
  fr: {
    tagline:        'Trajets locaux, contact direct',
    btnDriver:      'Je conduis',
    btnSearcher:    'Je cherche un stop',
    postRideTitle:  'Proposer un trajet',
    postReqTitle:   'Publier une demande',
    findTitle:      'Trouver un stop',
    labelName:      'Votre prénom',
    labelPhone:     'Numéro de téléphone',
    labelFrom:      'Départ',
    labelTo:        'Arrivée',
    labelDatetime:  'Date et heure de départ',
    labelFlex:      'Flexibilité',
    flexExact:      'Exact',
    flex30:         '±30 minutes',
    flex60:         '±60 minutes',
    btnPostRide:    'Publier le trajet',
    btnPostReq:     'Publier la demande',
    btnSearch:      'Rechercher',
    btnBack:        '← Retour',
    noRides:        'Aucun trajet trouvé.',
    btnWaitingReq:  'Publier une demande',
    flexLabel:      { 0: 'Exact', 30: '±30 min', 60: '±60 min' },
    at:             'à',
    locale:         'fr-FR',
  },
};

function detectLang() {
  const stored = localStorage.getItem('lang');
  if (stored === 'fr' || stored === 'en') return stored;
  return (navigator.language || '').startsWith('fr') ? 'fr' : 'en';
}

let lang = detectLang();
const t = () => STRINGS[lang];

function toggleLang() {
  lang = lang === 'en' ? 'fr' : 'en';
  localStorage.setItem('lang', lang);
  renderHome();
}

function langToggle() {
  const label = lang === 'en' ? '🇫🇷 FR' : '🇬🇧 EN';
  return `<button class="btn-lang" id="btn-lang">${label}</button>`;
}

// ── Helpers ───────────────────────────────────────────────────────────────────

function esc(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

function formatTime(iso) {
  const d = new Date(iso);
  const s = t();
  return d.toLocaleDateString(s.locale, { weekday: 'short', day: 'numeric', month: 'short' })
    + ' ' + s.at + ' ' + d.toLocaleTimeString(s.locale, { hour: '2-digit', minute: '2-digit' });
}

async function api(method, path, body) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch('/api' + path, opts);
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || res.statusText);
  return data;
}

async function getDestinations() {
  try { return await api('GET', '/destinations'); }
  catch { return []; }
}

function destinationList(id, destinations) {
  return `<datalist id="${id}">${destinations.map(d => `<option value="${esc(d)}">`).join('')}</datalist>`;
}

function bindLangToggle() {
  const btn = document.getElementById('btn-lang');
  if (btn) btn.onclick = toggleLang;
}

// ── Views ─────────────────────────────────────────────────────────────────────

function renderHome() {
  const s = t();
  app.innerHTML = `
    <div class="hero">
      ${langToggle()}
      <h1>Go-Stop</h1>
      <p class="tagline">${s.tagline}</p>
      <button class="btn btn-primary" id="btn-driver">${s.btnDriver}</button>
      <button class="btn btn-secondary" id="btn-searcher">${s.btnSearcher}</button>
    </div>`;
  document.getElementById('btn-driver').onclick = renderPostRide;
  document.getElementById('btn-searcher').onclick = renderSearchRides;
  bindLangToggle();
}

async function renderPostRide() {
  const s = t();
  const dests = await getDestinations();
  app.innerHTML = `
    <button class="btn-back" id="back">${s.btnBack}</button>
    ${langToggle()}
    <h2>${s.postRideTitle}</h2>
    <form id="ride-form">
      <div class="form-group"><label>${s.labelName}</label><input name="driver_name" required></div>
      <div class="form-group"><label>${s.labelPhone}</label><input name="phone" type="tel" required></div>
      <div class="form-group"><label>${s.labelFrom}</label><input name="origin" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>${s.labelTo}</label><input name="destination" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <div class="form-group"><label>${s.labelDatetime}</label><input name="departure_at" type="datetime-local" required></div>
      <div class="form-group">
        <label>${s.labelFlex}</label>
        <select name="flexibility">
          <option value="0">${s.flexExact}</option>
          <option value="30" selected>${s.flex30}</option>
          <option value="60">${s.flex60}</option>
        </select>
      </div>
      <button class="btn btn-primary" type="submit">${s.btnPostRide}</button>
      <div class="error" id="err"></div>
    </form>`;
  document.getElementById('back').onclick = renderHome;
  bindLangToggle();
  document.getElementById('ride-form').onsubmit = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    try {
      await api('POST', '/rides', {
        driver_name: fd.get('driver_name'),
        phone: fd.get('phone'),
        origin: fd.get('origin'),
        destination: fd.get('destination'),
        departure_at: new Date(fd.get('departure_at')).toISOString(),
        flexibility: parseInt(fd.get('flexibility')),
      });
      renderHome();
    } catch (err) {
      document.getElementById('err').textContent = err.message;
    }
  };
}

async function renderSearchRides() {
  const s = t();
  const dests = await getDestinations();
  app.innerHTML = `
    <button class="btn-back" id="back">${s.btnBack}</button>
    ${langToggle()}
    <h2>${s.findTitle}</h2>
    <form id="search-form">
      <div class="form-group"><label>${s.labelFrom}</label><input name="origin" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>${s.labelTo}</label><input name="destination" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <button class="btn btn-primary" type="submit">${s.btnSearch}</button>
    </form>
    <div id="results"></div>`;
  document.getElementById('back').onclick = renderHome;
  bindLangToggle();
  document.getElementById('search-form').onsubmit = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const origin = fd.get('origin');
    const dest = fd.get('destination');
    const results = document.getElementById('results');
    try {
      const rides = await api('GET', `/rides?origin=${encodeURIComponent(origin)}&destination=${encodeURIComponent(dest)}`);
      if (!rides.length) {
        results.innerHTML = `
          <div class="empty"><p>${s.noRides}</p>
          <button class="btn btn-secondary" id="btn-post-req">${s.btnWaitingReq}</button></div>`;
        document.getElementById('btn-post-req').onclick = () => renderPostRequest(origin, dest);
        return;
      }
      results.innerHTML = rides.map(r => `
        <div class="card">
          <div class="card-route">${esc(r.Origin)} → ${esc(r.Destination)}</div>
          <div class="card-meta">${formatTime(r.DepartureAt)} <span class="tag">${s.flexLabel[r.Flexibility] || esc(r.Flexibility) + ' min'}</span></div>
          <div class="card-contact">
            <strong>${esc(r.DriverName)}</strong> — <a href="tel:${esc(r.Phone)}">${esc(r.Phone)}</a>
          </div>
        </div>`).join('');
    } catch (err) {
      const div = document.createElement('div');
      div.className = 'error';
      div.textContent = err.message;
      results.replaceChildren(div);
    }
  };
}

async function renderPostRequest(origin = '', destination = '') {
  const s = t();
  const dests = await getDestinations();
  app.innerHTML = `
    <button class="btn-back" id="back">${s.btnBack}</button>
    ${langToggle()}
    <h2>${s.postReqTitle}</h2>
    <form id="req-form">
      <div class="form-group"><label>${s.labelName}</label><input name="searcher_name" required></div>
      <div class="form-group"><label>${s.labelPhone}</label><input name="phone" type="tel" required></div>
      <div class="form-group"><label>${s.labelFrom}</label><input name="origin" value="${esc(origin)}" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>${s.labelTo}</label><input name="destination" value="${esc(destination)}" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <div class="form-group"><label>${s.labelDatetime}</label><input name="departure_at" type="datetime-local" required></div>
      <div class="form-group">
        <label>${s.labelFlex}</label>
        <select name="flexibility">
          <option value="0">${s.flexExact}</option>
          <option value="30" selected>${s.flex30}</option>
          <option value="60">${s.flex60}</option>
        </select>
      </div>
      <button class="btn btn-primary" type="submit">${s.btnPostReq}</button>
      <div class="error" id="err"></div>
    </form>`;
  document.getElementById('back').onclick = renderHome;
  bindLangToggle();
  document.getElementById('req-form').onsubmit = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    try {
      await api('POST', '/requests', {
        searcher_name: fd.get('searcher_name'),
        phone: fd.get('phone'),
        origin: fd.get('origin'),
        destination: fd.get('destination'),
        departure_at: new Date(fd.get('departure_at')).toISOString(),
        flexibility: parseInt(fd.get('flexibility')),
      });
      renderHome();
    } catch (err) {
      document.getElementById('err').textContent = err.message;
    }
  };
}

renderHome();

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js').catch(console.error);
}
