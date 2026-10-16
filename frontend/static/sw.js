// web/js/sw.js
'use strict';

self.addEventListener('push', (event) => {
  let data = {};
  try { data = event.data.json(); } catch {}

  const title = data.title || 'Go-Stop';
  const inboxId = data.inbox_id || '';
  const options = {
    body: data.body || '',
    icon: '/icon-192.png',
    badge: '/icon-192.png',
    data: { url: data.url || '/', inboxId },
  };
  // A reminder about the same inbox notification replaces the earlier push
  // instead of stacking another one (and still alerts: renotify).
  if (inboxId) {
    options.tag = 'inbox-' + inboxId;
    options.renotify = true;
  }

  event.waitUntil(
    Promise.all([
      self.registration.showNotification(title, options),
      // Let open tabs refresh the in-app inbox badge right away.
      clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
        for (const client of clientList) client.postMessage({ type: 'inbox-updated' });
      }),
    ])
  );
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const { url = '/', inboxId = '' } = event.notification.data || {};
  // Tapping the push counts as reading its inbox notification (so reminders
  // stop): an open tab is told directly, a new one gets ?n=<id> and the app
  // marks it read on load.
  const target = inboxId ? url + (url.includes('?') ? '&' : '?') + 'n=' + encodeURIComponent(inboxId) : url;
  const wantPath = new URL(url, self.location.origin).pathname;
  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      for (const client of clientList) {
        if (new URL(client.url).pathname === wantPath && 'focus' in client) {
          if (inboxId) client.postMessage({ type: 'inbox-read', id: inboxId });
          return client.focus();
        }
      }
      if (clients.openWindow) return clients.openWindow(target);
    })
  );
});
