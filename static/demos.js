(() => {
 document.querySelectorAll('[data-demo]').forEach(demo => {
  const panels = [...demo.querySelectorAll('[data-demo-panel]')];
  const buttons = [...demo.querySelectorAll('[data-demo-view]')];
  demo.addEventListener('click', event => {
   const button = event.target.closest('button[data-demo-view]');
   if (!button || !demo.contains(button)) return;
   const view = button.dataset.demoView;
   if (!panels.some(panel => panel.dataset.demoPanel === view)) return;
   panels.forEach(panel => { panel.hidden = panel.dataset.demoPanel !== view; });
   buttons.forEach(control => {
    if (control.hasAttribute('aria-pressed')) {
     control.setAttribute('aria-pressed', String(control.dataset.demoView === view));
    }
   });
   if (button.closest('[data-demo-panel]')?.hidden) {
    const screen = button.closest('.phone-mockup,.app-mockup') || demo;
    const navButton = [...screen.querySelectorAll('nav button[data-demo-view]')]
     .find(control => control.dataset.demoView === view);
    navButton?.focus({ preventScroll:true });
   }
  });
 });
})();
