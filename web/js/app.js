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
    privacyTitle:    'Privacy',
    privacyClose:    'Close',
    notifTitle:      'Get notified of matches',
    notifBody:       'Allow notifications to be alerted when a matching ride or passenger is found.',
    notifEnable:     'Enable notifications',
    notifSkip:       'No thanks',
    notifDenied:     'Notifications blocked in browser settings.',
    privacyBody:    `<h3>What we collect</h3>
<p>When you post a ride or request we store: your name, phone number, origin, destination, departure time, and flexibility window. Nothing else.</p>
<h3>How long we keep it</h3>
<p>Rides and requests are <strong>automatically and permanently deleted</strong> at the end of their departure day. If you want to delete your post sooner, use the delete option — you'll need the phone number you posted with.</p>
<p>Push notification subscriptions are kept until you unsubscribe.</p>
<h3>Who can see your phone number</h3>
<p>Your phone number is visible to anyone who views your ride or request card. This is intentional — it's how the two parties contact each other directly.</p>
<h3>Cookies &amp; local storage</h3>
<p>No cookies. Go-Stop uses no tracking and no analytics.</p>
<p>The following is saved in your browser's <code>localStorage</code> (on your device only, never sent to the server):</p>
<ul>
<li>Your name and phone number — to pre-fill forms on your next visit</li>
<li>Your language preference (English or French)</li>
</ul>
<p>You can clear this at any time by clearing your browser's site data.</p>
<h3>Third parties</h3>
<p>No data is shared with third parties. Push notifications are delivered via the Web Push standard through your browser's push service (e.g. Google FCM for Chrome). The push payload contains only the match details you'd see on screen.</p>`,
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
    privacyTitle:    'Confidentialité',
    privacyClose:    'Fermer',
    notifTitle:      'Recevoir des alertes',
    notifBody:       'Activez les notifications pour être alerté(e) dès qu\'un trajet ou passager correspondant est trouvé.',
    notifEnable:     'Activer les notifications',
    notifSkip:       'Non merci',
    notifDenied:     'Notifications bloquées dans les paramètres du navigateur.',
    privacyBody:    `<h3>Ce que nous collectons</h3>
<p>Lorsque vous publiez un trajet ou une demande, nous enregistrons : votre prénom, numéro de téléphone, lieu de départ, destination, heure de départ et flexibilité. Rien d'autre.</p>
<h3>Durée de conservation</h3>
<p>Les trajets et demandes sont <strong>supprimés automatiquement et définitivement</strong> à la fin du jour de départ. Pour supprimer votre annonce plus tôt, utilisez l'option de suppression — le numéro de téléphone utilisé lors de la publication sera demandé.</p>
<p>Les abonnements aux notifications push sont conservés jusqu'à ce que vous vous désinscriviez.</p>
<h3>Qui peut voir votre numéro de téléphone</h3>
<p>Votre numéro est visible par toute personne qui consulte votre annonce. C'est volontaire — c'est ainsi que les deux parties se contactent directement.</p>
<h3>Cookies &amp; stockage local</h3>
<p>Aucun cookie. Go-Stop n'utilise ni traceurs ni analytiques.</p>
<p>Les informations suivantes sont enregistrées dans le <code>localStorage</code> de votre navigateur (sur votre appareil uniquement, jamais envoyées au serveur) :</p>
<ul>
<li>Votre prénom et numéro de téléphone — pour pré-remplir les formulaires à votre prochaine visite</li>
<li>Votre préférence de langue (français ou anglais)</li>
</ul>
<p>Vous pouvez effacer ces données à tout moment en vidant les données de site de votre navigateur.</p>
<h3>Tiers</h3>
<p>Aucune donnée n'est partagée avec des tiers. Les notifications push transitent par le standard Web Push via le service push de votre navigateur (ex. Google FCM pour Chrome). Le contenu transmis se limite aux informations de mise en relation visibles à l'écran.</p>`,
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

// Shows the CURRENT language flag so users know what language they are in.
// Clicking it switches to the other language.
function langToggle() {
  const label = lang === 'en' ? '🇬🇧 EN' : '🇫🇷 FR';
  return `<button class="btn-lang" id="btn-lang">${label}</button>`;
}

function privacyIcon() {
  return `<button class="btn-privacy" id="btn-privacy" aria-label="${t().privacyTitle}">ⓘ</button>`;
}

function showPrivacyModal() {
  const s = t();
  const overlay = document.createElement('div');
  overlay.className = 'modal-overlay';
  overlay.id = 'privacy-modal';
  overlay.innerHTML = `
    <div class="modal">
      <div class="modal-header">
        <h2>${s.privacyTitle}</h2>
        <button class="btn-modal-close" id="btn-modal-close">${s.privacyClose}</button>
      </div>
      <div class="modal-body">${s.privacyBody}</div>
    </div>`;
  document.body.appendChild(overlay);
  overlay.onclick = (e) => { if (e.target === overlay) overlay.remove(); };
  document.getElementById('btn-modal-close').onclick = () => overlay.remove();
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

function bindControls() {
  const langBtn = document.getElementById('btn-lang');
  if (langBtn) langBtn.onclick = toggleLang;
  const privBtn = document.getElementById('btn-privacy');
  if (privBtn) privBtn.onclick = showPrivacyModal;
}

// ── Push notifications ────────────────────────────────────────────────────────

function urlBase64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/');
  const raw = atob(base64);
  return Uint8Array.from([...raw].map(c => c.charCodeAt(0)));
}

async function trySubscribePush(phone) {
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) return false;
  try {
    const reg = await navigator.serviceWorker.ready;
    const { publicKey } = await api('GET', '/vapid-public-key');
    const sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(publicKey),
    });
    const { endpoint, keys: { p256dh, auth } } = sub.toJSON();
    await api('POST', '/subscriptions', { phone, endpoint, p256dh, auth });
    return true;
  } catch {
    return false;
  }
}

// Shows a "would you like notifications?" prompt after a successful post.
// Calls onDone() when the user has either enabled or skipped.
function renderNotificationPrompt(phone, onDone) {
  const s = t();
  const alreadyGranted = Notification.permission === 'granted';

  if (alreadyGranted) {
    // Already have permission — subscribe silently and move on.
    trySubscribePush(phone).then(onDone);
    return;
  }
  if (Notification.permission === 'denied') {
    onDone();
    return;
  }

  app.innerHTML = `
    <div class="notif-prompt">
      <div class="notif-icon">🔔</div>
      <h2>${s.notifTitle}</h2>
      <p>${s.notifBody}</p>
      <button class="btn btn-primary" id="btn-notif-yes">${s.notifEnable}</button>
      <button class="btn btn-secondary" id="btn-notif-no">${s.notifSkip}</button>
      <div class="error" id="notif-err"></div>
    </div>`;

  document.getElementById('btn-notif-yes').onclick = async () => {
    const permission = await Notification.requestPermission();
    if (permission === 'granted') {
      await trySubscribePush(phone);
      onDone();
    } else {
      document.getElementById('notif-err').textContent = s.notifDenied;
      setTimeout(onDone, 1500);
    }
  };
  document.getElementById('btn-notif-no').onclick = onDone;
}

// ── User profile (localStorage) ───────────────────────────────────────────────

function saveProfile(name, phone) {
  if (name)  localStorage.setItem('user_name', name);
  if (phone) localStorage.setItem('user_phone', phone);
}

function getProfile() {
  return {
    name:  localStorage.getItem('user_name')  || '',
    phone: localStorage.getItem('user_phone') || '',
  };
}

// Returns a datetime-local string 1 hour from now, rounded to nearest 5 min.
function defaultDeparture() {
  const d = new Date(Date.now() + 60 * 60 * 1000);
  d.setMinutes(Math.ceil(d.getMinutes() / 5) * 5, 0, 0);
  return d.toISOString().slice(0, 16);
}

// ── Views ─────────────────────────────────────────────────────────────────────

function topBar() {
  return `<div class="top-bar">${langToggle()}${privacyIcon()}</div>`;
}

function renderHome() {
  const s = t();
  app.innerHTML = `
    ${topBar()}
    <div class="hero">
      <h1>Go-Stop</h1>
      <p class="tagline">${s.tagline}</p>
      <button class="btn btn-primary" id="btn-driver">${s.btnDriver}</button>
      <button class="btn btn-secondary" id="btn-searcher">${s.btnSearcher}</button>
    </div>`;
  document.getElementById('btn-driver').onclick = renderPostRide;
  document.getElementById('btn-searcher').onclick = renderSearchRides;
  bindControls();
}

async function renderPostRide() {
  const s = t();
  const p = getProfile();
  const dests = await getDestinations();
  app.innerHTML = `
    <div class="top-bar"><button class="btn-back" id="back">${s.btnBack}</button>${langToggle()}${privacyIcon()}</div>
    <h2>${s.postRideTitle}</h2>
    <form id="ride-form">
      <div class="form-group"><label>${s.labelName}</label><input name="driver_name" value="${esc(p.name)}" required></div>
      <div class="form-group"><label>${s.labelPhone}</label><input name="phone" type="tel" value="${esc(p.phone)}" required></div>
      <div class="form-group"><label>${s.labelFrom}</label><input name="origin" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>${s.labelTo}</label><input name="destination" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <div class="form-group"><label>${s.labelDatetime}</label><input name="departure_at" type="datetime-local" value="${defaultDeparture()}" required></div>
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
  bindControls();
  document.getElementById('ride-form').onsubmit = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const phone = fd.get('phone');
    saveProfile(fd.get('driver_name'), phone);
    try {
      await api('POST', '/rides', {
        driver_name: fd.get('driver_name'),
        phone,
        origin: fd.get('origin'),
        destination: fd.get('destination'),
        departure_at: new Date(fd.get('departure_at')).toISOString(),
        flexibility: parseInt(fd.get('flexibility')),
      });
      renderNotificationPrompt(phone, renderHome);
    } catch (err) {
      document.getElementById('err').textContent = err.message;
    }
  };
}

async function renderSearchRides() {
  const s = t();
  const dests = await getDestinations();
  app.innerHTML = `
    <div class="top-bar"><button class="btn-back" id="back">${s.btnBack}</button>${langToggle()}${privacyIcon()}</div>
    <h2>${s.findTitle}</h2>
    <form id="search-form">
      <div class="form-group"><label>${s.labelFrom}</label><input name="origin" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>${s.labelTo}</label><input name="destination" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <button class="btn btn-primary" type="submit">${s.btnSearch}</button>
    </form>
    <div id="results"></div>`;
  document.getElementById('back').onclick = renderHome;
  bindControls();
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
  const p = getProfile();
  const dests = await getDestinations();
  app.innerHTML = `
    <div class="top-bar"><button class="btn-back" id="back">${s.btnBack}</button>${langToggle()}${privacyIcon()}</div>
    <h2>${s.postReqTitle}</h2>
    <form id="req-form">
      <div class="form-group"><label>${s.labelName}</label><input name="searcher_name" value="${esc(p.name)}" required></div>
      <div class="form-group"><label>${s.labelPhone}</label><input name="phone" type="tel" value="${esc(p.phone)}" required></div>
      <div class="form-group"><label>${s.labelFrom}</label><input name="origin" value="${esc(origin)}" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>${s.labelTo}</label><input name="destination" value="${esc(destination)}" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <div class="form-group"><label>${s.labelDatetime}</label><input name="departure_at" type="datetime-local" value="${defaultDeparture()}" required></div>
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
  bindControls();
  document.getElementById('req-form').onsubmit = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const phone = fd.get('phone');
    saveProfile(fd.get('searcher_name'), phone);
    try {
      await api('POST', '/requests', {
        searcher_name: fd.get('searcher_name'),
        phone,
        origin: fd.get('origin'),
        destination: fd.get('destination'),
        departure_at: new Date(fd.get('departure_at')).toISOString(),
        flexibility: parseInt(fd.get('flexibility')),
      });
      renderNotificationPrompt(phone, renderHome);
    } catch (err) {
      document.getElementById('err').textContent = err.message;
    }
  };
}

renderHome();

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js').catch(console.error);
}
