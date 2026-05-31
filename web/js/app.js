// web/js/app.js
'use strict';

const app = document.getElementById('app');

const FLEX_LABELS = { 0: 'Exact', 30: '±30 min', 60: '±60 min' };

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
  return d.toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short' })
    + ' at ' + d.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
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

function renderHome() {
  app.innerHTML = `
    <div class="hero">
      <h1>Go-Stop</h1>
      <p class="tagline">Local rides, direct contact</p>
      <button class="btn btn-primary" id="btn-driver">I'm driving</button>
      <button class="btn btn-secondary" id="btn-searcher">I need a ride</button>
    </div>`;
  document.getElementById('btn-driver').onclick = renderPostRide;
  document.getElementById('btn-searcher').onclick = renderSearchRides;
}

async function renderPostRide() {
  const dests = await getDestinations();
  app.innerHTML = `
    <button class="btn-back" id="back">← Back</button>
    <h2>Post a ride</h2>
    <form id="ride-form">
      <div class="form-group"><label>Your name</label><input name="driver_name" required></div>
      <div class="form-group"><label>Phone number</label><input name="phone" type="tel" required></div>
      <div class="form-group"><label>From</label><input name="origin" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>To</label><input name="destination" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <div class="form-group"><label>Date &amp; departure time</label><input name="departure_at" type="datetime-local" required></div>
      <div class="form-group">
        <label>Flexibility</label>
        <select name="flexibility">
          <option value="0">Exact</option>
          <option value="30" selected>±30 minutes</option>
          <option value="60">±60 minutes</option>
        </select>
      </div>
      <button class="btn btn-primary" type="submit">Post ride</button>
      <div class="error" id="err"></div>
    </form>`;
  document.getElementById('back').onclick = renderHome;
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
  const dests = await getDestinations();
  app.innerHTML = `
    <button class="btn-back" id="back">← Back</button>
    <h2>Find a ride</h2>
    <form id="search-form">
      <div class="form-group"><label>From</label><input name="origin" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>To</label><input name="destination" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <button class="btn btn-primary" type="submit">Search</button>
    </form>
    <div id="results"></div>`;
  document.getElementById('back').onclick = renderHome;
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
          <div class="empty"><p>No rides found.</p>
          <button class="btn btn-secondary" id="btn-post-req">Post a waiting request</button></div>`;
        document.getElementById('btn-post-req').onclick = () => renderPostRequest(origin, dest);
        return;
      }
      results.innerHTML = rides.map(r => `
        <div class="card">
          <div class="card-route">${esc(r.Origin)} → ${esc(r.Destination)}</div>
          <div class="card-meta">${formatTime(r.DepartureAt)} <span class="tag">${FLEX_LABELS[r.Flexibility] || esc(r.Flexibility) + ' min'}</span></div>
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
  const dests = await getDestinations();
  app.innerHTML = `
    <button class="btn-back" id="back">← Back</button>
    <h2>Post a waiting request</h2>
    <form id="req-form">
      <div class="form-group"><label>Your name</label><input name="searcher_name" required></div>
      <div class="form-group"><label>Phone number</label><input name="phone" type="tel" required></div>
      <div class="form-group"><label>From</label><input name="origin" value="${esc(origin)}" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>To</label><input name="destination" value="${esc(destination)}" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <div class="form-group"><label>Date &amp; time needed</label><input name="departure_at" type="datetime-local" required></div>
      <div class="form-group">
        <label>Flexibility</label>
        <select name="flexibility">
          <option value="0">Exact</option>
          <option value="30" selected>±30 minutes</option>
          <option value="60">±60 minutes</option>
        </select>
      </div>
      <button class="btn btn-primary" type="submit">Post request</button>
      <div class="error" id="err"></div>
    </form>`;
  document.getElementById('back').onclick = renderHome;
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
