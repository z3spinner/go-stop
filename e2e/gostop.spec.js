// @ts-check
const { test, expect } = require('@playwright/test');

const DRIVER   = { name: 'Alice', phone: '0611000001' };
const SEARCHER = { name: 'Bob',   phone: '0622000002' };
const BASE = 'http://localhost:8080';

// ── Helpers ───────────────────────────────────────────────────────────────────

async function setProfile(page, user) {
  // The profile/last-search stores write RAW strings (legacy-compatible), so
  // user_name/user_phone are plain values (not JSON-quoted). `lang` is also raw
  // (Paraglide reads it raw). addInitScript applies the values before each
  // (subsequent) navigation so the SvelteKit app picks them up on load.
  await page.addInitScript((u) => {
    localStorage.setItem('user_name', u.name);
    localStorage.setItem('user_phone', u.phone);
    localStorage.setItem('lang', 'fr');
  }, user);
}

async function setFr(page) {
  // `lang` is read raw by the Paraglide locale strategy.
  await page.addInitScript(() => localStorage.setItem('lang', 'fr'));
}

// Navigate and ensure French UI
async function gotoFr(page, url = BASE) {
  await page.goto(url);
  await setFr(page);
}

// ── 1. Home page ──────────────────────────────────────────────────────────────
test('home page loads with title and buttons', async ({ page }) => {
  await page.goto(BASE);
  await setFr(page);
  await page.reload();

  await expect(page).toHaveTitle('Go Stop Saillans!');
  await expect(page.locator('h1')).toContainText('Go Stop Saillans!');
  await expect(page.locator('button.btn-primary')).toContainText('Je conduis');
  await expect(page.locator('button.btn-secondary')).toContainText('Je cherche un stop');
  await expect(page.locator('button.btn-ghost-inline').first()).toBeVisible();
});

// ── 2. Stats page ─────────────────────────────────────────────────────────────
test('stats page loads and back button returns home', async ({ page }) => {
  await page.goto(BASE);
  await setFr(page);
  await page.reload();

  await page.locator('footer').getByText('Statistiques').click();
  await expect(page.locator('h2')).toBeVisible();
  await page.locator('text=← Retour').click();
  await expect(page.locator('h1')).toContainText('Go Stop Saillans!');
});

// ── 3. Post a ride ────────────────────────────────────────────────────────────
test('driver posts a ride and is redirected to My Rides', async ({ page }) => {
  // Grant notifications so renderNotificationPrompt calls onDone (renderMyRides) immediately
  // rather than showing the prompt page (which happens when permission is 'default')
  await page.context().grantPermissions(['notifications'], { origin: BASE });

  await page.goto(BASE);
  await setFr(page);
  await page.reload();

  await page.click('button.btn-primary');
  await expect(page).toHaveURL(/post-ride/);

  await page.fill('input[name=driver_name]', DRIVER.name);
  await page.fill('input[name=phone]', DRIVER.phone);
  await page.fill('input[name=origin]', 'Saillans');
  await page.fill('input[name=destination]', 'Crest');
  // departure_at is pre-filled with defaultDeparture() — use as-is to avoid Chrome datetime-local quirks

  const [response] = await Promise.all([
    page.waitForResponse(r => r.url().includes('/api/rides') && r.request().method() === 'POST'),
    page.click('button[type=submit]'),
  ]);
  expect(response.status()).toBe(201);

  await page.waitForFunction(
    () => document.querySelector('h2')?.textContent?.match(/My rides|Mes trajets/i),
    { timeout: 10000 }
  );
  await expect(page.locator('.card-route', { hasText: 'Saillans → Crest' }).first()).toBeVisible();
});

// ── 4. Public ride list strips phone, shows driver name ───────────────────────
test('public ride list hides phone and shows driver name', async ({ page }) => {
  await page.goto(BASE);
  const rides = await page.evaluate(() =>
    fetch('/api/rides').then(r => r.json())
  );
  const withPhone = rides.find(r => r.Phone);
  expect(withPhone).toBeUndefined();
  const withName = rides.find(r => r.DriverName === DRIVER.name);
  expect(withName).toBeDefined();
});

// ── 5. Search finds ride, driver name visible, phone hidden ───────────────────
test('searcher finds ride with driver name but no phone', async ({ page }) => {
  await page.goto(`${BASE}/search?origin=Saillans&destination=Crest`);
  await page.waitForSelector('.card');

  // Scope to the Alice ride seeded by test 3 (the shared dev DB may hold other
  // Saillans→Crest rides from manual testing that would otherwise sort first).
  const aliceCard = page.locator('.card', { hasText: DRIVER.name }).first();
  await expect(aliceCard).toContainText(DRIVER.name);
  await expect(aliceCard).not.toContainText(DRIVER.phone);
  await expect(aliceCard.locator('.btn-interest')).toBeVisible();
});

// ── 6. Full mutual interest flow ──────────────────────────────────────────────
test('searcher requests contact, driver accepts, phone revealed to both', async ({ page }) => {
  await page.goto(BASE);

  // Use a unique route per run to avoid ON CONFLICT reusing old accepted interests
  const uniqueOrigin = `TestA${Date.now()}`;
  const uniqueDest   = `TestB${Date.now()}`;

  const rideData = await page.evaluate(async ({ driver, origin, destination }) => {
    const r = await fetch('/api/rides', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        driver_name: driver.name, phone: driver.phone,
        origin, destination,
        departure_at: new Date(Date.now() + 3 * 3600 * 1000).toISOString(),
        flexibility: 0,
      }),
    });
    return r.json();
  }, { driver: DRIVER, origin: uniqueOrigin, destination: uniqueDest });
  expect(rideData.ID).toBeTruthy();

  // Set up as Bob (searcher) — clear any old interest keys
  await page.evaluate(() => {
    for (const k of Object.keys(localStorage).filter(k => k.startsWith('interest_'))) {
      localStorage.removeItem(k);
    }
  });
  await setProfile(page, SEARCHER);
  await page.goto(`${BASE}/search?origin=${encodeURIComponent(uniqueOrigin)}&destination=${encodeURIComponent(uniqueDest)}`);
  await page.waitForSelector('.btn-interest');

  // Request contact on the ride
  const firstCard = page.locator('.card').first();
  const rideId = await firstCard.locator('.btn-interest').getAttribute('data-ride-id');
  await firstCard.locator('.btn-interest').click();

  // After requesting, the card switches to the pending/resend state and shows a
  // confirmation message (the new UI offers a re-sendable button rather than a
  // permanently-disabled one).
  await expect(firstCard.locator('.btn-interest-resend')).toBeVisible({ timeout: 5000 });
  await expect(firstCard.locator('.interest-state')).toBeVisible({ timeout: 5000 });

  // Get interest ID from localStorage
  const interestId = await page.evaluate(id =>
    localStorage.getItem('interest_' + id), rideId
  );
  expect(interestId).toBeTruthy();

  // Pending — contact blocked
  const status404 = await page.evaluate(async ({ id, phone }) => {
    const r = await fetch(`/api/interests/${id}/contact`, { headers: { 'X-Phone': phone } });
    return r.status;
  }, { id: interestId, phone: SEARCHER.phone });
  expect(status404).toBe(404);

  // Driver accepts
  const acceptStatus = await page.evaluate(async ({ id, phone }) => {
    const r = await fetch(`/api/interests/${id}/accept`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone }),
    });
    return r.status;
  }, { id: interestId, phone: DRIVER.phone });
  expect(acceptStatus).toBe(200);

  // Searcher retrieves driver phone
  const forSearcher = await page.evaluate(async ({ id, phone }) => {
    const r = await fetch(`/api/interests/${id}/contact`, { headers: { 'X-Phone': phone } });
    return r.json();
  }, { id: interestId, phone: SEARCHER.phone });
  expect(forSearcher.phone).toBeTruthy();
  expect(forSearcher.role).toBe('driver');

  // Driver retrieves searcher phone
  const forDriver = await page.evaluate(async ({ id, phone }) => {
    const r = await fetch(`/api/interests/${id}/contact`, { headers: { 'X-Phone': phone } });
    return r.json();
  }, { id: interestId, phone: DRIVER.phone });
  expect(forDriver.phone).toBeTruthy();
  expect(forDriver.role).toBe('searcher');

  // Stranger blocked
  const strangerStatus = await page.evaluate(async ({ id }) => {
    const r = await fetch(`/api/interests/${id}/contact`, { headers: { 'X-Phone': '0699999999' } });
    return r.status;
  }, { id: interestId });
  expect(strangerStatus).toBe(403);
});

// ── 7. Driver sees searcher name in My Rides ──────────────────────────────────
test('driver sees pending interests with searcher name', async ({ page }) => {
  await page.goto(BASE);

  // Self-seed a ride owned by the driver with a *pending* interest from the
  // searcher. (Test 6 accepts its interest, and an accepted interest renders the
  // phone rather than the name — so nothing leaves a pending interest behind for
  // this test to find. Seeding our own makes it order-independent.)
  const uniqueOrigin = `PendA${Date.now()}`;
  const uniqueDest   = `PendB${Date.now()}`;
  const rideId = await page.evaluate(async ({ driver, searcher, origin, destination }) => {
    const r = await fetch('/api/rides', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        driver_name: driver.name, phone: driver.phone,
        origin, destination,
        departure_at: new Date(Date.now() + 3 * 3600 * 1000).toISOString(),
        flexibility: 0,
      }),
    });
    const ride = await r.json();
    // Searcher expresses interest; driver does NOT accept -> the interest stays
    // pending, so My Rides renders the searcher's name.
    await fetch(`/api/rides/${ride.ID}/interest`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone: searcher.phone, name: searcher.name }),
    });
    return ride.ID;
  }, { driver: DRIVER, searcher: SEARCHER, origin: uniqueOrigin, destination: uniqueDest });
  expect(rideId).toBeTruthy();

  // Navigate to My Rides — the gate form auto-submits when a profile phone is set.
  await setProfile(page, DRIVER);
  await page.goto('/my-rides');
  await page.waitForSelector('.card');

  // The pending interest on the seeded ride renders the searcher's name.
  await expect(page.locator(`#card-${rideId}`)).toContainText(SEARCHER.name, { timeout: 5000 });
});

// ── 7b. Searcher cancels a pending contact request ────────────────────────────
test('searcher cancels a pending contact request', async ({ page }) => {
  await page.goto(BASE);

  // Seed a unique ride so we own a fresh, un-accepted interest to cancel.
  const uniqueOrigin = `CancA${Date.now()}`;
  const uniqueDest   = `CancB${Date.now()}`;
  const rideId = await page.evaluate(async ({ driver, origin, destination }) => {
    const r = await fetch('/api/rides', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        driver_name: driver.name, phone: driver.phone, origin, destination,
        departure_at: new Date(Date.now() + 3 * 3600 * 1000).toISOString(), flexibility: 0,
      }),
    });
    return (await r.json()).ID;
  }, { driver: DRIVER, origin: uniqueOrigin, destination: uniqueDest });
  expect(rideId).toBeTruthy();

  // As Bob, express interest via the search UI.
  await page.evaluate(() => {
    for (const k of Object.keys(localStorage).filter(k => k.startsWith('interest_'))) localStorage.removeItem(k);
  });
  await setProfile(page, SEARCHER);
  await page.goto(`${BASE}/search?origin=${encodeURIComponent(uniqueOrigin)}&destination=${encodeURIComponent(uniqueDest)}`);
  await page.waitForSelector('.btn-interest');
  const card = page.locator('.card').first();
  await card.locator('.btn-interest').click();

  // Pending state shows a cancel button; capture the interest id.
  await expect(card.locator('.btn-interest-cancel')).toBeVisible({ timeout: 5000 });
  const interestId = await page.evaluate(id => localStorage.getItem('interest_' + id), rideId);
  expect(interestId).toBeTruthy();

  // Expressing interest pops the "enable notifications" modal; dismiss it so its
  // overlay doesn't intercept the cancel click.
  const overlay = page.locator('.modal-overlay');
  await overlay.waitFor({ state: 'visible', timeout: 5000 }).catch(() => {});
  await page.keyboard.press('Escape');
  await expect(overlay).toHaveCount(0, { timeout: 5000 });

  // Cancel it.
  await card.locator('.btn-interest-cancel').click();

  // UI reverts (no cancel/resend button) and the localStorage key is cleared.
  await expect(card.locator('.btn-interest-cancel')).toHaveCount(0, { timeout: 5000 });
  await expect.poll(() => page.evaluate(id => localStorage.getItem('interest_' + id), rideId)).toBeNull();

  // Backend: the interest row is gone, so contact lookup 404s.
  const status = await page.evaluate(async ({ id, phone }) => {
    const r = await fetch(`/api/interests/${id}/contact`, { headers: { 'X-Phone': phone } });
    return r.status;
  }, { id: interestId, phone: SEARCHER.phone });
  expect(status).toBe(404);
});

// ── 7c. Ride detail page: navigation, share, server OG, no phone leak ──────────
async function seedRide(page, origin, destination, flexibility = 30) {
  return page.evaluate(async ({ driver, origin, destination, flexibility }) => {
    const r = await fetch('/api/rides', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        driver_name: driver.name, phone: driver.phone, origin, destination,
        departure_at: new Date(Date.now() + 3 * 3600 * 1000).toISOString(), flexibility,
      }),
    });
    return (await r.json()).ID;
  }, { driver: DRIVER, origin, destination, flexibility });
}

test('ride detail page is public-safe: no phone in API or page, OG meta server-rendered', async ({ page, request }) => {
  await page.goto(BASE);
  const o = `OgA${Date.now()}`, d = `OgB${Date.now()}`;
  const rideId = await seedRide(page, o, d, 30);
  expect(rideId).toBeTruthy();

  // Public API must NOT leak the driver's phone.
  const ride = await (await request.get(`${BASE}/api/rides/${rideId}`)).json();
  expect(ride.Phone).toBeUndefined();
  expect(ride.Origin).toBe(o);

  // The server injects per-ride Open Graph tags into the raw HTML (what crawlers
  // see — they don't run JS).
  const html = await (await request.get(`${BASE}/rides/${rideId}`)).text();
  expect(html).toContain(`<meta property="og:title" content="${o} → ${d}"/>`);
  expect(html).toContain('property="og:image"');
  expect(html).toContain('name="twitter:card" content="summary_large_image"');

  // The rendered page shows the request-contact action + a share button, no phone.
  await setProfile(page, SEARCHER);
  await page.goto(`${BASE}/rides/${rideId}`);
  await expect(page.locator('.detail-card .btn-interest')).toBeVisible({ timeout: 5000 });
  await expect(page.locator('.btn-share')).toBeVisible();
  await expect(page.locator('a[href^="tel:"]')).toHaveCount(0);
});

test('clicking a ride card opens the ride detail page', async ({ page }) => {
  await page.goto(BASE);
  const o = `NavA${Date.now()}`, d = `NavB${Date.now()}`;
  const rideId = await seedRide(page, o, d, 0);
  expect(rideId).toBeTruthy();

  await setProfile(page, SEARCHER);
  await page.goto(`${BASE}/search?origin=${encodeURIComponent(o)}&destination=${encodeURIComponent(d)}`);
  await page.locator(`a.card-detail-link[data-ride-id="${rideId}"]`).first().click();

  await expect(page).toHaveURL(new RegExp(`/rides/${rideId}`));
  await expect(page.locator('.detail-card')).toBeVisible();
  await expect(page.locator('.btn-share')).toBeVisible();
});

test('home page has a share button', async ({ page }) => {
  await page.goto(BASE);
  await setFr(page);
  await page.reload();
  await expect(page.locator('.btn-share')).toBeVisible();
});

// ── 7d. Dedicated feedback screen (where the reminder notification lands) ──────
test('feedback screen records the driver answer as the main content', async ({ page }) => {
  await page.goto(BASE);
  const o = `FbA${Date.now()}`, d = `FbB${Date.now()}`;
  const rideId = await seedRide(page, o, d, 0); // owned by DRIVER
  expect(rideId).toBeTruthy();

  await setProfile(page, DRIVER);
  await page.goto(`${BASE}/rides/${rideId}/feedback`);

  // The two answer buttons are the page's main content.
  await expect(page.locator('.feedback-yes')).toBeVisible();
  await expect(page.locator('.feedback-no')).toBeVisible();

  // Answering "No" posts the feedback and confirms.
  const [resp] = await Promise.all([
    page.waitForResponse(r => r.url().includes(`/api/rides/${rideId}/feedback`) && r.request().method() === 'POST'),
    page.locator('.feedback-no').click(),
  ]);
  expect(resp.status()).toBe(204);
  await expect(page.locator('.feedback-done')).toBeVisible();
});

// ── 8. Alert creation — all 4 modes ──────────────────────────────────────────
test('searcher creates alerts in all four modes', async ({ page }) => {
  await setProfile(page, SEARCHER);
  await page.goto(BASE);

  for (const { mode, extra } of [
    { mode: 'time',    extra: async p => {
        await p.fill('input[name=alert_date]', '2030-12-01');
        await p.fill('input[name=alert_time]', '09:00');
    }},
    { mode: 'day',     extra: async p => {
        await p.fill('input[name=alert_date]', '2030-12-01');
    }},
    { mode: 'daily',   extra: async p => {
        await p.fill('input[name=alert_time]', '09:00');
    }},
    { mode: 'anytime', extra: async () => {} },
  ]) {
    await page.goto('/post-request?origin=Saillans&destination=Crest');
    await page.waitForSelector('#notify-form');

    // Name + phone come from the saved profile, so those fields are hidden here.
    await page.click(`.btn-mode[data-mode="${mode}"]`);
    await extra(page);

    const [response] = await Promise.all([
      page.waitForResponse(r => r.url().includes('/api/requests') && r.request().method() === 'POST'),
      page.click('button[type=submit]'),
    ]);
    expect(response.status()).toBe(201);
  }
});

// ── 9. My Alerts lists and deletes ───────────────────────────────────────────
test('searcher views and deletes an alert', async ({ page }) => {
  await setProfile(page, SEARCHER);
  await page.goto(BASE);
  const errors = [];
  page.on('console', msg => { if (msg.type() === 'error') errors.push(msg.text()); });

  await page.goto('/my-searches');
  await page.waitForSelector('#my-searches-form');

  // Submit and wait for API response (combined page fetches /requests and /interests in parallel)
  const [response] = await Promise.all([
    page.waitForResponse(r => r.url().includes('/api/requests') && r.request().method() === 'GET'),
    page.click('#my-searches-form button[type=submit]'),
  ]);
  expect(response.status()).toBe(200);
  const alerts = await response.json();
  expect(alerts.length).toBeGreaterThan(0);

  // Wait for rendering — check DOM directly
  await page.waitForFunction(() =>
    document.getElementById('my-alerts-list') &&
    document.getElementById('my-alerts-list').children.length > 0
  , { timeout: 10000 });

  if (errors.length) console.log('JS errors:', errors);

  const count = await page.locator('.card').count();
  expect(count).toBeGreaterThan(0);

  // Delete the first alert
  await page.locator('.btn-delete').first().click();
  // Message is language-dependent; just verify it becomes non-empty
  await expect(page.locator('.delete-msg').first()).not.toBeEmpty({ timeout: 5000 });
});

// ── 10. Return journey defaults to outbound + 2h ──────────────────────────────
test('return journey defaults to outbound + 2 hours', async ({ page }) => {
  await setFr(page);
  await page.goto('/post-ride');
  await page.waitForSelector('input[name=departure_at]');
  await page.fill('input[name=departure_at]', '2030-12-01T09:00');
  await page.click('#btn-return');
  const val = await page.inputValue('input[name=return_departure_at]');
  expect(val).toBe('2030-12-01T11:00');
});

// ── 11. Language switching ────────────────────────────────────────────────────
test('language switcher toggles UI language', async ({ page }) => {
  await page.goto(BASE);
  // Switch to English via localStorage and reload
  await page.evaluate(() => {
    localStorage.setItem('lang', 'en');
  });
  await page.reload();
  await expect(page.locator('button.btn-primary')).toContainText("I'm driving");

  // Switch back to French
  await page.evaluate(() => {
    localStorage.setItem('lang', 'fr');
  });
  await page.reload();
  await expect(page.locator('button.btn-primary')).toContainText('Je conduis');
});

// ── 12. Expired ride hidden from search ───────────────────────────────────────
test('expired ride does not appear in search results', async ({ page }) => {
  await page.goto(BASE);
  const pastISO = new Date(Date.now() - 2 * 3600 * 1000).toISOString();

  const postResp = await page.evaluate(async ({ past, driver }) => {
    const r = await fetch('/api/rides', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        driver_name: driver.name, phone: driver.phone,
        origin: 'Saillans', destination: 'Crest',
        departure_at: past, flexibility: 0,
      }),
    });
    return r.json();
  }, { past: pastISO, driver: DRIVER });

  const expiredId = postResp.ID;
  expect(expiredId).toBeTruthy();

  const rides = await page.evaluate(() =>
    fetch('/api/rides?origin=Saillans&destination=Crest').then(r => r.json())
  );
  expect(rides.find(r => r.ID === expiredId)).toBeUndefined();

  // Cleanup
  await page.evaluate(async ({ id, phone }) => {
    await fetch(`/api/rides/${id}`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone }),
    });
  }, { id: expiredId, phone: DRIVER.phone });
});

// ── 13. URL deep-linking ──────────────────────────────────────────────────────
test('search URL pre-fills origin and destination', async ({ page }) => {
  await page.goto(`${BASE}/search?origin=Saillans&destination=Crest`);
  await expect(page.locator('input[name=origin]')).toHaveValue('Saillans');
  await expect(page.locator('input[name=destination]')).toHaveValue('Crest');
});

test('reloading a search page stays on search', async ({ page }) => {
  await page.goto(`${BASE}/search?origin=Saillans&destination=Crest`);
  await page.reload();
  await expect(page).toHaveURL(/search\?origin=Saillans/);
  await expect(page.locator('input[name=origin]')).toHaveValue('Saillans');
});

// ── 15. Two-column results layout ─────────────────────────────────────────────
test('search shows forward and return columns', async ({ page }) => {
  await page.goto(`${BASE}/search?origin=Saillans&destination=Crest`);
  await page.waitForSelector('.results-col-header');

  const headers = await page.locator('.results-col-header').allTextContents();
  expect(headers.length).toBe(2);
  expect(headers[0]).toMatch(/Saillans.*Crest/);
  expect(headers[1]).toMatch(/Crest.*Saillans/);
});

// ── 16. Notify me button appears in empty column ───────────────────────────────
test('"notify me" appears when a direction has no rides', async ({ page }) => {
  // Saillans→Crest has rides (test 3 posted one), Crest→Saillans is empty
  await page.goto(`${BASE}/search?origin=Saillans&destination=Crest`);
  await page.waitForSelector('.results-col-header');

  // At least one column should have the notify button (likely the return direction)
  await expect(page.locator('.col-notify').first()).toBeVisible();
});

// ── 17. Notify me appears after results too ───────────────────────────────────
test('"notify me" also appears below ride results', async ({ page }) => {
  await page.goto(`${BASE}/search?origin=Saillans&destination=Crest`);
  await page.waitForSelector('.card');

  // The notify button should exist in the forward column that has results
  const notifyBtns = await page.locator('.col-notify').count();
  expect(notifyBtns).toBeGreaterThan(0);
});

// ── 18. Completely unknown route shows both columns empty with notify buttons ──
test('unknown route shows both columns empty with notify buttons', async ({ page }) => {
  await page.goto(`${BASE}/search?origin=NullIsland&destination=Nowhere`);
  await page.waitForSelector('.col-notify');

  const notifyBtns = await page.locator('.col-notify').count();
  expect(notifyBtns).toBe(2); // one per direction
  await expect(page.locator('.col-empty').first()).toBeVisible();
});

// ── 19. Date filter pre-fills notification alert ───────────────────────────────
test('date in search pre-fills the notification alert form', async ({ page }) => {
  await page.goto(`${BASE}/search?origin=Saillans&destination=Crest&departure_at=2030-06-15T09%3A00%3A00.000Z`);
  await page.waitForSelector('.col-notify');

  // Click the notify button
  await page.locator('.col-notify').first().click();
  await page.waitForSelector('#notify-form');

  // The date field should be pre-filled from the departure_at param
  const dateVal = await page.inputValue('input[name=alert_date]');
  expect(dateVal).toBe('2030-06-15');
});

// ── 20. Date+time in search pre-fills both fields in alert form ────────────────
test('date+time in search pre-fills both fields in notification alert', async ({ page }) => {
  await page.goto(`${BASE}/search?origin=Saillans&destination=Crest&departure_at=2030-06-15T09%3A00%3A00.000Z`);
  await page.waitForSelector('.col-notify');

  await page.locator('.col-notify').first().click();
  await page.waitForSelector('#notify-form');

  const dateVal = await page.inputValue('input[name=alert_date]');
  const timeVal = await page.inputValue('input[name=alert_time]');
  expect(dateVal).toBe('2030-06-15');
  // Time is converted to local timezone — verify format not exact value
  expect(timeVal).toMatch(/^\d{2}:\d{2}$/);
});

// ── 21. Date-filtered search hides rides from other dates ─────────────────────
test('search with date filter only shows rides on that date', async ({ page }) => {
  await page.goto(BASE);

  // Post a ride on a specific far-future date (unique enough to not clash)
  const targetDate = '2031-03-10';
  const otherDate  = '2031-03-11';
  const origin = `FilterTest${Date.now()}`;
  const dest   = `FilterDest${Date.now()}`;

  const postRide = async (departure) => page.evaluate(async ({ driver, origin, dest, dep }) => {
    const r = await fetch('/api/rides', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        driver_name: driver.name, phone: driver.phone,
        origin, destination: dest,
        departure_at: dep, flexibility: 0,
      }),
    });
    return r.json();
  }, { driver: DRIVER, origin, dest, dep: departure });

  const r1 = await postRide(`${targetDate}T09:00:00Z`); // on target date
  const r2 = await postRide(`${otherDate}T09:00:00Z`);  // on other date
  expect(r1.ID).toBeTruthy();
  expect(r2.ID).toBeTruthy();

  // Search with search_date = targetDate (date-only, no time)
  const rides = await page.evaluate(async ({ origin, dest, date }) => {
    const r = await fetch(`/api/rides?origin=${encodeURIComponent(origin)}&destination=${encodeURIComponent(dest)}&search_date=${encodeURIComponent(date)}`);
    return r.json();
  }, { origin, dest, date: targetDate });

  const ids = rides.map(r => r.ID);
  expect(ids).toContain(r1.ID);       // target date → should appear
  expect(ids).not.toContain(r2.ID);   // other date  → must be hidden
});

// ── 22. Date+time search excludes rides outside the ±60min window ─────────────
test('date+time search hides rides outside ±60 min window', async ({ page }) => {
  await page.goto(BASE);

  const origin = `TimeFilter${Date.now()}`;
  const dest   = `TimeFilterDest${Date.now()}`;

  // Post a near ride (09:00) and a far ride (15:00) on the same date
  const [near, far] = await page.evaluate(async ({ driver, origin, dest }) => {
    const post = async (dep) => {
      const r = await fetch('/api/rides', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          driver_name: driver.name, phone: driver.phone,
          origin, destination: dest, flexibility: 0,
          departure_at: dep,
        }),
      });
      return (await r.json()).ID;
    };
    return Promise.all([
      post('2031-09-01T09:00:00Z'),   // near — within ±60 min of 09:30
      post('2031-09-01T15:00:00Z'),   // far  — outside ±60 min of 09:30
    ]);
  }, { driver: DRIVER, origin, dest });

  // Search with departure_at = 09:30 on 2031-09-01
  const rides = await page.evaluate(async ({ o, d }) => {
    const r = await fetch(
      `/api/rides?origin=${encodeURIComponent(o)}&destination=${encodeURIComponent(d)}&departure_at=2031-09-01T09%3A30%3A00Z`
    );
    return r.json();
  }, { o: origin, d: dest });

  const ids = rides.map(r => r.ID);
  expect(ids).toContain(near);
  expect(ids).not.toContain(far);
});

// ── 23. Search fields survive page reload ─────────────────────────────────────
test('reload preserves date-only search input', async ({ page }) => {
  await page.goto(`${BASE}/search?origin=Saillans&destination=Crest&search_date=2030-07-15`);
  await page.reload();
  await expect(page.locator('input[name=search_date]')).toHaveValue('2030-07-15');
  await expect(page.locator('input[name=search_time]')).toHaveValue('');
});

test('reload preserves date+time search inputs', async ({ page }) => {
  // 09:30 CEST = 07:30 UTC
  await page.goto(`${BASE}/search?origin=Saillans&destination=Crest&departure_at=2030-07-15T07%3A30%3A00Z`);
  await page.reload();
  await expect(page.locator('input[name=search_date]')).toHaveValue('2030-07-15');
  const timeVal = await page.inputValue('input[name=search_time]');
  expect(timeVal).toMatch(/^\d{2}:\d{2}$/); // non-empty HH:MM (exact value depends on timezone)
});

test('reload preserves time-only search input', async ({ page }) => {
  await page.goto(`${BASE}/search?origin=Saillans&destination=Crest&search_time=09%3A30`);
  await page.reload();
  await expect(page.locator('input[name=search_time]')).toHaveValue('09:30');
  await expect(page.locator('input[name=search_date]')).toHaveValue('');
});

// ── 26. Time-only search filters correctly (timezone-aware) ───────────────────
test('time-only search shows rides within ±60min window', async ({ page }) => {
  await page.goto(BASE);

  const origin = `TZTest${Date.now()}`;
  const dest   = `TZTestDest${Date.now()}`;

  // Post rides at local times relative to browser timezone (Europe/Paris = CEST in summer)
  // near: 09:10 local (within ±60min of 09:30)
  // far:  15:00 local (outside ±60min of 09:30)
  const [nearID, farID] = await page.evaluate(async ({ driver, origin, dest }) => {
    const post = async (localHour, localMin) => {
      const d = new Date(2031, 8, 1, localHour, localMin, 0, 0); // Sep 1 2031, local time
      const r = await fetch('/api/rides', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          driver_name: driver.name, phone: driver.phone,
          origin, destination: dest, flexibility: 0,
          departure_at: d.toISOString(),
        }),
      });
      return (await r.json()).ID;
    };
    return Promise.all([post(9, 10), post(15, 0)]);
  }, { driver: DRIVER, origin, dest });

  expect(nearID).toBeTruthy();
  expect(farID).toBeTruthy();

  // Navigate to search page, fill fields, and submit
  await page.goto(`${BASE}/search?origin=${encodeURIComponent(origin)}&destination=${encodeURIComponent(dest)}`);
  await page.waitForSelector('#search-form');
  await page.fill('input[name=search_time]', '09:30');

  // Submit and wait for both API calls (forward + return) that include search_time
  const [fwdResponse] = await Promise.all([
    page.waitForResponse(r =>
      r.url().includes(`origin=${encodeURIComponent(origin)}`) &&
      r.url().includes('search_time') &&
      r.request().method() === 'GET'
    ),
    page.click('button[type=submit]'),
  ]);

  const fwdRides = await fwdResponse.json();
  const ids = fwdRides.map(r => r.ID);

  expect(ids).toContain(nearID);
  expect(ids).not.toContain(farID);
});

// ── 27. Notification queue — enqueued when ride matches alert ─────────────────
test('ride matching an alert enqueues a notification', async ({ page }) => {
  await page.goto(BASE);

  const origin = `NQTest${Date.now()}`;
  const dest   = `NQDest${Date.now()}`;

  // Post an anytime alert
  const alertResp = await page.evaluate(async ({ searcher, origin, dest }) => {
    const r = await fetch('/api/requests', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ searcher_name: searcher.name, phone: searcher.phone, origin, destination: dest }),
    });
    return r.json();
  }, { searcher: SEARCHER, origin, dest });
  expect(alertResp.ID).toBeTruthy();

  // Driver posts a matching ride
  const rideResp = await page.evaluate(async ({ driver, origin, dest }) => {
    const r = await fetch('/api/rides', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        driver_name: driver.name, phone: driver.phone,
        origin, destination: dest, flexibility: 0,
        departure_at: new Date(Date.now() + 4 * 3600 * 1000).toISOString(),
      }),
    });
    return r.json();
  }, { driver: DRIVER, origin, dest });
  expect(rideResp.ID).toBeTruthy();

  // Searcher fetches their pending notifications
  const notifs = await page.evaluate(async ({ phone }) => {
    const r = await fetch('/api/notifications', { headers: { 'X-Phone': phone } });
    return r.json();
  }, { phone: SEARCHER.phone });

  expect(notifs.length).toBeGreaterThan(0);
  const match = notifs.find(n => n.ride_id === rideResp.ID);
  expect(match).toBeDefined();
  expect(match.driver_name).toBe(DRIVER.name);
});

// ── 28. Notification queue — cleared when ride is deleted ─────────────────────
test('deleting a ride clears its notification queue entries', async ({ page }) => {
  await page.goto(BASE);

  const origin = `NQDel${Date.now()}`;
  const dest   = `NQDelDest${Date.now()}`;

  // Post alert and ride
  await page.evaluate(async ({ searcher, origin, dest }) => {
    await fetch('/api/requests', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ searcher_name: searcher.name, phone: searcher.phone, origin, destination: dest }),
    });
  }, { searcher: SEARCHER, origin, dest });

  const rideID = await page.evaluate(async ({ driver, origin, dest }) => {
    const r = await fetch('/api/rides', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        driver_name: driver.name, phone: driver.phone,
        origin, destination: dest, flexibility: 0,
        departure_at: new Date(Date.now() + 4 * 3600 * 1000).toISOString(),
      }),
    });
    return (await r.json()).ID;
  }, { driver: DRIVER, origin, dest });

  // Confirm notification enqueued
  const before = await page.evaluate(async ({ phone }) => {
    const r = await fetch('/api/notifications', { headers: { 'X-Phone': phone } });
    return r.json();
  }, { phone: SEARCHER.phone });
  expect(before.some(n => n.ride_id === rideID)).toBe(true);

  // Delete the ride
  await page.evaluate(async ({ id, phone }) => {
    await fetch(`/api/rides/${id}`, {
      method: 'DELETE', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone }),
    });
  }, { id: rideID, phone: DRIVER.phone });

  // Notification should be gone
  const after = await page.evaluate(async ({ phone }) => {
    const r = await fetch('/api/notifications', { headers: { 'X-Phone': phone } });
    return r.json();
  }, { phone: SEARCHER.phone });
  expect(after.some(n => n.ride_id === rideID)).toBe(false);
});

// ── 30. Driver can notify a matching searcher (Prévenir button) ───────────────
test('driver sees Prévenir button next to matching searcher and can click it', async ({ page }) => {
  await page.goto(BASE);
  await setFr(page);

  const origin = `PingTest${Date.now()}`;
  const dest   = `PingDest${Date.now()}`;

  // Searcher (Bob) posts an anytime alert
  await page.evaluate(async ({ searcher, origin, dest }) => {
    await fetch('/api/requests', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ searcher_name: searcher.name, phone: searcher.phone, origin, destination: dest }),
    });
  }, { searcher: SEARCHER, origin, dest });

  // Driver (Alice) posts a matching ride
  await page.context().grantPermissions(['notifications'], { origin: BASE });
  await setProfile(page, DRIVER);
  await page.evaluate(async ({ driver, origin, dest }) => {
    const r = await fetch('/api/rides', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        driver_name: driver.name, phone: driver.phone,
        origin, destination: dest, flexibility: 0,
        departure_at: new Date(Date.now() + 4 * 3600 * 1000).toISOString(),
      }),
    });
    return (await r.json()).ID;
  }, { driver: DRIVER, origin, dest });

  // Open My Rides — should show the seeker row with Prévenir button.
  // Scope to THIS ride's card (unique route) so pre-existing rides/seekers in the
  // shared dev DB don't shadow the assertions.
  await page.goto('/my-rides');
  await page.waitForSelector('#my-rides-form');
  await page.click('#my-rides-form button[type=submit]');
  const rideCard = page.locator('.card', { hasText: `${origin} → ${dest}` });
  await rideCard.locator('.btn-ping-searcher').first().waitFor({ timeout: 8000 });

  // Verify no phone number is shown in the seeker row
  const seekerText = await rideCard.locator('.seeker-row').first().innerText();
  expect(seekerText).toContain(SEARCHER.name);
  expect(seekerText).not.toContain(SEARCHER.phone);

  // Click Prévenir — should disable the button (API call succeeds)
  const [pingResp] = await Promise.all([
    page.waitForResponse(r => r.url().includes('/ping') && r.request().method() === 'POST'),
    rideCard.locator('.btn-ping-searcher').first().click(),
  ]);
  expect(pingResp.status()).toBe(204);
  await expect(rideCard.locator('.btn-ping-searcher').first()).toBeDisabled({ timeout: 3000 });

  // After pinging, Bob can see Alice's phone via the interest contact endpoint
  const contactResp = await page.evaluate(async ({ searcherPhone }) => {
    const interests = await fetch('/api/interests', {
      headers: { 'X-Phone': searcherPhone }
    }).then(r => r.json());
    const accepted = (interests || []).find(i => i.status === 'accepted');
    if (!accepted) return { found: false };
    const contact = await fetch('/api/interests/' + accepted.id + '/contact', {
      headers: { 'X-Phone': searcherPhone }
    }).then(r => r.json());
    return { found: true, role: contact.role, hasPhone: !!contact.phone };
  }, { searcherPhone: SEARCHER.phone });

  expect(contactResp.found).toBe(true);
  expect(contactResp.role).toBe('driver');
  expect(contactResp.hasPhone).toBe(true);

  const driversPhone = await page.evaluate(async ({ searcherPhone }) => {
    const interests = await fetch('/api/interests', {
      headers: { 'X-Phone': searcherPhone }
    }).then(r => r.json());
    const ds = (interests || []).find(i => i.status === 'driver_shared');
    if (!ds) return null;
    const contact = await fetch('/api/interests/' + ds.id + '/contact', {
      headers: { 'X-Phone': searcherPhone }
    }).then(r => r.json());
    return contact.phone || null;
  }, { searcherPhone: SEARCHER.phone });
  expect(driversPhone).toBeTruthy();

  const driverGetsSearcherPhone = await page.evaluate(async ({ driverPhone, searcherPhone }) => {
    const interests = await fetch('/api/interests', {
      headers: { 'X-Phone': searcherPhone }
    }).then(r => r.json());
    const ds = (interests || []).find(i => i.status === 'driver_shared');
    if (!ds) return 403;
    const r = await fetch('/api/interests/' + ds.id + '/contact', {
      headers: { 'X-Phone': driverPhone }
    });
    return r.status;
  }, { driverPhone: DRIVER.phone, searcherPhone: SEARCHER.phone });
  expect(driverGetsSearcherPhone).toBe(403);
});

// ── 31. Me page — saves profile and pre-fills other forms ─────────────────────
test('Me page saves name and phone to localStorage', async ({ page }) => {
  await page.goto(BASE);
  await setFr(page);
  await page.reload();

  await page.click('#btn-me');
  await expect(page).toHaveURL(/\/me/);
  await expect(page.locator('h2')).toContainText(/Mon profil|My profile|profil/i);

  await page.fill('input[name=name]', 'Marie');
  await page.fill('input[name=phone]', '0644000001');
  await page.click('button[type=submit]');

  await expect(page.locator('#me-saved')).toBeVisible({ timeout: 3000 });

  // stores write RAW strings (legacy-compatible), so read them directly.
  const stored = await page.evaluate(() => ({
    name:  localStorage.getItem('user_name'),
    phone: localStorage.getItem('user_phone'),
  }));
  expect(stored.name).toBe('Marie');
  expect(stored.phone).toBe('0644000001');
});

test('Me page pre-fills from existing localStorage profile', async ({ page }) => {
  // Seed the persisted-store keys (RAW strings, as the legacy app + existing
  // users store them) before navigation so the store hydrates from them.
  await page.addInitScript(() => {
    localStorage.setItem('lang', 'fr');
    localStorage.setItem('user_name', 'Jean');
    localStorage.setItem('user_phone', '0655000002');
  });

  await page.goto('/me');
  await page.waitForSelector('#me-form');

  await expect(page.locator('input[name=name]')).toHaveValue('Jean');
  await expect(page.locator('input[name=phone]')).toHaveValue('0655000002');
});

test('Me page values pre-fill the post-ride form', async ({ page }) => {
  await page.context().grantPermissions(['notifications'], { origin: BASE });
  await setFr(page);

  await page.goto('/me');
  await page.waitForSelector('#me-form');
  await page.fill('input[name=name]', 'Sophie');
  await page.fill('input[name=phone]', '0666000003');
  await page.click('button[type=submit]');
  await page.waitForSelector('#me-saved:not([style*="none"])');

  await page.click('#back');
  await page.waitForSelector('button.btn-primary');
  await page.click('button.btn-primary');

  // With a saved profile the name/phone inputs are hidden behind a summary.
  await expect(page.locator('.profile-summary')).toContainText('Sophie');
  await expect(page.locator('.profile-summary')).toContainText('0666000003');
  await expect(page.locator('input[name=driver_name]')).toHaveCount(0);

  // "Change" reveals the inputs, pre-filled from the profile.
  await page.click('.btn-edit-contact');
  await expect(page.locator('input[name=driver_name]')).toHaveValue('Sophie');
  await expect(page.locator('input[name=phone]')).toHaveValue('0666000003');
});
