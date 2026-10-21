/* Text motion inspired by Animata: Bottom-Up Letters and Typing Text.
   Adapted to the site's buildless HTML; no runtime dependencies. */
(() => {
 const tr = window.qonceptText || (text => text);
 const reduced = matchMedia('(prefers-reduced-motion: reduce)');
 const brand = document.querySelector('[data-brand-cycle]');
 const typing = document.querySelector('[data-typing-text]');
 let brandTimer, cleanupTimer, typingTimer;
 let rows = [], active = 0, word = 0, length = 6, erasing = true;
 const words = ['дизайн','сайты','программное обеспечение','автоматизацию бизнеса','обложки и анимацию'].map(tr);
 if (brand) {
  rows = ['by Qoncept','qoncept.by'].map((text,index) => {
   const row = document.createElement('span');
   row.className = `brand-row${index === 0 ? ' is-current' : ''}`;
   [...text].forEach((letter,i) => {
    const span = document.createElement('span');
    span.className = 'brand-char'; span.textContent = letter;
    span.style.setProperty('--letter-delay',`${i * 28}ms`);
    row.append(span);
   });
   return row;
  });
  brand.replaceChildren(...rows);
 }
 const stop = () => { clearInterval(brandTimer); clearTimeout(cleanupTimer); clearTimeout(typingTimer); };
 const type = () => {
  if (!typing || reduced.matches || document.hidden) return;
  const text = words[word];
  length += erasing ? -1 : 1;
  typing.textContent = text.slice(0,length);
  let delay = erasing ? 35 : 75;
  if (length === 0) { erasing = false; word = (word + 1) % words.length; delay = 260; }
  else if (!erasing && length === text.length) { erasing = true; delay = 2200; }
  typingTimer = setTimeout(type,delay);
 };
 const start = () => {
  stop();
  if (reduced.matches) {
   active = 0;
   rows.forEach((row,i) => { row.classList.toggle('is-current',i === 0); row.classList.remove('is-leaving'); });
   if (typing) typing.textContent = tr('дизайн и разработку');
   return;
  }
  if (document.hidden) return;
  if (rows.length) brandTimer = setInterval(() => {
   rows[active].classList.remove('is-current'); rows[active].classList.add('is-leaving');
   active = 1 - active; rows[active].classList.remove('is-leaving'); rows[active].classList.add('is-current');
   cleanupTimer = setTimeout(() => rows[1-active].classList.remove('is-leaving'),1000);
  },5000);
  if (typing) {
   word = 0; length = words[0].length; erasing = true; typing.textContent = words[0];
   typingTimer = setTimeout(type,2200);
  }
 };
 if (reduced.addEventListener) reduced.addEventListener('change',start);
 else reduced.addListener(start);
 document.addEventListener('visibilitychange',start);
 start();
})();
