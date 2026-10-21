(() => {
 const root = document.documentElement;
 const preference = window.matchMedia('(prefers-color-scheme: dark)');
 const applyTheme = () => {
  const theme = preference.matches ? 'dark' : 'light';
  root.dataset.theme = theme;
  root.dataset.themeMode = 'auto';
  const meta = document.querySelector('meta[name="theme-color"]');
  if (meta) meta.content = theme === 'dark' ? '#101112' : '#f4f1e9';
 };
 applyTheme();
 if (preference.addEventListener) preference.addEventListener('change', applyTheme);
 else preference.addListener(applyTheme);
})();
