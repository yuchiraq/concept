(() => {
 let messages = {};
 try { messages = JSON.parse(document.querySelector('#i18n-messages')?.textContent || '{}'); } catch (_) {}
 window.qonceptText = text => Object.prototype.hasOwnProperty.call(messages,text) ? messages[text] : text;
})();
