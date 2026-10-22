(() => {
 const tr = window.qonceptText || (text => text);
 const button = document.querySelector('#copy-qr-link');
 document.querySelector('.qr-destination-form')?.addEventListener('submit', event => {
  const submit = event.currentTarget.querySelector('[type="submit"]');
  submit.disabled = true;
  event.currentTarget.setAttribute('aria-busy', 'true');
 });
 window.addEventListener('pageshow', () => {
  const form = document.querySelector('.qr-destination-form');
  form?.removeAttribute('aria-busy');
  const submit = form?.querySelector('[type="submit"]');
  if (submit) submit.disabled = false;
 });
 button?.addEventListener('click', async () => {
  const input = document.querySelector('#tracking-link');
  try { await navigator.clipboard.writeText(input.value); button.textContent = tr('Ссылка скопирована'); }
  catch (_) { input.focus(); input.select(); button.textContent = tr('Скопируйте выделенную ссылку'); }
 });
})();
