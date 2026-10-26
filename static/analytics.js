/* Only explicit, non-personal conversion events. Never serialize a form or URL
   query. The real Metrica library is not requested on previews or private pages. */
(() => {
 const productionOrigins = new Set(['https://qoncept.by', 'https://www.qoncept.by']);
 const publicPath = /^(?:\/(?:ru|en|uk|be))?(?:\/(?:websites|design|software|outdoor-advertising|business-design))?\/?$/;
 if (!productionOrigins.has(location.origin) || !publicPath.test(location.pathname)) return;
 if (window.qonceptDisableAnalytics === true || navigator.globalPrivacyControl === true || navigator.doNotTrack === '1') return;
 if (window.qonceptAnalytics) return;

 const configured = document.currentScript?.dataset.counterId || '113798986';
 if (!/^[1-9]\d*$/.test(configured)) return;
 const counterID = Number(configured);
 if (!Number.isSafeInteger(counterID)) return;
 const pageURL = location.origin + location.pathname;
 // Only the referring origin is useful for acquisition reports. Never forward
 // a search term, email address, token, or private path from its full URL.
 let referrer = '';
 if (document.referrer) {
  try {
   const source = new URL(document.referrer);
   referrer = ['https:', 'http:'].includes(source.protocol) ? source.origin + '/' : pageURL;
  } catch (_) { referrer = pageURL; }
 }
 const goals = new Set(['lead_form_open', 'lead_submit_success', 'contact_telegram', 'contact_email', 'contact_instagram']);
 const send = (...args) => {
  try { window.ym(counterID, ...args); } catch (_) { /* Analytics must never interrupt contact or navigation. */ }
 };
 if (typeof window.ym !== 'function') {
  window.ym = function() { (window.ym.a = window.ym.a || []).push(arguments); };
  window.ym.l = Date.now();
 }

 send('init', {
  defer: true,
  url: pageURL,
  referrer,
  webvisor: false,
  clickmap: false,
  trackLinks: false,
  trackHash: false,
  accurateTrackBounce: false,
  ecommerce: false,
  childIframe: false,
  disableYtm: true,
  sendTitle: false
 });
 // Establish the sanitized artificial URL before reachGoal: the SDK also uses
 // this state for the goal's referrer instead of reading location.href again.
 send('hit', pageURL, { referer: referrer, title: 'by Qoncept' });
 window.qonceptAnalytics = Object.freeze({
  goal(name) { if (goals.has(name)) send('reachGoal', name); }
 });

 document.addEventListener('click', event => {
  if (event.isTrusted === false || (typeof event.button === 'number' && event.button !== 0)) return;
  const link = event.target.closest?.('a[href]');
  if (!link) return;
  let target;
  try { target = new URL(link.getAttribute('href'), location.origin); } catch (_) { return; }
  const host = target.hostname.toLowerCase().replace(/^www\./, '');
  const path = target.pathname.replace(/\/$/, '');
  if (target.protocol === 'mailto:' && path.toLowerCase() === 'chiraqconcept@icloud.com') window.qonceptAnalytics.goal('contact_email');
  else if (target.protocol === 'https:' && host === 't.me' && path === '/yu_cq162') window.qonceptAnalytics.goal('contact_telegram');
  else if (target.protocol === 'https:' && host === 'instagram.com' && path === '/chiraqconcept162') window.qonceptAnalytics.goal('contact_instagram');
 });

 const script = document.createElement('script');
 script.async = true;
 script.src = `https://mc.yandex.ru/metrika/tag.js?id=${counterID}`;
 script.referrerPolicy = 'origin';
 document.head.appendChild(script);
})();
