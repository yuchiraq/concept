(() => {
 const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
 const root = document.documentElement;
 document.querySelector('.theme-toggle').addEventListener('click', () => {
  root.dataset.theme = root.dataset.theme === 'dark' ? 'light' : 'dark';
  document.querySelector('meta[name="theme-color"]').content = root.dataset.theme === 'dark' ? '#101112' : '#f4f1e9';
  try { localStorage.setItem('qoncept-theme', root.dataset.theme); } catch (_) {}
 });
 if ('IntersectionObserver' in window) {
  const reveal = new IntersectionObserver(entries => entries.forEach(entry => {
   if (entry.isIntersecting) { entry.target.classList.add('is-visible'); reveal.unobserve(entry.target); }
  }), { threshold: .08 });
  document.querySelectorAll('.reveal').forEach(node => reveal.observe(node));
  if (!reducedMotion.matches) root.classList.add('motion-ready');
  const counters = new IntersectionObserver(entries => entries.forEach(entry => {
   if (!entry.isIntersecting) return;
   counters.unobserve(entry.target);
   if (reducedMotion.matches) return;
   const target = Number(entry.target.dataset.counter); let start;
   const tick = now => { start ??= now; const progress = Math.min((now - start) / 1400, 1); entry.target.textContent = Math.round(target * (1 - Math.pow(1-progress, 4))); if(progress < 1) requestAnimationFrame(tick); };
   requestAnimationFrame(tick);
  }), {threshold: .6});
  document.querySelectorAll('[data-counter]').forEach(node => counters.observe(node));
 }
 if (window.matchMedia('(pointer: fine)').matches) {
  document.querySelectorAll('.spotlight').forEach(card => card.addEventListener('pointermove', event => {
   const rect = card.getBoundingClientRect(); card.style.setProperty('--x', `${event.clientX - rect.left}px`); card.style.setProperty('--y', `${event.clientY - rect.top}px`);
  }));
  document.querySelectorAll('.magnetic').forEach(button => {
   button.addEventListener('pointermove', event => {if(reducedMotion.matches) return; const rect = button.getBoundingClientRect(); button.style.transform = `translate(${(event.clientX-rect.left-rect.width/2)*.12}px, ${(event.clientY-rect.top-rect.height/2)*.12}px)`;});
   button.addEventListener('pointerleave', () => button.style.transform = '');
  });
 }
 document.querySelectorAll('dialog').forEach(dialog => {
  dialog.querySelector('.dialog-close').addEventListener('click', () => dialog.close());
  dialog.addEventListener('click', event => {if(event.target !== dialog) return; const rect = dialog.getBoundingClientRect(); if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom || dialog.id === 'lightbox') dialog.close();});
 });
 const projectDialog = document.querySelector('#project-dialog');
 document.querySelectorAll('[data-project]').forEach(button => button.addEventListener('click', () => {
  const data = document.querySelector(`#data-${button.dataset.project}`).content;
  document.querySelector('#dialog-title').textContent = data.querySelector('[data-title]').textContent;
  document.querySelector('#dialog-category').textContent = data.querySelector('[data-category]').textContent;
  document.querySelector('#dialog-description').textContent = data.querySelector('[data-description]').textContent;
  document.querySelector('#dialog-details').replaceChildren(...Array.from(data.querySelectorAll('li'), item => item.cloneNode(true)));
  document.querySelector('#dialog-link').href = data.querySelector('a').href;
  projectDialog.showModal();
 }));
 const artworks = Array.from(document.querySelectorAll('[data-art]'));
 const lightbox = document.querySelector('#lightbox'); let current = 0;
 function showArt(index) {
  current = (index + artworks.length) % artworks.length;
  const art = artworks[current];
  const image = document.querySelector('#lightbox-image'); image.src = art.dataset.art; image.alt = `${art.dataset.title} — Chiraq Concept`;
  document.querySelector('#lightbox-caption').textContent = `${art.dataset.title} / Chiraq Concept · ${current+1} из ${artworks.length}`;
 }
 artworks.forEach((art,index) => art.addEventListener('click', () => { showArt(index); lightbox.showModal(); }));
 document.querySelector('.lightbox-prev').addEventListener('click', () => showArt(current-1));
 document.querySelector('.lightbox-next').addEventListener('click', () => showArt(current+1));
 lightbox.addEventListener('keydown', event => {if(event.key === 'ArrowRight'){event.preventDefault();showArt(current+1);} if(event.key === 'ArrowLeft'){event.preventDefault();showArt(current-1);}});
})();
