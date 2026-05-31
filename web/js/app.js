// web/js/app.js
'use strict';

const app = document.getElementById('app');
let SITE_NAME = 'Go-Stop';

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
    notifDenied:      'Notifications blocked in browser settings.',
    btnMyRides:       'My rides',
    myRidesTitle:     'My rides',
    labelPhoneCheck:  'Your phone number',
    btnShowRides:     'Show my rides',
    noMyRides:        'No active rides found for this number.',
    btnDelete:        'Delete',
    deleteOk:         'Deleted.',
    deleteErr:        'Could not delete — is that the right phone number?',
    labelSearchTime:   'Around what time? (optional)',
    colOutbound:       'Outbound',
    colReturn:         'Return',
    noRidesCol:        'No rides available.',
    tripTypeLabel:     'Trip type',
    tripOneWay:        'One way',
    tripReturn:        'Return',
    returnSection:     'Return journey',
    labelReturnTime:   'Return departure time',
    labelReturnFlex:   'Return flexibility',
    btnNotifyRoute:   '🔔 Notify me of new rides on this route',
    notifRouteTitle:  'Get notified',
    notifRouteBody:   'We\'ll alert you when a ride matching this route is posted. Enter your details below.',
    notifRouteSet:    '✓ You\'ll be notified when a matching ride appears.',
    btnMyAlerts:      'My alerts',
    myAlertsTitle:    'My alerts',
    btnShowAlerts:    'Show my alerts',
    noMyAlerts:       'No active alerts found for this number.',
    alertCard:        (r) => `${r.Origin} → ${r.Destination}`,
    detailRideTitle:  'Ride available',
    detailReqTitle:   'Ride request',
    labelDriver:      'Driver',
    labelSearcher:    'Passenger',
    labelDeparture:   'Departure',
    labelContact:     'Contact',
    footerPrivacy:    'Privacy',
    aboutTitle:       'About Go Stop',
    aboutBody:        (siteName) => `<p><strong>Go Stop</strong> is a local ride-sharing platform, positioned between hitchhiking and carpooling. It connects drivers offering one-time trips with people looking for a lift. Direct contact by phone — no accounts required.</p>
<h3>Your community</h3>
<p>This instance is deployed for <strong>${esc(siteName)}</strong>.</p>
<h3>Deploy for your community</h3>
<p>Go Stop is free and open source. Deploy your own instance in one click:</p>
<p><a href="https://my.scalingo.com/deploy?source=https://github.com/z3spinner/go-stop" target="_blank" rel="noopener">▶ Deploy on Scalingo</a></p>
<p style="font-size:0.8rem;color:var(--gray-600)">Source: <a href="https://github.com/z3spinner/go-stop" target="_blank" rel="noopener">github.com/z3spinner/go-stop</a> · AGPL-3.0 licence</p>`,
    feedbackTitle:   'Did anyone join your ride?',
    feedbackYes:     'Yes, someone joined',
    feedbackNo:      'No, I drove alone',
    feedbackThanks:  'Thanks!',
    statsTitle:      'This week',
    statsEmpty:      'No confirmed rides yet this week.',
    statsAllTime:    (n) => `All time: ${n} confirmed`,
    btnAllStats:     'All stats →',
    statsPageTitle:  'Stats',
    statsRouteCount: (n) => `${n} ✓`,
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
    labelTo:        'Destination',
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
    notifDenied:      'Notifications bloquées dans les paramètres du navigateur.',
    btnMyRides:       'Mes trajets',
    myRidesTitle:     'Mes trajets',
    labelPhoneCheck:  'Votre numéro de téléphone',
    btnShowRides:     'Voir mes trajets',
    noMyRides:        'Aucun trajet actif trouvé pour ce numéro.',
    btnDelete:        'Supprimer',
    deleteOk:         'Supprimé.',
    deleteErr:        'Impossible de supprimer — numéro incorrect ?',
    labelSearchTime:   'Vers quelle heure ? (optionnel)',
    colOutbound:       'Aller',
    colReturn:         'Retour',
    noRidesCol:        'Aucun trajet disponible.',
    tripTypeLabel:     'Type de trajet',
    tripOneWay:        'Aller simple',
    tripReturn:        'Aller-retour',
    returnSection:     'Trajet retour',
    labelReturnTime:   'Heure de départ retour',
    labelReturnFlex:   'Flexibilité retour',
    btnNotifyRoute:   '🔔 Me prévenir des nouveaux trajets sur ce parcours',
    notifRouteTitle:  'Recevoir des alertes',
    notifRouteBody:   'Vous serez alerté(e) dès qu\'un trajet correspondant à ce parcours est publié. Indiquez vos coordonnées.',
    notifRouteSet:    '✓ Vous serez alerté(e) dès qu\'un trajet correspondant apparaît.',
    btnMyAlerts:      'Mes alertes',
    myAlertsTitle:    'Mes alertes',
    btnShowAlerts:    'Voir mes alertes',
    noMyAlerts:       'Aucune alerte active trouvée pour ce numéro.',
    alertCard:        (r) => `${r.Origin} → ${r.Destination}`,
    detailRideTitle:  'Trajet disponible',
    detailReqTitle:   'Demande de trajet',
    labelDriver:      'Conducteur',
    labelSearcher:    'Passager',
    labelDeparture:   'Départ',
    labelContact:     'Contact',
    footerPrivacy:    'Confidentialité',
    aboutTitle:       'À propos de Go Stop',
    aboutBody:        (siteName) => `<p><strong>Go Stop</strong> est une plateforme locale de covoiturage, à mi-chemin entre l'autostop et le covoiturage formel. Elle met en relation des conducteurs qui proposent un trajet ponctuel et des personnes qui cherchent un stop.</p>
<p>Aucun compte n'est requis. Le contact se fait directement par téléphone.</p>
<h3>Votre communauté</h3>
<p>Cette instance est déployée pour <strong>${esc(siteName)}</strong>.</p>
<h3>Déployer pour votre communauté</h3>
<p>Go Stop est un logiciel libre. Vous pouvez déployer votre propre instance en un clic :</p>
<p><a href="https://my.scalingo.com/deploy?source=https://github.com/z3spinner/go-stop" target="_blank" rel="noopener">▶ Déployer sur Scalingo</a></p>
<p style="font-size:0.8rem;color:var(--gray-600)">Code source : <a href="https://github.com/z3spinner/go-stop" target="_blank" rel="noopener">github.com/z3spinner/go-stop</a> · Licence AGPL-3.0</p>`,
    feedbackTitle:   'Quelqu\'un est-il venu ?',
    feedbackYes:     'Oui, quelqu\'un est venu',
    feedbackNo:      'Non, j\'ai conduit seul(e)',
    feedbackThanks:  'Merci !',
    statsTitle:      'Cette semaine',
    statsEmpty:      'Aucun trajet confirmé cette semaine.',
    statsAllTime:    (n) => `Depuis le début : ${n} confirmés`,
    btnAllStats:     'Toutes les stats →',
    statsPageTitle:  'Statistiques',
    statsRouteCount: (n) => `${n} ✓`,
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
  renderFooter();
  renderHome();
}

function renderFooter() {
  const s = t();
  const footer = document.getElementById('app-footer');
  if (!footer) return;
  footer.innerHTML = `<button class="btn-footer-privacy" id="btn-footer-privacy">${s.footerPrivacy}</button>`;
  document.getElementById('btn-footer-privacy').onclick = showPrivacyModal;
}

// Shows the CURRENT language flag so users know what language they are in.
// Clicking it switches to the other language.
function langToggle() {
  const label = lang === 'en' ? '🇬🇧 EN' : '🇫🇷 FR';
  return `<button class="btn-lang" id="btn-lang">${label}</button>`;
}

function aboutIcon() {
  return `<button class="btn-privacy" id="btn-about" aria-label="${t().aboutTitle}">ⓘ</button>`;
}

function privacyFooter() {
  const s = t();
  return `<footer class="app-footer"><button class="btn-footer-privacy" id="btn-footer-privacy">${s.footerPrivacy}</button></footer>`;
}

function showAboutModal() {
  const s = t();
  const overlay = document.createElement('div');
  overlay.className = 'modal-overlay';
  overlay.innerHTML = `
    <div class="modal">
      <div class="modal-header">
        <h2>${s.aboutTitle}</h2>
        <button class="btn-modal-close" id="btn-modal-close">${s.privacyClose}</button>
      </div>
      <div class="modal-body">${s.aboutBody(SITE_NAME)}</div>
    </div>`;
  document.body.appendChild(overlay);
  overlay.onclick = (e) => { if (e.target === overlay) overlay.remove(); };
  document.getElementById('btn-modal-close').onclick = () => overlay.remove();
}

function showPrivacyModal() {
  const s = t();
  const overlay = document.createElement('div');
  overlay.className = 'modal-overlay';
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

async function api(method, path, body, extraHeaders) {
  const opts = { method, headers: { 'Content-Type': 'application/json', ...extraHeaders } };
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
  const aboutBtn = document.getElementById('btn-about');
  if (aboutBtn) aboutBtn.onclick = showAboutModal;
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

function saveLastSearch(origin, destination) {
  if (origin)      localStorage.setItem('last_origin', origin);
  if (destination) localStorage.setItem('last_destination', destination);
}

function getLastSearch() {
  return {
    origin:      localStorage.getItem('last_origin')      || '',
    destination: localStorage.getItem('last_destination') || '',
  };
}

// Returns a datetime-local string 1 hour from now in the user's LOCAL timezone,
// rounded to nearest 5 min. Uses local getters (not toISOString) because
// datetime-local inputs expect local time, not UTC.
function defaultDeparture() {
  const d = new Date(Date.now() + 60 * 60 * 1000);
  d.setMinutes(Math.ceil(d.getMinutes() / 5) * 5, 0, 0);
  const pad = n => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// ── Views ─────────────────────────────────────────────────────────────────────

function controls() {
  return `<div class="controls">${langToggle()}${aboutIcon()}</div>`;
}

function topBar() {
  return `<div class="top-bar">${controls()}</div>`;
}

function pageBar() {
  const s = t();
  return `<div class="top-bar page-bar"><button class="btn-back" id="back">${s.btnBack}</button>${controls()}</div>`;
}

// Push a URL into browser history when navigating to a view so that
// reload restores the correct page.
function pushRoute(path) {
  if (window.location.pathname !== path) {
    history.pushState({ path }, '', path);
  }
}

// Called on popstate (back/forward button) — re-render whatever the URL says.
window.addEventListener('popstate', () => {
  (async () => { if (!await handleDeepLink()) renderHome(); })();
});

async function renderStats() {
  pushRoute('/stats');
  const s = t();
  app.innerHTML = `
    ${pageBar()}
    <h2>${s.statsPageTitle}</h2>
    <div id="stats-content"><p class="section-hint">…</p></div>`;
  document.getElementById('back').onclick = renderHome;
  bindControls();

  try {
    const stats = await api('GET', '/stats');
    const totalLine = stats.total_confirmed > 0
      ? `<p class="stats-total">${s.statsAllTime(stats.total_confirmed)}</p>`
      : '';
    const rows = stats.top_routes && stats.top_routes.length
      ? stats.top_routes.map(r => `
          <div class="stats-row">
            <span class="stats-route">${esc(r.Origin)} → ${esc(r.Destination)}</span>
            <span class="stats-count">${s.statsRouteCount(r.Count)}</span>
          </div>`).join('')
      : `<p class="section-hint">${s.statsEmpty}</p>`;

    document.getElementById('stats-content').innerHTML = `
      ${totalLine}
      <div class="stats-week-title">${s.statsTitle}</div>
      ${rows}`;
  } catch (err) {
    document.getElementById('stats-content').innerHTML =
      `<p class="error">${esc(err.message)}</p>`;
  }
}

async function renderHome() {
  history.replaceState({ path: '/' }, '', '/');
  const s = t();
  app.innerHTML = `
    ${topBar()}
    <div class="hero">
      <h1>${esc(SITE_NAME)}</h1>
      <p class="tagline">${s.tagline}</p>
      <button class="btn btn-primary" id="btn-driver">${s.btnDriver}</button>
      <button class="btn btn-secondary" id="btn-searcher">${s.btnSearcher}</button>
      <div class="ghost-row">
        <button class="btn-ghost-inline" id="btn-my-rides">${s.btnMyRides}</button>
        <span class="ghost-sep">·</span>
        <button class="btn-ghost-inline" id="btn-my-alerts">${s.btnMyAlerts}</button>
      </div>
    </div>
    <div id="home-stats"></div>`;
  document.getElementById('btn-driver').onclick = renderPostRide;
  document.getElementById('btn-searcher').onclick = renderSearchRides;
  document.getElementById('btn-my-rides').onclick = renderMyRides;
  document.getElementById('btn-my-alerts').onclick = renderMyAlerts;
  bindControls();
  loadHomeStats();
}

async function loadHomeStats() {
  const s = t();
  try {
    const stats = await api('GET', '/stats');
    if (!stats.top_routes || !stats.top_routes.length) return;
    const rows = stats.top_routes.map(r =>
      `<div class="stats-row">
        <span class="stats-route">${esc(r.Origin)} → ${esc(r.Destination)}</span>
        <span class="stats-count">${s.statsRouteCount(r.Count)}</span>
      </div>`
    ).join('');
    document.getElementById('home-stats').innerHTML = `
      <div class="stats-widget">
        <div class="stats-widget-title">${s.statsTitle}</div>
        ${rows}
        <button class="btn-all-stats" id="btn-all-stats">${s.btnAllStats}</button>
      </div>`;
    document.getElementById('btn-all-stats').onclick = renderStats;
  } catch {
    // silently omit if unavailable
  }
}

async function renderPostRide() {
  pushRoute('/post-ride');
  const s = t();
  const p = getProfile();
  const dests = await getDestinations();
  app.innerHTML = `
    ${pageBar()}
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
      <div class="form-group trip-type-group">
        <label>${s.tripTypeLabel}</label>
        <div class="trip-type-toggle">
          <button type="button" class="trip-type-btn active" id="btn-oneway">${s.tripOneWay}</button>
          <button type="button" class="trip-type-btn" id="btn-return">${s.tripReturn}</button>
        </div>
      </div>
      <div id="return-section" class="return-section hidden">
        <div class="return-section-title">${s.returnSection}</div>
        <div class="form-group"><label>${s.labelReturnTime}</label><input name="return_departure_at" type="datetime-local"></div>
        <div class="form-group">
          <label>${s.labelReturnFlex}</label>
          <select name="return_flexibility">
            <option value="0">${s.flexExact}</option>
            <option value="30" selected>${s.flex30}</option>
            <option value="60">${s.flex60}</option>
          </select>
        </div>
      </div>
      <button class="btn btn-primary" type="submit">${s.btnPostRide}</button>
      <div class="error" id="err"></div>
    </form>`;
  document.getElementById('back').onclick = renderHome;
  bindControls();

  let isReturn = false;
  document.getElementById('btn-oneway').onclick = () => {
    isReturn = false;
    document.getElementById('btn-oneway').classList.add('active');
    document.getElementById('btn-return').classList.remove('active');
    document.getElementById('return-section').classList.add('hidden');
    document.querySelector('[name=return_departure_at]').required = false;
  };
  document.getElementById('btn-return').onclick = () => {
    isReturn = true;
    document.getElementById('btn-return').classList.add('active');
    document.getElementById('btn-oneway').classList.remove('active');
    const sec = document.getElementById('return-section');
    sec.classList.remove('hidden');
    const retInput = document.querySelector('[name=return_departure_at]');
    retInput.required = true;
    if (!retInput.value) retInput.value = defaultDeparture();
  };

  document.getElementById('ride-form').onsubmit = async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const phone = fd.get('phone');
    const origin = fd.get('origin');
    const destination = fd.get('destination');
    saveProfile(fd.get('driver_name'), phone);
    try {
      await api('POST', '/rides', {
        driver_name: fd.get('driver_name'),
        phone,
        origin,
        destination,
        departure_at: new Date(fd.get('departure_at')).toISOString(),
        flexibility: parseInt(fd.get('flexibility')),
      });
      if (isReturn) {
        await api('POST', '/rides', {
          driver_name: fd.get('driver_name'),
          phone,
          origin: destination,
          destination: origin,
          departure_at: new Date(fd.get('return_departure_at')).toISOString(),
          flexibility: parseInt(fd.get('return_flexibility')),
        });
      }
      renderNotificationPrompt(phone, renderHome);
    } catch (err) {
      document.getElementById('err').textContent = err.message;
    }
  };
}

async function renderSearchRides() {
  pushRoute('/search');
  const s = t();
  const ls = getLastSearch();
  const dests = await getDestinations();
  app.innerHTML = `
    ${pageBar()}
    <h2>${s.findTitle}</h2>
    <form id="search-form">
      <div class="form-group"><label>${s.labelFrom}</label><input name="origin" value="${esc(ls.origin)}" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>${s.labelTo}</label><input name="destination" value="${esc(ls.destination)}" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <div class="form-group"><label class="label-optional">${s.labelSearchTime}</label><input name="departure_at" type="datetime-local"></div>
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
    const deptRaw = fd.get('departure_at');
    saveLastSearch(origin, dest);
    const results = document.getElementById('results');
    const timeParam = deptRaw ? `&departure_at=${encodeURIComponent(new Date(deptRaw).toISOString())}` : '';
    const fwdUrl = `/rides?origin=${encodeURIComponent(origin)}&destination=${encodeURIComponent(dest)}${timeParam}`;
    const retUrl = `/rides?origin=${encodeURIComponent(dest)}&destination=${encodeURIComponent(origin)}${timeParam}`;
    try {
      const [rides, returnRides] = await Promise.all([api('GET', fwdUrl), api('GET', retUrl)]);

      function rideCard(r) {
        return `<div class="card card-compact">
          <div class="card-meta">${formatTime(r.DepartureAt)} <span class="tag">${s.flexLabel[r.Flexibility] || esc(r.Flexibility) + ' min'}</span></div>
          <div class="card-contact"><strong>${esc(r.DriverName)}</strong><br><a href="tel:${esc(r.Phone)}">${esc(r.Phone)}</a></div>
        </div>`;
      }

      function colEmpty(fromLoc, toLoc) {
        return `<div class="col-empty">
          <p>${s.noRidesCol}</p>
          <button class="btn-notify-route col-notify" data-from="${esc(fromLoc)}" data-to="${esc(toLoc)}" data-dept="${esc(deptRaw)}">${s.btnNotifyRoute}</button>
        </div>`;
      }

      results.innerHTML = `
        <div class="results-grid">
          <div class="results-col">
            <div class="results-col-header">${esc(origin)} → ${esc(dest)}</div>
            ${rides.length ? rides.map(rideCard).join('') : colEmpty(origin, dest)}
          </div>
          <div class="results-col">
            <div class="results-col-header">${esc(dest)} → ${esc(origin)}</div>
            ${returnRides.length ? returnRides.map(rideCard).join('') : colEmpty(dest, origin)}
          </div>
        </div>`;

      results.querySelectorAll('.col-notify').forEach(btn => {
        btn.onclick = () => renderNotifyRoute(btn.dataset.from, btn.dataset.to, btn.dataset.dept);
      });
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
    ${pageBar()}
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

// ── My rides ──────────────────────────────────────────────────────────────────

function renderMyRides() {
  pushRoute('/my-rides');
  const s = t();
  const p = getProfile();
  app.innerHTML = `
    ${pageBar()}
    <h2>${s.myRidesTitle}</h2>
    <form id="my-rides-form">
      <div class="form-group"><label>${s.labelPhoneCheck}</label><input name="phone" type="tel" value="${esc(p.phone)}" required></div>
      <button class="btn btn-primary" type="submit">${s.btnShowRides}</button>
    </form>
    <div id="my-rides-list"></div>`;
  document.getElementById('back').onclick = renderHome;
  bindControls();
  document.getElementById('my-rides-form').onsubmit = async (e) => {
    e.preventDefault();
    const phone = new FormData(e.target).get('phone');
    const list = document.getElementById('my-rides-list');
    try {
      const rides = await api('GET', '/rides', null, { 'X-Phone': phone });
      if (!rides.length) {
        list.innerHTML = `<div class="empty">${s.noMyRides}</div>`;
        return;
      }
      list.innerHTML = rides.map(r => {
        const isPast = new Date(r.DepartureAt) < new Date();
        const needsFeedback = isPast && !r.FeedbackGiven;
        const feedbackSection = needsFeedback ? `
          <div class="feedback-prompt" id="fb-${esc(r.ID)}">
            <span class="feedback-question">${s.feedbackTitle}</span>
            <div class="feedback-btns">
              <button class="btn-fb-yes" data-id="${esc(r.ID)}" data-phone="${esc(phone)}">${s.feedbackYes}</button>
              <button class="btn-fb-no"  data-id="${esc(r.ID)}" data-phone="${esc(phone)}">${s.feedbackNo}</button>
            </div>
            <div class="feedback-thanks hidden" id="fb-thanks-${esc(r.ID)}">${s.feedbackThanks}</div>
          </div>` : '';
        return `
          <div class="card" id="card-${esc(r.ID)}">
            <div class="card-route">${esc(r.Origin)} → ${esc(r.Destination)}</div>
            <div class="card-meta">${formatTime(r.DepartureAt)} <span class="tag">${s.flexLabel[r.Flexibility] || esc(r.Flexibility) + ' min'}</span></div>
            ${feedbackSection}
            <button class="btn btn-danger btn-delete" data-id="${esc(r.ID)}" data-phone="${esc(phone)}">${s.btnDelete}</button>
            <div class="delete-msg" id="msg-${esc(r.ID)}"></div>
          </div>`;
      }).join('');

      // Bind feedback buttons
      list.querySelectorAll('.btn-fb-yes, .btn-fb-no').forEach(btn => {
        btn.onclick = async () => {
          const taken = btn.classList.contains('btn-fb-yes');
          try {
            await api('POST', `/rides/${btn.dataset.id}/feedback`, {
              phone: btn.dataset.phone, taken,
            });
            const prompt = document.getElementById('fb-' + btn.dataset.id);
            const btns = prompt.querySelector('.feedback-btns');
            const question = prompt.querySelector('.feedback-question');
            if (btns) btns.remove();
            if (question) question.remove();
            document.getElementById('fb-thanks-' + btn.dataset.id).classList.remove('hidden');
          } catch {
            // silently fail — will retry next visit
          }
        };
      });

      // Bind delete buttons
      list.querySelectorAll('.btn-delete').forEach(btn => {
        btn.onclick = async () => {
          try {
            await api('DELETE', `/rides/${btn.dataset.id}`, { phone: btn.dataset.phone });
            const card = document.getElementById('card-' + btn.dataset.id);
            card.style.opacity = '0.4';
            btn.disabled = true;
            document.getElementById('msg-' + btn.dataset.id).textContent = s.deleteOk;
          } catch {
            document.getElementById('msg-' + btn.dataset.id).textContent = s.deleteErr;
          }
        };
      });
    } catch (err) {
      const div = document.createElement('div');
      div.className = 'error';
      div.textContent = err.message;
      list.replaceChildren(div);
    }
  };
  // Auto-submit if phone is pre-filled
  if (p.phone) document.getElementById('my-rides-form').requestSubmit();
}

// ── Notify me on route ────────────────────────────────────────────────────────

async function renderNotifyRoute(origin, destination, departureAt = '') {
  const s = t();
  const p = getProfile();
  const dests = await getDestinations();
  // Use the time from the search if provided, otherwise default to 1h from now.
  const deptValue = departureAt || defaultDeparture();
  app.innerHTML = `
    ${pageBar()}
    <h2>${s.notifRouteTitle}</h2>
    <p class="section-hint">${s.notifRouteBody}</p>
    <form id="notify-form">
      <div class="form-group"><label>${s.labelName}</label><input name="searcher_name" value="${esc(p.name)}" required></div>
      <div class="form-group"><label>${s.labelPhone}</label><input name="phone" type="tel" value="${esc(p.phone)}" required></div>
      <div class="form-group"><label>${s.labelFrom}</label><input name="origin" value="${esc(origin)}" list="dests-from" required autocomplete="off">${destinationList('dests-from', dests)}</div>
      <div class="form-group"><label>${s.labelTo}</label><input name="destination" value="${esc(destination)}" list="dests-to" required autocomplete="off">${destinationList('dests-to', dests)}</div>
      <div class="form-group"><label>${s.labelDatetime}</label><input name="departure_at" type="datetime-local" value="${esc(deptValue)}" required></div>
      <div class="form-group">
        <label>${s.labelFlex}</label>
        <select name="flexibility">
          <option value="0">${s.flexExact}</option>
          <option value="30" selected>${s.flex30}</option>
          <option value="60">${s.flex60}</option>
        </select>
      </div>
      <button class="btn btn-primary" type="submit">${s.notifEnable}</button>
      <div class="error" id="err"></div>
    </form>`;
  document.getElementById('back').onclick = () => renderSearchRides();
  bindControls();
  document.getElementById('notify-form').onsubmit = async (e) => {
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
      renderNotificationPrompt(phone, () => {
        app.innerHTML = `<div class="notif-prompt"><div class="notif-icon">✓</div><p>${s.notifRouteSet}</p><button class="btn btn-secondary" id="btn-home">${s.btnBack}</button></div>`;
        document.getElementById('btn-home').onclick = renderHome;
      });
    } catch (err) {
      document.getElementById('err').textContent = err.message;
    }
  };
}

// ── My alerts (waiting requests) ─────────────────────────────────────────────

function renderMyAlerts() {
  pushRoute('/my-alerts');
  const s = t();
  const p = getProfile();
  app.innerHTML = `
    ${pageBar()}
    <h2>${s.myAlertsTitle}</h2>
    <form id="my-alerts-form">
      <div class="form-group"><label>${s.labelPhoneCheck}</label><input name="phone" type="tel" value="${esc(p.phone)}" required></div>
      <button class="btn btn-primary" type="submit">${s.btnShowAlerts}</button>
    </form>
    <div id="my-alerts-list"></div>`;
  document.getElementById('back').onclick = renderHome;
  bindControls();
  document.getElementById('my-alerts-form').onsubmit = async (e) => {
    e.preventDefault();
    const phone = new FormData(e.target).get('phone');
    const list = document.getElementById('my-alerts-list');
    try {
      const reqs = await api('GET', '/requests', null, { 'X-Phone': phone });
      if (!reqs.length) {
        list.innerHTML = `<div class="empty">${s.noMyAlerts}</div>`;
        return;
      }
      list.innerHTML = reqs.map(r => `
        <div class="card" id="card-${esc(r.ID)}">
          <div class="card-route">${esc(r.Origin)} → ${esc(r.Destination)}</div>
          <div class="card-meta">${formatTime(r.DepartureAt)} <span class="tag">${s.flexLabel[r.Flexibility] || esc(r.Flexibility) + ' min'}</span></div>
          <button class="btn btn-danger btn-delete" data-id="${esc(r.ID)}" data-phone="${esc(phone)}">${s.btnDelete}</button>
          <div class="delete-msg" id="msg-${esc(r.ID)}"></div>
        </div>`).join('');
      list.querySelectorAll('.btn-delete').forEach(btn => {
        btn.onclick = async () => {
          try {
            await api('DELETE', `/requests/${btn.dataset.id}`, { phone: btn.dataset.phone });
            const card = document.getElementById('card-' + btn.dataset.id);
            card.style.opacity = '0.4';
            btn.disabled = true;
            document.getElementById('msg-' + btn.dataset.id).textContent = s.deleteOk;
          } catch {
            document.getElementById('msg-' + btn.dataset.id).textContent = s.deleteErr;
          }
        };
      });
    } catch (err) {
      const div = document.createElement('div');
      div.className = 'error';
      div.textContent = err.message;
      list.replaceChildren(div);
    }
  };
  if (p.phone) document.getElementById('my-alerts-form').requestSubmit();
}

// ── Deep link from push notification ─────────────────────────────────────────

async function renderItemDetail(type, item) {
  const s = t();
  const isRide = type === 'rides';
  const title = isRide ? s.detailRideTitle : s.detailReqTitle;
  const personLabel = isRide ? s.labelDriver : s.labelSearcher;
  const name = isRide ? item.DriverName : item.SearcherName;
  const phone = item.Phone;

  app.innerHTML = `
    ${pageBar()}
    <h2>${title}</h2>
    <div class="card detail-card">
      <div class="card-route">${esc(item.Origin)} → ${esc(item.Destination)}</div>
      <div class="card-meta">${formatTime(item.DepartureAt)} <span class="tag">${s.flexLabel[item.Flexibility] || esc(item.Flexibility) + ' min'}</span></div>
      <table class="detail-table">
        <tr><td>${personLabel}</td><td><strong>${esc(name)}</strong></td></tr>
        <tr><td>${s.labelContact}</td><td><a href="tel:${esc(phone)}">${esc(phone)}</a></td></tr>
      </table>
    </div>`;
  document.getElementById('back').onclick = () => {
    history.replaceState({}, '', '/');
    renderHome();
  };
  bindControls();
  history.replaceState({}, '', '/');
}

async function handleDeepLink() {
  const path = window.location.pathname;

  // Item detail from push notification
  const itemMatch = path.match(/^\/(rides|requests)\/([^/]+)$/);
  if (itemMatch) {
    const [, type, id] = itemMatch;
    try {
      const item = await api('GET', `/${type}/${id}`);
      renderItemDetail(type, item);
    } catch {
      history.replaceState({}, '', '/');
      renderHome();
    }
    return true;
  }

  // SPA view routes
  switch (path) {
    case '/post-ride':    await renderPostRide();    return true;
    case '/search':       await renderSearchRides(); return true;
    case '/my-rides':     renderMyRides();           return true;
    case '/my-alerts':    renderMyAlerts();          return true;
    case '/post-request': await renderPostRequest(); return true;
    case '/stats':        await renderStats();       return true;
  }

  return false;
}

(async () => {
  try {
    const cfg = await api('GET', '/config');
    SITE_NAME = cfg.siteName || 'Go-Stop';
    document.title = SITE_NAME;
  } catch {}
  renderFooter();
  if (!await handleDeepLink()) renderHome();
})();

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js').catch(console.error);
}
