const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

// Exercise the browser script without depending on an installed browser. The
// fake media element can reject autoplay, as Safari does under device policies.
function eventTarget() {
 const listeners = new Map();
 return {
  addEventListener(type, handler) { listeners.set(type, [...(listeners.get(type) || []), handler]); },
  emit(type, event = {}) { (listeners.get(type) || []).forEach(handler => handler(event)); }
 };
}

function classes() {
 const values = new Set();
 return {
  add(value) { values.add(value); },
  toggle(value, enabled) { if (enabled) values.add(value); else values.delete(value); },
  contains(value) { return values.has(value); }
 };
}

function browser({ reduced = false, saveData = false, blocked = false } = {}) {
 const poster = { classList: classes() };
 const source = { dataset: { src: '/static/videos/previews/example.mp4' }, removeAttribute(name) { if (name === 'data-src') delete this.dataset.src; } };
 const video = Object.assign(eventTarget(), {
  dataset: {}, paused: true, controls: true, loadCount: 0, playCount: 0, blocked,
  closest() { return poster; },
  setAttribute() {},
  querySelectorAll() { return source.dataset.src ? [source] : []; },
  load() { this.loadCount++; this.emit('emptied'); },
  play() {
   this.playCount++;
   if (this.blocked) return Promise.reject(new Error('NotAllowedError'));
   this.paused = false;
   this.emit('playing');
   return Promise.resolve();
  },
  pause() { if (!this.paused) { this.paused = true; this.emit('pause'); } }
 });
 const mediaQuery = Object.assign(eventTarget(), { matches: reduced });
 const connection = Object.assign(eventTarget(), { saveData });
 const observers = [];
 class IntersectionObserver {
  constructor(callback, options) { this.callback = callback; this.options = options; observers.push(this); }
  observe() {}
  unobserve() {}
  emit(visible) { this.callback([{ target: video, isIntersecting: visible, intersectionRatio: visible ? .75 : 0 }]); }
 }
 const document = Object.assign(eventTarget(), {
  documentElement: { classList: classes() }, hidden: false,
  querySelector() { return null; },
  querySelectorAll(selector) { return selector === 'video[data-motion-preview]' ? [video] : []; }
 });
 const window = Object.assign(eventTarget(), {
  IntersectionObserver,
  matchMedia(query) { return query === '(prefers-reduced-motion: reduce)' ? mediaQuery : { matches: false }; }
 });
 vm.runInNewContext(fs.readFileSync(path.join(__dirname, '../static/app.js'), 'utf8'), {
  window, document, navigator: { connection }, IntersectionObserver
 });
 return { video, source, poster, document, window, mediaQuery, connection,
  nearby: observers.find(observer => observer.options.rootMargin),
  visible: observers.find(observer => Array.isArray(observer.options.threshold))
 };
}

async function settle() { await Promise.resolve(); await Promise.resolve(); await Promise.resolve(); }

test('videos stay unloaded until near the viewport and play muted inline', async () => {
 const state = browser();
 assert.equal(state.video.loadCount, 0);
 assert.equal(state.source.src, undefined);
 assert.equal(state.video.controls, false);
 assert.equal(state.video.muted, true);
 assert.equal(state.video.defaultMuted, true);
 assert.equal(state.video.playsInline, true);
 state.nearby.emit(true);
 assert.equal(state.video.loadCount, 1);
 assert.equal(state.video.playCount, 0);
 assert.match(state.source.src, /previews\/example\.mp4$/);
 state.visible.emit(true);
 await settle();
 assert.equal(state.poster.classList.contains('is-playing'), true);
 state.visible.emit(false);
 assert.equal(state.video.paused, true);
 assert.equal(state.poster.classList.contains('is-playing'), false);
});

test('denied autoplay keeps the poster and retries on a real page gesture', async () => {
 const state = browser({ blocked: true });
 state.visible.emit(true);
 await settle();
 assert.equal(state.poster.classList.contains('is-playing'), false);
 assert.equal(state.video.controls, false);
 state.video.blocked = false;
 state.document.emit('touchend');
 await settle();
 assert.equal(state.video.playCount, 2);
 assert.equal(state.poster.classList.contains('is-playing'), true);
});

test('a fast scroll away and back starts a new play without waiting for the aborted attempt', async () => {
 const state = browser();
 const pending = [];
 state.video.play = function() {
  this.playCount++;
  this.paused = false;
  return new Promise((resolve, reject) => pending.push({ resolve, reject }));
 };
 state.visible.emit(true);
 state.visible.emit(false);
 state.visible.emit(true);
 assert.equal(state.video.playCount, 2);
 state.video.emit('playing');
 assert.equal(state.poster.classList.contains('is-playing'), true);
 pending[0].reject(new Error('AbortError'));
 await settle();
 assert.equal(state.poster.classList.contains('is-playing'), true);
 pending[1].resolve();
 await settle();
});

test('a late native playing event cannot keep an offscreen preview running', () => {
 const state = browser();
 state.visible.emit(false);
 state.video.paused = false;
 state.video.emit('playing');
 assert.equal(state.video.paused, true);
 assert.equal(state.poster.classList.contains('is-playing'), false);
});

for (const preference of ['reduced', 'saveData']) {
 test(`${preference} leaves a static cover without downloading video`, () => {
  const state = browser({ [preference]: true });
  state.nearby.emit(true);
  state.visible.emit(true);
  state.document.emit('click');
  assert.equal(state.video.loadCount, 0);
  assert.equal(state.video.playCount, 0);
  assert.equal(state.poster.classList.contains('is-playing'), false);
 });
}

test('background tabs pause previews and returning to the page resumes them', async () => {
 const state = browser();
 state.visible.emit(true);
 await settle();
 state.document.hidden = true;
 state.document.emit('visibilitychange');
 assert.equal(state.video.paused, true);
 state.document.hidden = false;
 state.document.emit('visibilitychange');
 await settle();
 assert.equal(state.video.paused, false);
 assert.equal(state.video.loadCount, 1);
});
