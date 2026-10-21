(() => {
 const tr = window.qonceptText || (text => text);
 const button = document.querySelector('#copy-qr-link');
 button?.addEventListener('click', async () => {
  const input = document.querySelector('#tracking-link');
  try { await navigator.clipboard.writeText(input.value); button.textContent = tr('Ссылка скопирована'); }
  catch (_) { input.focus(); input.select(); button.textContent = tr('Скопируйте выделенную ссылку'); }
 });
})();
