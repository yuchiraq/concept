const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const analyticsSource = fs.readFileSync(path.join(__dirname, '../static/analytics.js'), 'utf8');
const contactSource = fs.readFileSync(path.join(__dirname, '../static/contact.js'), 'utf8');
const plain = value => JSON.parse(JSON.stringify(value));

function events() {
 const handlers = new Map();
 return {
  addEventListener(name, handler) { handlers.set(name, [...(handlers.get(name) || []), handler]); },
  emit(name, event = {}) { return Promise.all((handlers.get(name) || []).map(handler => handler(event))); }
 };
}

function analytics({ url = 'https://qoncept.by/', referrer = '', privacy = {}, disabled = false, counter = '113798986' } = {}) {
 const calls = [];
 const scripts = [];
 const document = Object.assign(events(), {
  referrer, currentScript: { dataset: { counterId: counter } },
  head: { appendChild(script) { scripts.push(script); } },
  createElement(tag) { assert.equal(tag, 'script'); return {}; }
 });
 const window = { ym(...args) { calls.push(plain(args)); }, qonceptDisableAnalytics: disabled };
 const context = vm.createContext({ document, window, location: new URL(url), URL, navigator: privacy });
 vm.runInContext(analyticsSource, context);
 return { calls, scripts, document, window, runAgain() { vm.runInContext(analyticsSource, context); } };
}

test('production uses one explicit clean pageview and disables automatic capture', () => {
 const state = analytics({ url: 'https://qoncept.by/en/software?email=PRIVATE&token=SECRET#PRIVATE', referrer: 'https://search.example/private/path?q=PRIVATE#SECRET' });
 assert.equal(state.scripts.length, 1);
 assert.equal(state.scripts[0].src, 'https://mc.yandex.ru/metrika/tag.js?id=113798986');
 assert.equal(state.scripts[0].async, true);
 assert.equal(state.scripts[0].referrerPolicy, 'origin');
 assert.equal(state.calls.length, 2);
 const init = state.calls[0];
 assert.deepEqual(init.slice(0, 2), [113798986, 'init']);
 assert.equal(init[2].defer, true);
 for (const option of ['webvisor', 'clickmap', 'trackLinks', 'trackHash', 'accurateTrackBounce', 'ecommerce', 'childIframe', 'sendTitle']) assert.equal(init[2][option], false, option);
 assert.equal(init[2].disableYtm, true);
 assert.deepEqual(state.calls[1], [113798986, 'hit', 'https://qoncept.by/en/software', { referer: 'https://search.example/', title: 'by Qoncept' }]);
 assert.doesNotMatch(JSON.stringify(state.calls), /PRIVATE|SECRET|private\/path|\?email|#PRIVATE/);
 state.runAgain();
 assert.equal(state.calls.length, 2);
 assert.equal(state.scripts.length, 1);
});

test('only the five fixed goals are accepted and extra data cannot enter params', () => {
 const state = analytics();
 const names = ['lead_form_open', 'lead_submit_success', 'contact_telegram', 'contact_email', 'contact_instagram'];
 for (const name of names) state.window.qonceptAnalytics.goal(name, { email: 'PRIVATE', name: 'PRIVATE', url: 'https://qoncept.by/?PRIVATE' });
 state.window.qonceptAnalytics.goal('arbitrary_PRIVATE');
 assert.deepEqual(state.calls.slice(2), names.map(name => [113798986, 'reachGoal', name]));
 assert.doesNotMatch(JSON.stringify(state.calls), /PRIVATE/);
});

test('direct contacts classify only fixed owner addresses without sending the href', async () => {
 const state = analytics();
 for (const href of ['https://t.me/yu_cq162?start=PRIVATE', 'mailto:chiraqconcept@icloud.com?subject=PRIVATE', 'https://www.instagram.com/chiraqconcept162/#PRIVATE']) {
  await state.document.emit('click', { isTrusted: true, button: 0, target: { closest() { return { getAttribute() { return href; } }; } } });
 }
 assert.deepEqual(state.calls.slice(2), ['contact_telegram', 'contact_email', 'contact_instagram'].map(goal => [113798986, 'reachGoal', goal]));
 for (const href of ['https://t.me/another_account', 'https://t.me.evil.example/yu_cq162', 'mailto:customer@example.com', 'javascript:alert(1)', 'https://instagram.com/another_account']) {
  await state.document.emit('click', { isTrusted: true, button: 0, target: { closest() { return { getAttribute() { return href; } }; } } });
 }
 assert.equal(state.calls.length, 5);
 assert.doesNotMatch(JSON.stringify(state.calls), /PRIVATE/);
 await state.document.emit('click', { isTrusted: false, button: 0, target: { closest() { return { getAttribute() { return 'https://t.me/yu_cq162'; } }; } } });
 assert.equal(state.calls.length, 5);
});

for (const url of [
 'http://localhost:8089/', 'http://127.0.0.1:8089/en/', 'https://qoncept.by:8089/',
 'http://qoncept.by/', 'https://preview.qoncept.by/', 'https://qoncept.by.evil.example/',
 'https://qoncept.by/site/qr', 'https://qoncept.by/en/site/qr', 'https://qoncept.by/api/contact',
 'https://qoncept.by/qr?campaign=card', 'https://qoncept.by/private-personal-path',
 'https://www.qoncept.by/site/qr/login'
]) {
 test(`no library or events on ${url}`, () => {
  const state = analytics({ url });
  assert.equal(state.calls.length, 0);
  assert.equal(state.scripts.length, 0);
  assert.equal(state.window.qonceptAnalytics, undefined);
 });
}

test('language routes and www are public while privacy opt-outs disable the counter', () => {
 for (const url of ['https://www.qoncept.by/', 'https://qoncept.by/uk/business-design', 'https://qoncept.by/be/', 'https://qoncept.by/en/outdoor-advertising']) assert.equal(analytics({ url }).scripts.length, 1);
 for (const options of [{ privacy: { globalPrivacyControl: true } }, { privacy: { doNotTrack: '1' } }, { disabled: true }]) {
  const state = analytics(options);
  assert.equal(state.calls.length, 0);
  assert.equal(state.scripts.length, 0);
 }
});

test('invalid counter overrides do not request an external script', () => {
 for (const counter of ['0', '-1', 'id=1&secret=x', 'NaN', '9007199254740992']) assert.equal(analytics({ counter }).scripts.length, 0);
});

function contactForm(response = { ok: true, data: { success: true } }) {
 const goals = [];
 let requests = 0;
 const input = value => Object.assign(events(), { value, setCustomValidity() {}, focus() {} });
 const contact = input('@test_user');
 const label = { textContent: '' };
 const submit = { disabled: false, querySelector() { return label; } };
 const form = Object.assign(events(), {
  hidden: false, dataset: {},
  elements: { contact, contactMethod: input('telegram'), name: input('Private Test Name'), service: input('website'), message: input('Private Test Message'), website: input('') },
  querySelector() { return submit; }, querySelectorAll() { return []; }, reportValidity() { return true; }, reset() {}
 });
 const contactsButton = Object.assign(events(), { getAttribute() { return 'false'; }, setAttribute() {} });
 const done = events();
 const dialog = {
  open: false, showModal() { this.open = true; }, close() { this.open = false; },
  querySelector(selector) { return selector === '.lead-contacts-toggle' ? contactsButton : selector === '.lead-done' ? done : { hidden: true }; }
 };
 const nodes = {
  '#lead-dialog': dialog, '#lead-form': form, '#lead-contact-label': {}, '#lead-contact-help': {},
  '#lead-status': { hidden: true }, '#lead-success': { hidden: true, focus() {} }
 };
 const document = Object.assign(events(), { documentElement: { dataset: { page: '/', service: 'website' } }, querySelector(selector) { return nodes[selector] || null; } });
 vm.runInNewContext(contactSource, {
  document, window: { qonceptAnalytics: { goal(name) { goals.push(name); } } },
  location: new URL('http://localhost:8089/?utm_source=test'), URLSearchParams,
  sessionStorage: { getItem() { return null; }, setItem() {} },
  setTimeout() { return 1; }, clearTimeout() {}, AbortController,
  async fetch(url) {
   requests++;
   assert.equal(url, '/api/contact');
   if (response.error) throw new Error('mock network failure');
   return { ok: response.ok, async json() { return response.data; } };
  }
 });
 return { goals, form, dialog, document, requests: () => requests };
}

test('form-open goal fires only when the modal opens', async () => {
 const state = contactForm();
 const event = { preventDefault() {}, target: { closest() { return { textContent: 'Discuss project', dataset: {} }; } } };
 await state.document.emit('click', event);
 await state.document.emit('click', event);
 assert.deepEqual(state.goals, ['lead_form_open']);
 state.dialog.close();
 await state.document.emit('click', event);
 assert.deepEqual(state.goals, ['lead_form_open', 'lead_form_open']);
 assert.equal(state.requests(), 0);
});

for (const [label, response, expected] of [
 ['confirmed delivery', { ok: true, data: { success: true } }, ['lead_submit_success']],
 ['server rejection', { ok: false, data: { success: false } }, []],
 ['unsuccessful result', { ok: true, data: { success: false } }, []],
 ['string success is not accepted', { ok: true, data: { success: 'true' } }, []],
 ['HTTP error with success-shaped body', { ok: false, data: { success: true } }, []],
 ['network failure', { error: true }, []]
]) {
 test(`lead goal: ${label}`, async () => {
  const state = contactForm(response);
  assert.deepEqual(state.goals, []);
  await state.form.emit('submit', { preventDefault() {} });
  assert.equal(state.requests(), 1);
  assert.deepEqual(state.goals, expected);
  assert.doesNotMatch(JSON.stringify(state.goals), /Private|test_user/);
 });
}
