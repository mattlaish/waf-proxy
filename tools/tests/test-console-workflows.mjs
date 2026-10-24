import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { after, before, test } from 'node:test';

// Uses a locally provided Playwright installation; never downloads a browser.
// All HTTP requests are intercepted. No running WAF, secrets or live writes.
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.WAF_PLAYWRIGHT_MODULE || 'playwright');
const root = new URL('../../', import.meta.url);
const source = readFileSync(new URL('static/admin.html', root), 'utf8');
const theme = readFileSync(new URL('static/theme.css', root), 'utf8');
const sample = JSON.parse(readFileSync(new URL('config.sample.json', root), 'utf8'));
const clone = x => JSON.parse(JSON.stringify(x));
let browser;
before(async () => {
  browser = await chromium.launch({
    headless: true,
    ...(process.env.WAF_BROWSER_EXECUTABLE ? { executablePath: process.env.WAF_BROWSER_EXECUTABLE } : {}),
  });
});
after(async () => { await browser?.close(); });

async function session({ failures = {}, holds = {}, role = 'admin', pagePolicies = [] } = {}) {
  const active = { ...clone(sample), sites: [{
    name: 'shop', listen: ':8080', hostnames: ['shop.test'], pool: 'default-pool',
    policy: sample.policies[0].name, preserve_host: false, ai_mode: 'off',
    page_policies: clone(pagePolicies),
  }] };
  let draft = null, running = false, scanPolls = 0;
  const calls = [], errors = [];
  const context = await browser.newContext();
  const page = await context.newPage();
  page.on('pageerror', error => errors.push(error.message));
  await context.addInitScript(() => sessionStorage.setItem('waf_token', 'test-session-only'));
  const siteMap = () => [{
    site: 'shop', nodes: 2,
    tree: { name: '/', full: '/', is_dir: true, children: [{ name: 'login', full: '/login', is_dir: false, children: [] }] },
    crawl: { running: running && ++scanPolls < 3, pages: running ? 2 : 0, forms: running ? 1 : 0, fields: running ? 2 : 0, finished: scanPolls >= 3 ? '2026-10-18T12:00:00Z' : '' },
  }];
  await context.route('**/*', async route => {
    const req = route.request(), url = new URL(req.url()), path = url.pathname + url.search;
    const method = req.method(), body = req.postDataJSON?.();
    calls.push({ path, method, body });
    const send = (data, status = 200, contentType = 'application/json') => route.fulfill({ status, contentType, body: contentType === 'application/json' ? JSON.stringify(data) : data });
    if (url.origin !== 'http://waf-console.test') return route.abort();
    if (path === '/') return send(source.replace('  resume();', '  window.testUX={configChanges,changeTable,attentionItems,validListen,validBackend,buildWizardSite,readPagePolicy,pageContext,pageErrors,api,loadIndependent,scanFieldSnapshot,state:uxState};\n  resume();'), 200, 'text/html');
    if (path === '/theme.css') return send(theme, 200, 'text/css');
    if (holds[path]) await holds[path];
    if (failures[path]) return send({ error: 'fixture unavailable' }, failures[path]);
    if (path === '/api/whoami') return send({ user: 'operator', role, can_edit_config: ['admin', 'operator'].includes(role), can_review: role !== 'viewer', can_manage_users: role === 'admin' });
    if (path.startsWith('/api/config')) {
      if (method === 'DELETE') {
        if (role !== 'admin' && role !== 'operator') return send({ error: 'forbidden' }, 403);
        if (url.searchParams.get('draft') !== '1') return send({ error: 'draft=1 required' }, 400);
        draft = null;return send({ discarded: true });
      }
      if (method === 'PUT') {
        if (url.searchParams.has('draft')) draft = clone(body);
        else { Object.assign(active, clone(body)); draft = null; }
        return send({ config: clone(body), draft: url.searchParams.has('draft'), applied: !url.searchParams.has('draft'), apply_ready: true });
      }
      return send(clone(url.searchParams.has('draft') ? (draft || active) : active));
    }
    if (path === '/api/interfaces') return send([
      { name: 'ens33', up: true, plane: 'management', addresses: ['192.168.1.66/21'] },
      { name: 'ens37', up: true, plane: 'data', addresses: ['192.168.1.71/21'] },
    ]);
    if (path === '/api/status') return send({ sites: active.sites.map(s => ({ ...s, engine: s.engine_mode || active.engine_mode })), pools: 2, nodes: 1, uptime: '1m', match_count: 1 });
    if (path === '/api/metrics') return send({ history: [] });
    if (path === '/api/pools') return send([{ name: 'default-pool', members: [{ node: 'app1', target: '127.0.0.1:8080', healthy: false }] }]);
    if (path.startsWith('/api/notifications')) return send({ items: [], unread: 0 });
    if (path.startsWith('/api/matches')) return send([{ site: 'shop', uri: '/login', rule_id: 942100, severity: 'WARNING', time: '12:00:00', msg: 'SQL injection pattern', phase: 2 }]);
    if (path.startsWith('/api/access')) return send([{ site: 'shop', path: '/login', method: 'POST', status: 403 }]);
    if (path === '/api/security/operations') return send([{ id: 'op-login', site: 'shop', method: 'POST', path: '/login', samples: 2 }]);
    if (path === '/api/sitemap') return send(siteMap());
    if (path === '/api/crawl') { running = true; scanPolls = 0; return send({ ok: true }); }
    if (url.pathname === '/api/sitemap/clear') return send({ ok: method === 'POST' }, method === 'POST' ? 200 : 405);
    if (path.startsWith('/api/forms')) {
      if (!url.searchParams.get('path')) return send({ error: 'path required' }, 400);
      return send(url.searchParams.get('path') === '/login' ? [
        { name: 'username', type: 'text', method: 'POST', action: '/login', discovery_source: 'passive' },
        ...running ? [{ name: 'password', type: 'password', method: 'POST', action: '/login', discovery_source: 'both' }] : [],
      ] : []);
    }
    if (path.startsWith('/api/pagepolicy?')) return send(clone(active.sites[0].page_policies));
    if (path === '/api/pagepolicy/upsert') {
      const rows = active.sites[0].page_policies, i = rows.findIndex(p => p.path === body.policy.path && (p.match || 'prefix') === (body.policy.match || 'prefix'));
      if (i < 0) rows.push(clone(body.policy)); else rows[i] = clone(body.policy);
      return send({ ok: true });
    }
    if (path === '/api/pagepolicy/delete') {
      active.sites[0].page_policies = active.sites[0].page_policies.filter(p => p.path !== body.path || (body.match && (p.match || 'prefix') !== body.match));
      return send({ ok: true });
    }
    return send([]);
  });
  await page.goto('http://waf-console.test/');
  await page.locator('#ux_change_state').filter({ hasText: 'matches active settings' }).waitFor();
  return { page, context, calls, errors, active, failures, close: () => context.close() };
}
const tab = (page, name) => page.locator('.tab[data-view="' + name + '"]').click();
const choice = (page, value) => page.locator('#ux_dialog [data-dialog-choice="' + value + '"]').click();
const submit = page => page.locator('#ux_form button[type="submit"]').click();
const writes = calls => calls.filter(c => !['GET', 'HEAD'].includes(c.method));

test('browser startup, dashboard signals and responsive layout', async () => {
  const s = await session();
  try {
    await s.page.locator('#ux_attention').filter({ hasText: 'Application server unavailable' }).waitFor();
    assert.match(await s.page.locator('#ux_attention').innerText(), /not a WAF-block rate/);
    await s.page.setViewportSize({ width: 390, height: 844 });
    assert.equal(await s.page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('unsaved navigation cancel, draft save and apply cancellation do not activate', async () => {
  const s = await session();
  try {
    await tab(s.page, 'sites');
    await s.page.locator('#site_list [data-sel]').first().click();
    await s.page.locator('#sites input[data-f="name"]').first().fill('renamed');
    await tab(s.page, 'pools');
    await choice(s.page, 'cancel');
    assert.equal(await s.page.locator('#sites input[data-f="name"]').first().inputValue(), 'renamed');
    assert.equal(writes(s.calls).length, 0);
    await s.page.locator('#ux_save').click();
    await s.page.locator('#ux_change_state').filter({ hasText: 'Draft saved' }).waitFor();
    assert.equal(writes(s.calls).at(-1).path, '/api/config?draft=1');
    await s.page.locator('#ux_review').click();
    await choice(s.page, 'cancel');
    assert.equal(writes(s.calls).filter(c => c.path === '/api/config').length, 0);
    await s.page.locator('#ux_review').click();
    await choice(s.page, 'apply');
    await s.page.locator('#ux_change_state').filter({ hasText: 'matches active settings' }).waitFor();
    assert.equal(writes(s.calls).filter(c => c.path === '/api/config').length, 1);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('partial API failures retain endpoint selectors and show retry/permission states', async () => {
  const s = await session({ failures: { '/api/security/graphql/policies': 503, '/api/security/sequence/model': 403 } });
  try {
    await tab(s.page, 'api');
    await s.page.locator('#api_ops_body').filter({ hasText: '/login' }).waitFor();
    assert.match(await s.page.locator('#api_obj_op').textContent(), /shop.*POST.*login/);
    assert.match(await s.page.locator('#ux_errors').innerText(), /does not have access/);
    assert.equal(await s.page.locator('[data-resource-retry="/api/security/sequence/model"]').count(), 0);
    delete s.failures['/api/security/graphql/policies'];
    await s.page.locator('[data-resource-retry="/api/security/graphql/policies"]').click();
    await s.page.waitForFunction(() => !window.testUX.state.resources.get('/api/security/graphql/policies')?.error);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('guided setup validates inline, excludes management NIC and saves only a draft', async () => {
  const s = await session();
  try {
    await s.page.locator('#ux_wizard').click();
    await submit(s.page);
    assert.equal(await s.page.locator('#ux_input_name').getAttribute('aria-invalid'), 'true');
    await s.page.locator('#ux_input_name').fill('portal');
    await s.page.locator('#ux_input_domains').fill('portal.test');
    await s.page.locator('#ux_input_listen').fill('192.168.1.80:443');
    await submit(s.page);
    await s.page.locator('#ux_input_pool').selectOption('default-pool');
    await submit(s.page);
    assert.doesNotMatch(await s.page.locator('#ux_input_iface').innerText(), /ens33/);
    await s.page.locator('#ux_input_cert').fill('/etc/waf/portal.crt');
    await s.page.locator('#ux_input_key').fill('/etc/waf/portal.key');
    await submit(s.page);
    await s.page.locator('#ux_input_policy').selectOption(sample.policies[0].name);
    await submit(s.page);
    await choice(s.page, 'save');
    await s.page.locator('#ux_change_state').filter({ hasText: 'Draft saved' }).waitFor();
    const saved = writes(s.calls).find(c => c.path === '/api/config?draft=1');
    assert.equal(saved.body.sites.find(x => x.name === 'portal').engine_mode, 'DetectionOnly');
    assert.equal(writes(s.calls).filter(c => c.path === '/api/config').length, 0);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('event field-only exception review, active apply and session undo', async () => {
  const s = await session();
  try {
    await tab(s.page, 'logs');
    await s.page.locator('[data-event]').first().click();
    await choice(s.page, 'field');
    await s.page.locator('#ux_input_name').fill('password');
    await submit(s.page);
    await s.page.locator('#pp:not([hidden])').waitFor();
    await s.page.waitForFunction(() => document.querySelector('#pp_fields [data-ff="name"][value="password"]'));
    await s.page.locator('#pp_save').click();
    await choice(s.page, 'apply');
    await s.page.locator('#pp[hidden]').waitFor({ state: 'attached' });
    const saved = writes(s.calls).find(c => c.path === '/api/pagepolicy/upsert').body.policy;
    assert.deepEqual(saved.exclude_rule_ids, []);
    assert.equal(saved.match, 'exact');
    assert.deepEqual(saved.fields.find(f => f.name === 'password').exclude_rule_ids, [942100]);
    assert.deepEqual(saved.fields.map(f => f.name), ['password']);
    assert.equal(saved.fields[0].profile, '');
    assert.equal(saved.fields.some(f => 'discovery_source' in f), false);
    await s.page.locator('[data-event]').first().click();
    await choice(s.page, 'page');
    await s.page.locator('#ux_page_undo').click();
    await choice(s.page, 'apply');
    await s.page.locator('#pp[hidden]').waitFor({ state: 'attached' });
    assert.equal(s.active.sites[0].page_policies.length, 0);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('page close protects unsaved edits and preserves hidden target exceptions', async () => {
  const original = { path: '/login', match: 'exact', exclude_targets: [{ rule_id: 942100, target: 'ARGS_POST:token' }], fields: [], source: 'manual' };
  const s = await session({ pagePolicies: [original] });
  try {
    await tab(s.page, 'logs');await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');
    await s.page.locator('#pp_note').fill('keep this note');
    await s.page.locator('#ppclose').click();await choice(s.page, 'cancel');
    assert.equal(await s.page.locator('#pp_note').inputValue(), 'keep this note');
    assert.deepEqual(await s.page.evaluate(() => window.testUX.readPagePolicy().exclude_targets), original.exclude_targets);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('scan polls to completion and offers new fields without missing path requests', async () => {
  const s = await session();
  try {
    await tab(s.page, 'map');
    await s.page.locator('.crawl').click();
    await s.page.locator('#ux_scan_state button').waitFor({ timeout: 15000 });
    assert.match(await s.page.locator('#ux_scan_state').innerText(), /1 new field entries/);
    assert.equal(s.calls.some(c => c.path.startsWith('/api/forms?') && !new URL(c.path, 'http://test').searchParams.get('path')), false);
    await s.page.locator('#ux_scan_state button').click();
    assert.match(await s.page.locator('#ux_dialog_body').innerText(), /password/);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('nested secrets are redacted; configuration conflict blocks stale apply', async () => {
  const s = await session();
  try {
    const preview = await s.page.evaluate(() => {
      const ux = window.testUX;
      return ux.changeTable(ux.configChanges({}, { ai: { api_key: 'DO-NOT-DISPLAY', nested: { password: 'ALSO-HIDDEN' } } }));
    });
    assert.doesNotMatch(preview, /DO-NOT-DISPLAY|ALSO-HIDDEN/);
    await tab(s.page, 'sites');
    await s.page.locator('#site_list [data-sel]').first().click();
    await s.page.locator('#sites input[data-f="name"]').first().fill('renamed');
    s.active.engine_mode = 'On';
    await s.page.locator('#ux_review').click();await choice(s.page, 'apply');
    await s.page.locator('#toast').filter({ hasText: 'Active settings changed' }).waitFor();
    assert.equal(writes(s.calls).length, 0);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('modal keyboard focus is contained, Escape cancels and focus returns', async () => {
  const s = await session();
  try {
    await s.page.locator('#ux_wizard').click();
    await s.page.keyboard.press('Shift+Tab');
    assert.equal(await s.page.evaluate(() => document.querySelector('#ux_dialog').contains(document.activeElement)), true);
    await s.page.keyboard.press('Escape');
    assert.equal(await s.page.locator('#ux_dialog').isVisible(), false);
    assert.equal(await s.page.evaluate(() => document.activeElement.id), 'ux_wizard');
    assert.equal(writes(s.calls).length, 0);
  } finally { await s.close(); }
});

test('viewer controls cannot mutate configuration', async () => {
  const s = await session({ role: 'viewer' });
  try {
    assert.equal(await s.page.locator('#ux_wizard').isDisabled(), true);
    assert.equal(await s.page.locator('#ux_save').isDisabled(), true);
    const error = await s.page.evaluate(async () => { try { await window.testUX.api('/api/config', { method: 'PUT', body: '{}' }); } catch (e) { return e.status; } });
    assert.equal(error, 403);
    assert.equal(writes(s.calls).length, 0);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('a slow API section does not delay other section results', async () => {
  let release;
  const held = new Promise(resolve => { release = resolve; });
  const s = await session({ holds: { '/api/security/graphql/policies': held } });
  try {
    await tab(s.page, 'api');
    await s.page.locator('#api_ops_body').filter({ hasText: '/login' }).waitFor();
    assert.equal(await s.page.evaluate(() => window.testUX.state.resources.get('/api/security/graphql/policies').loading), true);
    assert.match(await s.page.locator('#api_obj_op').textContent(), /POST.*login/);
    release();
    await s.page.waitForFunction(() => !window.testUX.state.resources.get('/api/security/graphql/policies').loading);
    assert.deepEqual(s.errors, []);
  } finally { release();await s.close(); }
});

test('page policy scope selection and delete preserve the other scope on the same URL', async () => {
  const prefix = { path: '/login', match: 'prefix', mode: 'DetectionOnly', fields: [], source: 'manual' };
  const exact = { path: '/login', match: 'exact', mode: 'On', fields: [], source: 'manual' };
  const s = await session({ pagePolicies: [prefix, exact] });
  try {
    await tab(s.page, 'logs');await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');
    await s.page.locator('#ux_input_match').selectOption('exact');await submit(s.page);
    await s.page.locator('#pp:not([hidden])').waitFor();
    assert.equal(await s.page.locator('#pp_mode').inputValue(), 'On');
    await s.page.locator('#pp_delete').click();await choice(s.page, 'apply');
    await s.page.locator('#pp[hidden]').waitFor({ state: 'attached' });
    assert.deepEqual(s.active.sites[0].page_policies, [prefix]);
    assert.equal(writes(s.calls).find(c => c.path === '/api/pagepolicy/delete').body.match, 'exact');
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('a pending full configuration draft cannot overwrite a later page action', async () => {
  const s = await session();
  try {
    await tab(s.page, 'sites');await s.page.locator('#site_list [data-sel]').first().click();
    await s.page.locator('#sites input[data-f="listen"]').fill(':8081');await s.page.locator('#ux_save').click();
    await s.page.locator('#ux_change_state').filter({ hasText: 'Draft saved' }).waitFor();
    await tab(s.page, 'logs');await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');
    await s.page.locator('#pp_note').fill('page edit');await s.page.locator('#pp_save').click();
    await s.page.locator('#ux_page_errors').filter({ hasText: 'full configuration draft first' }).waitFor();
    assert.equal(writes(s.calls).filter(c => c.path.startsWith('/api/pagepolicy')).length, 0);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('numeric configuration errors remain inline and prevent an invalid draft request', async () => {
  const s = await session();
  try {
    await tab(s.page, 'policies');
    await s.page.locator('#l7_requests').fill('-1');
    await s.page.locator('#ux_save').click();
    await s.page.locator('#ux_config_errors').filter({ hasText: 'allowed range' }).waitFor();
    assert.equal(await s.page.locator('#l7_requests').getAttribute('aria-invalid'), 'true');
    assert.equal(writes(s.calls).length, 0);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('clearing discovery requires confirmation and sends the registered POST method', async () => {
  const s = await session({ pagePolicies: [{ path: '/login', match: 'exact', mode: 'On' }] });
  try {
    await tab(s.page, 'map');await s.page.locator('.clear').click();await choice(s.page, 'cancel');
    assert.equal(writes(s.calls).length, 0);
    await s.page.locator('.clear').click();await choice(s.page, 'clear');
    await s.page.waitForFunction(() => document.querySelector('#toast').textContent.includes('Discovered pages cleared'));
    assert.equal(writes(s.calls).at(-1).path, '/api/sitemap/clear?site=shop');
    assert.equal(writes(s.calls).at(-1).method, 'POST');
    assert.equal(s.active.sites[0].page_policies[0].mode, 'On');
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('a single different form action opens its target and carries discovery candidates', async () => {
  const s = await session();
  try {
    await s.page.route('**/api/forms?**', route => {
      const path = new URL(route.request().url()).searchParams.get('path');
      return route.fulfill({ json: path === '/login' ? [{ name: 'username', type: 'text', method: 'POST', action: '/authenticate', discovery_source: 'crawled' }] : [] });
    });
    await tab(s.page, 'logs');await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');
    await s.page.locator('#pp_formaction option[value="/authenticate"]').waitFor({ state: 'attached' });
    assert.equal(await s.page.locator('#pp_formaction option').count(), 1);
    assert.equal(await s.page.locator('#pp_fields .fpolicy').count(), 0);
    assert.equal(await s.page.locator('[data-add-discovered]').first().isDisabled(), true);
    await s.page.locator('#pp_open_form').click();
    await s.page.waitForFunction(() => document.querySelector('#pp_path').value === '/authenticate' && document.querySelector('[data-add-discovered]'));
    assert.equal(await s.page.locator('#pp_fields .fpolicy').count(), 0);
    await s.page.locator('[data-add-discovered]').first().click();
    assert.equal(await s.page.locator('#pp_fields [data-ff="name"]').inputValue(), 'username');
    await s.page.locator('#pp_save').click();await choice(s.page, 'apply');
    await s.page.locator('#pp[hidden]').waitFor({ state: 'attached' });
    const policy = writes(s.calls).find(c => c.path === '/api/pagepolicy/upsert').body.policy;
    assert.equal(policy.path, '/authenticate');
    assert.deepEqual(policy.fields.map(f => f.name), ['username']);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('discovery candidates never enable rules until the operator adds them', async () => {
  const s = await session();
  try {
    await tab(s.page, 'logs');await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');
    await s.page.locator('[data-add-discovered]').first().waitFor();
    assert.equal(await s.page.locator('#pp_fields .fpolicy').count(), 0);
    await s.page.locator('#pp_note').fill('note only');
    await s.page.locator('#pp_save').click();await choice(s.page, 'apply');
    await s.page.locator('#pp[hidden]').waitFor({ state: 'attached' });
    const policy = writes(s.calls).find(c => c.path === '/api/pagepolicy/upsert').body.policy;
    assert.deepEqual(policy.fields, []);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('field-only exceptions preserve existing rules and never add discovered siblings', async () => {
  const username = { name: 'username', source: 'ARGS_POST', profile: 'email', allow_pattern: '', required: false, min_length: 0, max_length: 100, exclude_rule_ids: [] };
  const s = await session({ pagePolicies: [{ path: '/login', match: 'exact', mode: 'On', methods: ['POST'], fields: [username], source: 'manual' }] });
  try {
    await tab(s.page, 'logs');await s.page.locator('[data-event]').first().click();await choice(s.page, 'field');
    await s.page.locator('#ux_input_name').fill('password');await submit(s.page);
    await s.page.waitForFunction(() => document.querySelector('#pp_fields [data-ff="name"][value="password"]'));
    await s.page.locator('#pp_save').click();await choice(s.page, 'apply');
    await s.page.locator('#pp[hidden]').waitFor({ state: 'attached' });
    const policy = writes(s.calls).find(c => c.path === '/api/pagepolicy/upsert').body.policy;
    assert.deepEqual(policy.fields.find(f => f.name === 'username'), username);
    assert.deepEqual(policy.fields.find(f => f.name === 'password').exclude_rule_ids, [942100]);
    assert.equal(policy.fields.length, 2);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('wizard Back retains the unfinished current step without requiring validation', async () => {
  const s = await session();
  try {
    await s.page.locator('#ux_wizard').click();
    await s.page.locator('#ux_input_name').fill('portal');await s.page.locator('#ux_input_domains').fill('portal.test');await submit(s.page);
    await s.page.locator('#ux_input_backend').fill('http://192.0.2.10:8080');
    await s.page.locator('[data-form-back]').click();await submit(s.page);
    assert.equal(await s.page.locator('#ux_input_backend').inputValue(), 'http://192.0.2.10:8080');
    await s.page.locator('#ux_input_pool').selectOption('default-pool');await submit(s.page);
    await s.page.locator('#ux_input_cert').fill('/etc/waf/portal.crt');
    await s.page.locator('#ux_input_prefix').fill('999');
    await s.page.locator('[data-form-back]').click();await submit(s.page);
    assert.equal(await s.page.locator('#ux_input_cert').inputValue(), '/etc/waf/portal.crt');
    assert.equal(await s.page.locator('#ux_input_prefix').inputValue(), '999');
    await submit(s.page);
    assert.equal(await s.page.locator('#ux_input_prefix').getAttribute('aria-invalid'), 'true');
    assert.equal(writes(s.calls).length, 0);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('a deleted prefix policy remains undoable from a new exact-page editor', async () => {
  const original = { path: '/login', match: 'prefix', mode: 'DetectionOnly', fields: [], source: 'manual' };
  const s = await session({ pagePolicies: [original] });
  try {
    await tab(s.page, 'logs');await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');
    await s.page.locator('#pp_delete').click();await choice(s.page, 'apply');
    await s.page.locator('#pp[hidden]').waitFor({ state: 'attached' });
    await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');
    await s.page.locator('#pp:not([hidden])').waitFor();
    assert.equal(await s.page.locator('#pp_match').inputValue(), 'exact');
    await s.page.locator('#ux_page_undo').click();await choice(s.page, 'apply');
    await s.page.locator('#pp[hidden]').waitFor({ state: 'attached' });
    assert.deepEqual(s.active.sites[0].page_policies, [original]);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('view active settings and continue saved draft are read-only and distinct', async () => {
  const s = await session();
  try {
    await tab(s.page, 'sites');await s.page.locator('#site_list [data-sel]').first().click();
    await s.page.locator('#sites input[data-f="listen"]').fill(':8081');await s.page.locator('#ux_save').click();
    await s.page.locator('#ux_change_state').filter({ hasText: 'Draft saved' }).waitFor();
    const savedWrites = writes(s.calls).length;
    await s.page.getByRole('button', { name: 'Load active settings', exact: true }).click();await choice(s.page, 'reload');
    await s.page.locator('#ux_change_state').filter({ hasText: 'matches active settings' }).waitFor();
    await s.page.locator('#ux_resume_draft').click();
    await s.page.locator('#ux_change_state').filter({ hasText: 'Draft saved' }).waitFor();
    assert.equal(writes(s.calls).length, savedWrites);
    assert.equal(s.active.sites[0].listen, ':8080');
    assert.equal(await s.page.locator('#sites input[data-f="listen"]').inputValue(), ':8081');
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('discard saved draft confirms, preserves live settings and survives browser reload', async () => {
  const s = await session();
  try {
    await tab(s.page, 'sites');await s.page.locator('#site_list [data-sel]').first().click();
    await s.page.locator('#sites input[data-f="listen"]').fill(':8081');await s.page.locator('#ux_save').click();
    await s.page.locator('#ux_change_state').filter({ hasText: 'Draft saved' }).waitFor();
    const before = clone(s.active);
    await s.page.locator('#ux_discard_draft').click();await choice(s.page, 'cancel');
    assert.equal(writes(s.calls).some(c => c.method === 'DELETE'), false);
    await s.page.locator('#ux_discard_draft').click();await choice(s.page, 'discard');
    await s.page.locator('#ux_change_state').filter({ hasText: 'matches active settings' }).waitFor();
    assert.deepEqual(s.active, before);
    assert.equal(writes(s.calls).filter(c => c.method === 'DELETE' && c.path === '/api/config?draft=1').length, 1);
    await s.page.reload();
    await s.page.locator('#ux_change_state').filter({ hasText: 'matches active settings' }).waitFor();
    assert.equal(await s.page.evaluate(() => window.testUX.state.saved.sites[0].listen), ':8080');
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('discard failure keeps the saved draft available and never claims success', async () => {
  const s = await session();
  try {
    await tab(s.page, 'sites');await s.page.locator('#site_list [data-sel]').first().click();
    await s.page.locator('#sites input[data-f="listen"]').fill(':8081');await s.page.locator('#ux_save').click();
    await s.page.locator('#ux_change_state').filter({ hasText: 'Draft saved' }).waitFor();
    await s.page.route('**/api/config?draft=1', route => route.request().method() === 'DELETE' ? route.fulfill({ status: 500, body: 'disk error' }) : route.fallback());
    await s.page.locator('#ux_discard_draft').click();await choice(s.page, 'discard');
    await s.page.locator('#toast').filter({ hasText: 'disk error' }).waitFor();
    assert.equal(await s.page.locator('#ux_errors [data-resource-retry="/api/config?draft=1"]').count(), 0);
    await s.page.reload();await s.page.locator('#ux_change_state').filter({ hasText: 'Draft saved' }).waitFor();
    assert.equal(await s.page.evaluate(() => window.testUX.state.saved.sites[0].listen), ':8081');
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('reviewer and viewer cannot discard the shared saved draft', async () => {
  for (const role of ['reviewer', 'viewer']) {
    const s = await session({ role });
    try {
      assert.equal(await s.page.locator('#ux_discard_draft').isDisabled(), true);
      const status = await s.page.evaluate(async () => { try { await window.testUX.api('/api/config?draft=1', { method: 'DELETE' }); } catch (e) { return e.status; } });
      assert.equal(status, 403);assert.equal(writes(s.calls).length, 0);
      assert.deepEqual(s.errors, []);
    } finally { await s.close(); }
  }
});

test('a candidate that is already configured stays disabled for reviewers', async () => {
  const field = { name: 'username', source: 'ARGS_POST', profile: 'email', required: false, min_length: 0, max_length: 0, exclude_rule_ids: [] };
  const s = await session({ role: 'reviewer', pagePolicies: [{ path: '/login', match: 'exact', fields: [field], source: 'manual' }] });
  try {
    await tab(s.page, 'logs');await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');
    await s.page.locator('[data-add-discovered]').first().waitFor();
    assert.equal(await s.page.locator('[data-add-discovered]').first().isDisabled(), true);
    assert.equal(await s.page.locator('#pp_fields .fpolicy').count(), 1);
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('undo can select either scope without changing the other policy on the same URL', async () => {
  const exact = { path: '/login', match: 'exact', mode: 'On', fields: [], source: 'manual' };
  const prefix = { path: '/login', match: 'prefix', mode: 'DetectionOnly', fields: [], source: 'manual' };
  const s = await session({ pagePolicies: [exact, prefix] });
  try {
    await tab(s.page, 'logs');
    for (const scope of ['exact', 'prefix']) {
      await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');
      await s.page.locator('#ux_input_match').selectOption(scope);await submit(s.page);
      await s.page.locator('#pp:not([hidden])').waitFor();
      await s.page.locator('#pp_note').fill(scope + ' changed');
      await s.page.locator('#pp_save').click();await choice(s.page, 'apply');
      await s.page.locator('#pp[hidden]').waitFor({ state: 'attached' });
    }
    await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');
    await s.page.locator('#ux_input_match').selectOption('prefix');await submit(s.page);
    await s.page.locator('#pp:not([hidden])').waitFor();
    await s.page.locator('#ux_page_undo').click();
    await s.page.locator('#ux_input_match').selectOption('exact');await submit(s.page);await choice(s.page, 'apply');
    await s.page.locator('#pp[hidden]').waitFor({ state: 'attached' });
    assert.deepEqual(s.active.sites[0].page_policies.find(p => p.match === 'exact'), exact);
    assert.equal(s.active.sites[0].page_policies.find(p => p.match === 'prefix').note, 'prefix changed');
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});

test('undo refuses to overwrite a page policy changed by someone else', async () => {
  const original = { path: '/login', match: 'prefix', mode: 'DetectionOnly', fields: [], source: 'manual' };
  const s = await session({ pagePolicies: [original] });
  try {
    await tab(s.page, 'logs');await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');
    await s.page.locator('#pp:not([hidden])').waitFor();
    await s.page.locator('#pp_delete').click();await choice(s.page, 'apply');await s.page.locator('#pp[hidden]').waitFor({ state: 'attached' });
    s.active.sites[0].page_policies.push({ ...original, mode: 'On', note: 'another operator' });
    await s.page.locator('[data-event]').first().click();await choice(s.page, 'page');await s.page.locator('#pp:not([hidden])').waitFor();
    const before = writes(s.calls).length;
    await s.page.locator('#ux_page_undo').click();
    await s.page.locator('#ux_page_errors').filter({ hasText: 'Active settings have changed' }).waitFor();
    assert.equal(writes(s.calls).length, before);
    assert.equal(s.active.sites[0].page_policies[0].note, 'another operator');
    assert.deepEqual(s.errors, []);
  } finally { await s.close(); }
});
