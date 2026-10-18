import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';

const html = readFileSync(new URL('../../static/admin.html', import.meta.url), 'utf8');
const script = html.match(/<script\b[^>]*>([\s\S]*?)<\/script>/i)[1];

// Execute the shipping console, including its startup code and event bindings.
// No browser, live WAF, credentials, or external service is needed.
function consoleHarness(fixtures = {}) {
  const elements = new Map();
  const element = id => {
    if (!elements.has(id)) elements.set(id, {
      value: '', checked: false, disabled: false, hidden: false, title: '',
      textContent: '', innerHTML: '', style: {}, dataset: {}, options: [],
      classList: { contains: () => false, toggle() {}, add() {}, remove() {} },
      addEventListener() {}, removeAttribute() {}, querySelectorAll: () => [],
      setAttribute() {}, before() {}, append() {},
    });
    return elements.get(id);
  };
  const context = vm.createContext({
    document: {
      getElementById: element, querySelectorAll: () => [], addEventListener() {},
      createElement: () => element('created-' + elements.size),
      body: { dataset: {} },
    },
    sessionStorage: { getItem: () => '', removeItem() {}, setItem() {} },
    fetch: async (path, options) => typeof fixtures[path] === 'function' ? fixtures[path](options) : ({
      status: 200, ok: true,
      json: async () => fixtures[path] ?? [],
    }),
    setInterval() {}, setTimeout() {}, clearTimeout() {}, URL, AbortController,
    window: { addEventListener() {} },
  });
  const exports = `
    globalThis.consoleCopy = { uiLabel, inspectionStep, fieldPolicyRow, siteCard,
      policyCard, loadAPISecurity, loadDoctor, collectL7Abuse, collectCIDRPolicy,
      configChanges, changeTable, stableConfig, affectedSites, validListen,
      validBackend, buildWizardSite, pageContext, pageErrors, pageRecord,
      attentionItems, api, loadIndependent, uxState };
  `;
  assert.equal((script.match(/  resume\(\);/g) || []).length, 1);
  vm.runInContext(script.replace('  resume();', exports + '\n  resume();'), context);
  return { copy: context.consoleCopy, element };
}

test('console starts and displays modes without changing their API values', () => {
  const { copy } = consoleHarness();
  for (const [value, label] of Object.entries({
    LEARN: 'Learning', DETECT: 'Monitor', ENFORCE: 'Block',
    DetectionOnly: 'Monitor', On: 'Block', Off: 'Off',
    UNKNOWN_TRANSITION: 'Unexpected next step',
    FAILSAFE: 'Acceleration paused for safety',
    'Form (SQLi+XSS strict)': 'Forms — strict protection',
    API1_NORMALIZED_PATH: 'Observed URL fields',
  })) assert.equal(copy.uiLabel(value), label);
  assert.equal(copy.inspectionStep(2), 'Request content');
});

test('unrecognized values remain available and inherited object keys are not labels', () => {
  const { copy } = consoleHarness();
  for (const value of ['FUTURE_STATUS', 'constructor', '__proto__', '<img src=x>', 404]) {
    assert.equal(copy.uiLabel(value), String(value));
  }
  assert.equal(copy.uiLabel(null), '—');
});

test('friendly static dropdown labels preserve exact request values', () => {
  for (const [id, values] of Object.entries({
    api_sequence_mode: ['LEARN', 'DETECT'],
    api_bola_policy_action: ['REVIEW', 'SUPPRESS'],
    api_obj_action: ['INCLUDE', 'SUPPRESS'],
    api_obj_location: ['path', 'query', 'body', 'graphql_variable'],
    pp_match: ['prefix', 'exact'],
    pp_mode: ['', 'Off', 'DetectionOnly', 'On'],
    vector_mode: ['off', 'auto', 'required'],
  })) {
    const content = html.match(new RegExp(`<select id="${id}">([\\s\\S]*?)</select>`))[1];
    assert.deepEqual([...content.matchAll(/<option value="([^"]*)"/g)].map(x => x[1]), values, id);
  }
});

test('buttons that only save drafts do not promise activation', () => {
  for (const id of ['nodes_save', 'ai_save', 'ha_save', 'nt_save', 'sl_save']) {
    const label = html.match(new RegExp(`<button[^>]*id="${id}"[^>]*>([^<]*)</button>`))[1];
    assert.equal(label, 'Save draft', id);
  }
  assert.ok(html.includes('Connection tests use the active settings'));
});

test('field editor retains original field settings and escapes submitted names', () => {
  const { copy } = consoleHarness();
  const row = copy.fieldPolicyRow({
    name: '<password>', source: 'ARGS_POST', profile: 'password', required: true,
    min_length: 12, max_length: 128, exclude_rule_ids: [942100], discovery_source: 'both',
  }, 0);
  assert.ok(row.includes('value="&lt;password&gt;"'));
  assert.ok(row.includes('value="ARGS_POST" selected'));
  assert.ok(row.includes('value="password" selected'));
  assert.ok(row.includes('data-ff="required" checked>Required field'));
  assert.ok(row.includes('Live traffic and page scan'));
  assert.ok(row.includes('value="942100"'));
  assert.ok(!row.includes('Require acceleration'));
});

test('site and policy editors keep protection values and sensitivity settings', () => {
  const { copy } = consoleHarness();
  const site = copy.siteCard({ name: 'shop', listen: ':443', hostnames: [], engine_mode: 'DetectionOnly' }, 0);
  assert.ok(site.includes('class="modechip DetectionOnly">Monitor'));
  assert.ok(site.includes('value="DetectionOnly" selected'));
  const policy = copy.policyCard({ name: 'standard', paranoia_level: 3, exclusions: [] }, 0);
  assert.ok(policy.includes('Inspection sensitivity'));
  assert.ok(policy.includes('data-f="paranoia_level"'));
  assert.ok(policy.includes('value="3" selected>3 — Strict'));
});

test('API protection renders friendly findings and keeps mode-change payload selectors', async () => {
  const { copy, element } = consoleHarness({
    '/api/security/operations': [{ id: 'op', site: 'shop', method: 'POST', path: '/login' }],
    '/api/security/schema/enforcement': [{ operation_id: 'op', mode: 'DETECT', profile_version_id: 'profile' }],
    '/api/security/graphql/policies': [{ id: 'gql', endpoint_operation_id: 'op', mode: 'DETECT' }],
    '/api/security/graphql/violations?limit=100': [{ type: 'DEPTH_LIMIT', mode: 'DETECT', action: 'detect' }],
  });
  await copy.loadAPISecurity();
  for (const id of ['api_enforce_body', 'api_graphql_policies']) {
    const content = element(id).innerHTML;
    assert.ok(content.includes('data-mode="ENFORCE"'), id);
    assert.ok(content.includes('>Block</button>'), id);
    assert.ok(content.includes('>Monitor</span>'), id);
  }
  assert.ok(element('api_graphql_violations').innerHTML.includes('Query is too deeply nested'));
  assert.ok(!element('toast').textContent, element('toast').textContent);
});

test('diagnostics explain results without claiming unexecuted checks passed', async () => {
  const { copy, element } = consoleHarness({
    '/api/doctor': {
      checks: [{ name: 'vectorscan.native', status: 'NOT_RUN', detail: 'acceleration disabled' }],
      truth_boundary: { real_coraza_gate: 'requires release-host qualification transcript' },
    },
  });
  await copy.loadDoctor();
  assert.ok(element('doctor_body').innerHTML.includes('Pattern-matching acceleration'));
  assert.ok(element('doctor_body').innerHTML.includes('Not checked'));
  assert.ok(!element('doctor_body').innerHTML.includes('Passed'));
  assert.ok(element('doctor_truth').innerHTML.includes('A separate deployment verification report is required.'));
});

test('renamed traffic controls serialize the same settings', () => {
  const { copy, element } = consoleHarness();
  element('l7_enabled').checked = true;
  element('l7_requests').value = '100';
  element('l7_window').value = '10';
  element('l7_concurrent').value = '5';
  element('l7_tls_handshakes').value = '20';
  assert.deepEqual(JSON.parse(JSON.stringify(copy.collectL7Abuse())), {
    enabled: true, requests_per_window: 100, window_seconds: 10,
    max_concurrent_connections: 5, tls_handshake_per_window: 20,
  });
});

test('change preview escapes content and redacts secrets inside added records', () => {
  const { copy } = consoleHarness();
  const preview = copy.changeTable(copy.configChanges({}, {
    sites: [{ name: '<shop>', tls_key_provider: { pin: 'PRIVATE-PIN' } }],
    ai: { api_key: 'PRIVATE-KEY', nested: [{ password: 'PRIVATE-PASSWORD' }] },
  }));
  assert.doesNotMatch(preview, /PRIVATE-PIN|PRIVATE-KEY|PRIVATE-PASSWORD|<shop>/);
  assert.match(preview, /hidden/);
  assert.match(preview, /&lt;shop&gt;/);
});

test('stable snapshots omit only editor metadata and preserve rule order', () => {
  const { copy } = consoleHarness();
  const snapshot = JSON.parse(JSON.stringify(copy.stableConfig({
    sites: [{ name: 'shop', _new: true, page_policies: [{ path: '/a' }, { path: '/' }] }],
  })));
  assert.equal('_new' in snapshot.sites[0], false);
  assert.deepEqual(snapshot.sites[0].page_policies.map(p => p.path), ['/a', '/']);
  const changes = copy.configChanges({ rules: [{ name: 'a' }, { name: 'b' }] }, { rules: [{ name: 'b' }, { name: 'a' }] });
  assert.equal(changes[0].path, 'rules.order');
});

test('affected-site preview follows direct site, group, policy and global changes', () => {
  const { copy } = consoleHarness();
  const config = { sites: [{ name: 'a', pool: 'one', policy: 'strict' }, { name: 'b', pool: 'two', policy: 'standard' }] };
  assert.deepEqual(Array.from(copy.affectedSites(config, config, [{ path: 'pools[one].scheme' }])), ['a']);
  assert.deepEqual(Array.from(copy.affectedSites(config, config, [{ path: 'policies[strict].rules' }])), ['a']);
  assert.deepEqual(Array.from(copy.affectedSites(config, config, [{ path: 'sites[b].listen' }])), ['b']);
  assert.deepEqual(Array.from(copy.affectedSites(config, config, [{ path: 'engine_mode' }])), ['a', 'b']);
});

test('site wizard rejects backend credentials and URL paths and defaults safe transport', () => {
  const { copy } = consoleHarness();
  assert.equal(copy.validListen('[::1]:443'), '');
  assert.notEqual(copy.validListen(':65536'), '');
  assert.notEqual(copy.validBackend('https://user:secret@server.test'), '');
  assert.notEqual(copy.validBackend('http://server.test/private'), '');
  assert.equal(copy.validBackend('https://server.test:8443'), '');
  const plan = copy.buildWizardSite({
    name: 'shop', domains: 'shop.test', listen: ':443', backend: 'https://server.test',
    tls: 'http', policy: 'standard', mode: 'DetectionOnly',
  }, { sites: [], pools: [], nodes: [] });
  assert.equal(plan.nodes[0].host, 'server.test');
  assert.equal(plan.pools[0].members[0].port, 443);
  assert.equal(plan.pools[0].scheme, 'https');
  assert.equal(plan.pools[0].backend_tls, undefined); // Never disables verification.
});

test('page policy context preserves execution order and identifies prefix overlap', () => {
  const { copy } = consoleHarness();
  copy.uxState.contextConfig = {
    engine_mode: 'On', sites: [{ name: 'shop', policy: 'strict' }],
    policies: [{ name: 'strict', paranoia_level: 2 }],
  };
  const pages = [{ path: '/', match: 'prefix', mode: 'DetectionOnly' }, { path: '/login', match: 'exact', mode: 'On' }];
  const context = copy.pageContext('shop', '/login', pages);
  assert.ok(context.indexOf('/ (prefix') < context.indexOf('/login (exact'));
  assert.match(context, /Last matching mode override: Block/);
  assert.match(context, /exceptions apply.*all methods/);
  assert.equal(copy.pageRecord(pages, '/login', 'prefix'), null);
  assert.equal(copy.pageRecord(pages, '/login', 'exact').mode, 'On');
});

test('independent requests keep cached results and persistent failure evidence', async () => {
  const fixtures = { '/api/example': [{ id: 1 }] };
  const { copy } = consoleHarness(fixtures);
  await copy.loadIndependent(['/api/example']);
  fixtures['/api/example'] = async () => { throw new Error('offline'); };
  const result = await copy.loadIndependent(['/api/example']);
  assert.equal(result[0][0].id, 1);
  assert.equal(copy.uxState.resources.get('/api/example').error, 'offline');
  assert.ok(copy.uxState.resources.get('/api/example').at);
});

test('duplicate mutations are blocked without clearing the first in-flight guard', async () => {
  let release;
  const held = new Promise(resolve => { release = resolve; });
  const { copy } = consoleHarness({ '/api/example': async () => {
    await held;return { ok: true, status: 200, json: async () => ({ ok: true }) };
  } });
  const first = copy.api('/api/example', { method: 'POST', body: '{}' });
  await assert.rejects(copy.api('/api/example', { method: 'POST', body: '{}' }), /already in progress/);
  assert.ok(copy.uxState.pending.has('POST /api/example'));
  release();await first;
  assert.equal(copy.uxState.pending.has('POST /api/example'), false);
});

test('dashboard separates response samples from WAF blocks and flags unknown TLS expiry', () => {
  const { copy } = consoleHarness();
  const items = copy.attentionItems({ sites: [{ name: 'shop', tls: true }] }, [], [{}], [{ status: 403 }], []);
  const text = items.map(x => x.text).join(' ');
  assert.match(text, /not a WAF-block rate/);
  assert.match(text, /Certificate expiry unavailable/);
  assert.match(text, /not necessarily blocked/);
});

test('console no longer uses native password or chained prompt dialogs', () => {
  assert.doesNotMatch(script, /\bprompt\s*\(/);
  assert.ok(script.includes('beforeunload'));
  assert.ok(script.includes('e.key==="Tab"'));
  assert.ok(script.includes('e.key==="Escape"'));
  assert.ok(script.includes('getClientRects'));
});

test('failed mutations are not offered as blindly repeatable read retries', async () => {
  const path = '/api/sitemap/clear?site=shop';
  const { copy, element } = consoleHarness({ [path]: async () => { throw new Error('connection lost'); } });
  await assert.rejects(copy.api(path, { method: 'POST', body: '{}' }), /connection lost/);
  assert.equal(copy.uxState.resources.get(path).method, 'ACTION');
  assert.doesNotMatch(element('ux_errors').innerHTML, /data-resource-retry/);
  assert.match(element('ux_errors').innerHTML, /Check the current result/);
});
