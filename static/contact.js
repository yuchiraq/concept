(() => {
 const dialog = document.querySelector('#lead-dialog');
 const form = document.querySelector('#lead-form');
 if (!dialog || !form) return;
 const contact = form.elements.contact;
 const contactLabel = document.querySelector('#lead-contact-label');
 const contactHelp = document.querySelector('#lead-contact-help');
 const status = document.querySelector('#lead-status');
 const success = document.querySelector('#lead-success');
 const submit = form.querySelector('[type=submit]');
 const contactsButton = dialog.querySelector('.lead-contacts-toggle');
 const contactsPanel = dialog.querySelector('#lead-contacts');
 let source = '';
 let attribution = {};
 const params = new URLSearchParams(location.search);
 try {
  attribution = JSON.parse(sessionStorage.getItem('qoncept-attribution') || '{}');
  if (!attribution || typeof attribution !== 'object') attribution = {};
  if (params.get('utm_source')) {
   attribution = {utmSource:params.get('utm_source').slice(0,120),utmMedium:(params.get('utm_medium')||'').slice(0,120),utmCampaign:(params.get('utm_campaign')||'').slice(0,120)};
   sessionStorage.setItem('qoncept-attribution',JSON.stringify(attribution));
  }
 } catch (_) {
  attribution = {utmSource:(params.get('utm_source')||'').slice(0,120),utmMedium:(params.get('utm_medium')||'').slice(0,120),utmCampaign:(params.get('utm_campaign')||'').slice(0,120)};
 }
 function updateChannel() {
  const method = form.elements.contactMethod.value;
  const settings = {
   telegram:{label:'Ваш Telegram',placeholder:'@username',help:'Укажите имя пользователя, чтобы я мог вам написать.',type:'text',autocomplete:'off',mode:'text'},
   phone:{label:'Ваш телефон',placeholder:'+375 29 123-45-67',help:'Номер с кодом страны для обратной связи.',type:'tel',autocomplete:'tel',mode:'tel'},
   email:{label:'Ваша почта',placeholder:'you@example.com',help:'Адрес, на который можно написать о проекте.',type:'email',autocomplete:'email',mode:'email'}
  }[method];
  contactLabel.textContent = settings.label;
  contactHelp.textContent = settings.help;
  contact.placeholder = settings.placeholder;
  contact.type = settings.type;
  contact.autocomplete = settings.autocomplete;
  contact.inputMode = settings.mode;
  contact.value = '';
  contact.setCustomValidity('');
 }
 form.querySelectorAll('[name=contactMethod]').forEach(input => input.addEventListener('change',updateChannel));
 contact.addEventListener('input',() => contact.setCustomValidity(''));
 function openForm(trigger) {
  source = trigger.textContent.trim().replace(/\s+/g,' ').slice(0,120);
  const service = trigger.dataset.service || (location.pathname === '/design' ? 'design' : location.pathname === '/websites' ? 'website' : 'unsure');
  form.elements.service.value = service;
  if (form.hidden) { form.hidden = false; success.hidden = true; }
  if (!form.dataset.sending) { status.hidden = true; status.textContent = ''; }
  document.querySelector('#project-dialog[open]')?.close();
  if (!dialog.open) dialog.showModal();
  form.elements.name.focus({preventScroll:true});
 }
 document.addEventListener('click',event => {
  const trigger = event.target.closest('[data-open-contact]');
  if (!trigger) return;
  event.preventDefault();
  openForm(trigger);
 });
 contactsButton.addEventListener('click',() => {
  const expanded = contactsButton.getAttribute('aria-expanded') === 'true';
  contactsButton.setAttribute('aria-expanded',String(!expanded));
  contactsPanel.hidden = expanded;
 });
 dialog.querySelector('.lead-done').addEventListener('click',() => dialog.close());
 form.addEventListener('submit',async event => {
  event.preventDefault();
  if (form.dataset.sending) return;
  const method = form.elements.contactMethod.value;
  const value = contact.value.trim();
  if (method === 'telegram' && !/^(?:@|https:\/\/t\.me\/)?[A-Za-z0-9_]{5,32}$/.test(value)) contact.setCustomValidity('Укажите Telegram в формате @username.');
  if (method === 'phone' && (!/^[\d+() .-]+$/.test(value) || value.replace(/\D/g,'').length < 7 || value.replace(/\D/g,'').length > 15)) contact.setCustomValidity('Укажите корректный номер телефона с кодом страны.');
  if (!form.reportValidity()) return;
  const payload = {name:form.elements.name.value.trim(),contactMethod:method,contact:value,service:form.elements.service.value,message:form.elements.message.value.trim(),website:form.elements.website.value,page:location.pathname,source,utmSource:String(attribution.utmSource||'').slice(0,120),utmMedium:String(attribution.utmMedium||'').slice(0,120),utmCampaign:String(attribution.utmCampaign||'').slice(0,120)};
  form.dataset.sending = 'true';
  submit.disabled = true;
  submit.querySelector('span').textContent = 'Отправляем…';
  status.hidden = true;
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(),15000);
  try {
   const response = await fetch('/api/contact',{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload),signal:controller.signal});
   const result = await response.json();
   if (!response.ok || result.success !== true) {
    const error = new Error('delivery'); error.code = result.error; throw error;
   }
   form.reset(); updateChannel(); form.hidden = true; success.hidden = false; success.focus({preventScroll:true});
  } catch (error) {
   status.textContent = error.code === 'rate_limit' ? 'Слишком много попыток. Попробуйте позже или воспользуйтесь кнопкой «Мои контакты».' : error.code === 'invalid_fields' ? 'Проверьте имя и контакт, затем попробуйте ещё раз.' : 'Заявка не отправлена. Попробуйте ещё раз или напишите мне напрямую: кнопка «Мои контакты» находится рядом с формой.';
   status.hidden = false;
  } finally {
   clearTimeout(timeout); delete form.dataset.sending; submit.disabled = false; submit.querySelector('span').textContent = 'Отправить заявку';
  }
 });
})();
