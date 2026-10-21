(() => {
 const tr = window.qonceptText || (text => text);
 const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
 const root = document.documentElement;
 const onMediaChange = (query, handler) => {
  if (query.addEventListener) query.addEventListener('change', handler);
  else query.addListener(handler);
 };

 if ('IntersectionObserver' in window) {
  const reveal = new IntersectionObserver(entries => entries.forEach(entry => {
   if (entry.isIntersecting) { entry.target.classList.add('is-visible'); reveal.unobserve(entry.target); }
  }), { threshold: .08 });
  document.querySelectorAll('.reveal').forEach(node => reveal.observe(node));
  if (!reducedMotion.matches) root.classList.add('motion-ready');
  onMediaChange(reducedMotion, () => root.classList.toggle('motion-ready', !reducedMotion.matches));
  const counters = new IntersectionObserver(entries => entries.forEach(entry => {
   if (!entry.isIntersecting) return;
   counters.unobserve(entry.target);
   if (reducedMotion.matches) return;
   const target = Number(entry.target.dataset.counter);
   if (!Number.isFinite(target)) return;
   let start;
   const tick = now => {
    start ??= now;
    const progress = reducedMotion.matches ? 1 : Math.min((now - start) / 1400, 1);
    entry.target.textContent = Math.round(target * (1 - Math.pow(1 - progress, 4)));
    if (progress < 1) requestAnimationFrame(tick);
   };
   requestAnimationFrame(tick);
  }), { threshold: .6 });
  document.querySelectorAll('[data-counter]').forEach(node => counters.observe(node));
 }

 if (window.matchMedia('(pointer: fine)').matches) {
  document.querySelectorAll('.spotlight').forEach(card => card.addEventListener('pointermove', event => {
   const rect = card.getBoundingClientRect();
   card.style.setProperty('--x', `${event.clientX - rect.left}px`);
   card.style.setProperty('--y', `${event.clientY - rect.top}px`);
  }));
  document.querySelectorAll('.magnetic').forEach(button => {
   button.addEventListener('pointermove', event => {
    if (reducedMotion.matches) return;
    const rect = button.getBoundingClientRect();
    button.style.transform = `translate(${(event.clientX - rect.left - rect.width / 2) * .12}px, ${(event.clientY - rect.top - rect.height / 2) * .12}px)`;
   });
   button.addEventListener('pointerleave', () => button.style.transform = '');
  });
 }

 const topbar = document.querySelector('.topbar');
 const topnav = document.querySelector('#primary-nav') || document.querySelector('.topnav');
 const navToggle = topbar?.querySelector('.nav-toggle');
 if (topbar && topnav) {
  const mobileNav = window.matchMedia('(max-width: 760px)');
  const scrollPosition = () => Math.max(0, Math.min(window.scrollY, document.documentElement.scrollHeight - window.innerHeight));
  let lastY = scrollPosition();
  let direction = 0;
  let distance = 0;
  let pendingScroll = false;
  let ignoreScrollUntil = 0;
  let manuallyOpened = false;
  const setCollapsed = collapsed => {
   collapsed = mobileNav.matches && collapsed;
   if (collapsed && topnav.contains(document.activeElement) && document.activeElement.matches(':focus-visible')) return;
   topbar.classList.toggle('is-nav-collapsed', collapsed);
   topnav.inert = collapsed;
   if (collapsed) topnav.setAttribute('aria-hidden', 'true');
   else topnav.removeAttribute('aria-hidden');
   if (navToggle) {
    navToggle.setAttribute('aria-expanded', String(!collapsed));
    navToggle.setAttribute('aria-label', collapsed ? tr('Показать навигацию') : tr('Скрыть навигацию'));
   }
  };
  const updateNav = () => {
   pendingScroll = false;
   const y = scrollPosition();
   const delta = y - lastY;
   lastY = y;
   if (!mobileNav.matches) return;
   if (manuallyOpened) { direction = 0; distance = 0; return; }
   if (performance.now() < ignoreScrollUntil) { direction = 0; distance = 0; return; }
   if (y < 48) { distance = 0; setCollapsed(false); return; }
   const nextDirection = Math.sign(delta);
   if (!nextDirection) return;
   if (nextDirection !== direction) { direction = nextDirection; distance = 0; }
   distance += Math.abs(delta);
   if (distance >= 18) {
    setCollapsed(direction > 0 && y > 96);
    distance = 0;
   }
  };
  window.addEventListener('scroll', () => {
   if (!pendingScroll) { pendingScroll = true; requestAnimationFrame(updateNav); }
  }, { passive: true });
  const manuallySetCollapsed = collapsed => {
   // Ignore any focus/scroll restoration caused by a tap while the menu animates.
   // The nav itself overlays the page, so opening it never moves document content.
   ignoreScrollUntil = performance.now() + 400;
   manuallyOpened = !collapsed;
   lastY = scrollPosition();
   direction = 0;
   distance = 0;
   setCollapsed(collapsed);
  };
  navToggle?.addEventListener('click', () => manuallySetCollapsed(!topbar.classList.contains('is-nav-collapsed')));
  topbar.addEventListener('click', event => {
   if (!event.target.closest('.nav-toggle') && topbar.classList.contains('is-nav-collapsed')) manuallySetCollapsed(false);
  });
  topbar.addEventListener('focusin', event => {
   if (!event.target.closest('.nav-toggle') && event.target.matches(':focus-visible')) manuallySetCollapsed(false);
  });
  const resumeAutoNavigation = () => {
   if (!manuallyOpened) return;
   manuallyOpened = false;
   ignoreScrollUntil = 0;
   lastY = scrollPosition();
   direction = 0;
   distance = 0;
  };
  // Inertial scrolling and focus restoration are not new scroll gestures. Keep a
  // manually opened menu visible until the visitor starts another interaction.
  window.addEventListener('wheel', resumeAutoNavigation, { passive: true });
  window.addEventListener('touchstart', event => {
   if (!topbar.contains(event.target)) resumeAutoNavigation();
  }, { passive: true });
  window.addEventListener('keydown', event => {
   if (['PageDown', 'PageUp', 'ArrowDown', 'ArrowUp', 'Home', 'End', ' '].includes(event.key) && !event.target.closest('input, textarea, select, [contenteditable="true"]')) resumeAutoNavigation();
  });
  onMediaChange(mobileNav, () => { manuallyOpened = false; lastY = scrollPosition(); direction = 0; distance = 0; setCollapsed(false); });
  setCollapsed(false);
 }

 const videos = Array.from(document.querySelectorAll('video'));
 if (videos.length) {
  const visibility = new Map();
  const syncPreview = video => {
   if (!video.hasAttribute('data-motion-preview')) return;
   if (visibility.get(video) && !reducedMotion.matches && !document.hidden) video.play()?.catch(() => {});
   else video.pause();
  };
  videos.forEach(video => {
   video.playsInline = true;
   if (video.hasAttribute('data-motion-preview')) {
    video.controls = false;
    video.autoplay = !reducedMotion.matches;
    video.muted = true;
    video.defaultMuted = true;
    video.loop = true;
    video.disablePictureInPicture = true;
    video.disableRemotePlayback = true;
    video.tabIndex = -1;
   }
  });
  if ('IntersectionObserver' in window) {
   const videoObserver = new IntersectionObserver(entries => entries.forEach(entry => {
    const visible = entry.isIntersecting && entry.intersectionRatio >= .15;
    visibility.set(entry.target, visible);
    if (!visible) entry.target.pause();
    syncPreview(entry.target);
   }), { threshold: [0, .15] });
   videos.forEach(video => videoObserver.observe(video));
  } else videos.forEach(video => { visibility.set(video, true); syncPreview(video); });
  onMediaChange(reducedMotion, () => videos.forEach(syncPreview));
  document.addEventListener('visibilitychange', () => videos.forEach(video => {
   if (document.hidden) video.pause();
   else syncPreview(video);
  }));
  // A normal page tap can also resume muted animation if Safari blocked autoplay.
  document.addEventListener('pointerup', () => videos.forEach(syncPreview), { passive: true });
 }

 document.querySelectorAll('dialog').forEach(dialog => {
  dialog.querySelector('.dialog-close')?.addEventListener('click', () => dialog.close());
  dialog.addEventListener('click', event => {
   if (event.target !== dialog) return;
   const rect = dialog.getBoundingClientRect();
   if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom || dialog.id === 'lightbox') dialog.close();
  });
 });

 const projectDialog = document.querySelector('#project-dialog');
 if (projectDialog) document.querySelectorAll('[data-project]').forEach(button => button.addEventListener('click', () => {
  const template = document.getElementById(`data-${button.dataset.project}`);
  if (!template?.content) return;
  const data = template.content;
  [['#dialog-title', '[data-title]'], ['#dialog-category', '[data-category]'], ['#dialog-description', '[data-description]']].forEach(([target, source]) => {
   const node = projectDialog.querySelector(target);
   if (node) node.textContent = data.querySelector(source)?.textContent || '';
  });
  projectDialog.querySelector('#dialog-details')?.replaceChildren(...Array.from(data.querySelectorAll('li'), item => item.cloneNode(true)));
  const link = projectDialog.querySelector('#dialog-link');
  const sourceLink = data.querySelector('a[data-cta]') || data.querySelector('a');
  if (link) {
   let href = sourceLink?.getAttribute('href');
   const isWorkService = button.dataset.project === 'workservice';
   if (isWorkService && (!sourceLink?.hasAttribute('data-cta') || /appworks\.avayusstroi\.by/i.test(href || ''))) href = 'https://t.me/yu_cq162';
   link.hidden = !href;
   if (href) link.href = href;
   else link.removeAttribute('href');
   if (isWorkService) {
    link.href = '#contacts';
    link.dataset.openContact = '';
    link.dataset.service = 'software';
    link.removeAttribute('target');
   } else {
    delete link.dataset.openContact;
    delete link.dataset.service;
    link.target = '_blank';
   }
   const icon = link.querySelector('.icon-arrow')?.cloneNode(true);
   link.replaceChildren(document.createTextNode(sourceLink?.dataset.label || (isWorkService ? tr('Обсудить свой сервис') : tr('Открыть сайт'))));
   if (icon) link.append(icon);
  }
  if (!projectDialog.open) projectDialog.showModal();
 }));

 const artworks = Array.from(document.querySelectorAll('[data-art]'));
 const lightbox = document.querySelector('#lightbox');
 const lightboxImage = lightbox?.querySelector('#lightbox-image');
 const lightboxCaption = lightbox?.querySelector('#lightbox-caption');
 let current = 0;
 function showArt(index) {
  if (!artworks.length || !lightboxImage) return;
  current = (index + artworks.length) % artworks.length;
  const art = artworks[current];
  const title = art.dataset.title || tr('Авторская работа');
  lightboxImage.src = art.dataset.art;
  lightboxImage.alt = `${title} — Chiraq Concept`;
  if (lightboxCaption) lightboxCaption.textContent = `${title} / Chiraq Concept · ${current + 1} ${tr('из')} ${artworks.length}`;
 }
 if (lightbox && lightboxImage && artworks.length) {
  artworks.forEach((art, index) => art.addEventListener('click', () => { showArt(index); if (!lightbox.open) lightbox.showModal(); }));
  lightbox.querySelector('.lightbox-prev')?.addEventListener('click', () => showArt(current - 1));
  lightbox.querySelector('.lightbox-next')?.addEventListener('click', () => showArt(current + 1));
  lightbox.addEventListener('keydown', event => {
   if (event.key === 'ArrowRight') { event.preventDefault(); showArt(current + 1); }
   if (event.key === 'ArrowLeft') { event.preventDefault(); showArt(current - 1); }
  });
 }
})();
