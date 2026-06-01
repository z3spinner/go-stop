// @ts-check
const { test, expect } = require('@playwright/test');

const DRIVER   = { name: 'Alice', phone: '0611000001' };
const SEARCHER = { name: 'Bob',   phone: '0622000002' };
const BASE = 'http://localhost:8080';

// ── Helpers ───────────────────────────────────────────────────────────────────

async function setProfile(page, user) {
  await page.evaluate(({ name, phone }) => {
    localStorage.setItem('user_name', name);
    localStorage.setItem('user_phone', phone);
    localStorage.setItem('lang', 'fr');
  }, user);
}

async function setFr(page) {
  await page.evaluate(() => localStorage.setItem('lang', 'fr'));
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

  await expect(page.locator('.card').first()).toContainText(DRIVER.name);
  await expect(page.locator('.card').first()).not.toContainText(DRIVER.phone);
  await expect(page.locator('.btn-interest').first()).toBeVisible();
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

  // Button becomes disabled with pending text, confirmation message appears
  await expect(firstCard.locator('.btn-interest')).toBeDisabled({ timeout: 5000 });
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
  await setProfile(page, DRIVER);
  await page.reload();

  // Navigate to My Rides via button
  await page.evaluate(() => renderMyRides());
  await page.waitForSelector('.card');

  // Wait for interests to load async
  await page.waitForTimeout(1000);
  const pageText = await page.locator('#app').innerText();
  expect(pageText).toContain(SEARCHER.name);
});

// ── 8. Alert creation — all 4 modes ──────────────────────────────────────────
test('searcher creates alerts in all four modes', async ({ page }) => {
  await page.goto(BASE);
  await setProfile(page, SEARCHER);

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
    await page.evaluate(() => renderNotifyRoute('Saillans', 'Crest'));
    await page.waitForSelector('#notify-form');

    await page.fill('input[name=searcher_name]', SEARCHER.name);
    await page.fill('input[name=phone]', SEARCHER.phone);
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
  await page.goto(BASE);
  await setProfile(page, SEARCHER);
  const errors = [];
  page.on('console', msg => { if (msg.type() === 'error') errors.push(msg.text()); });

  await page.evaluate(() => { localStorage.setItem('lang', 'fr'); renderMyAlerts(); });
  await page.waitForSelector('#my-alerts-form');

  // Submit and wait for API response
  const [response] = await Promise.all([
    page.waitForResponse(r => r.url().includes('/api/requests') && r.request().method() === 'GET'),
    page.click('#my-alerts-form button[type=submit]'),
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
  await page.goto(BASE);
  await setFr(page);
  await page.reload();
  await page.evaluate(() => renderPostRide());
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

  // Search with departure_at = targetDate (midnight UTC → same date)
  const deptParam = encodeURIComponent(`${targetDate}T00:00:00.000Z`);
  const rides = await page.evaluate(async ({ origin, dest, dept }) => {
    const r = await fetch(`/api/rides?origin=${encodeURIComponent(origin)}&destination=${encodeURIComponent(dest)}&departure_at=${dept}`);
    return r.json();
  }, { origin, dest, dept: deptParam });

  const ids = rides.map(r => r.ID);
  expect(ids).toContain(r1.ID);       // target date → should appear
  expect(ids).not.toContain(r2.ID);   // other date  → must be hidden
});
