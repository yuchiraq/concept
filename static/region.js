/* A local timezone hint complements browser locales on the first automatic visit.
   Explicit language links and saved manual choices never trigger this probe. */
(() => {
 const read = name => document.cookie.split('; ').find(item => item.startsWith(`${name}=`))?.slice(name.length+1);
 const probe = read('qoncept_region_probe');
 if (!probe) return;
 try {
  if (decodeURIComponent(probe.replace(/\+/g,' ')) !== location.pathname) return;
  const flags = `; Path=/; SameSite=Lax${location.protocol === 'https:' ? '; Secure' : ''}`;
  document.cookie = `qoncept_region_probe=; Max-Age=0${flags}`;
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const country = {'Europe/Minsk':'BY','Europe/Kyiv':'UA','Europe/Kiev':'UA','Europe/Moscow':'RU'}[zone];
  if (!country) return;
  document.cookie = `qoncept_region=${country}; Max-Age=86400${flags}`;
  const lang = country === 'UA' ? 'uk' : 'ru';
  if (document.documentElement.lang === lang) return;
  const path = location.pathname.replace(/^\/(?:en|uk|be)(?=\/|$)/,'') || '/';
  location.replace((lang === 'ru' ? path : `/${lang}${path}`) + location.search + location.hash);
 } catch (_) { /* Browser locale remains the fallback. */ }
})();
