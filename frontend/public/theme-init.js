(function() {
  const theme = localStorage.getItem('cc-theme') || 'system';
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  const resolvedTheme = theme === 'system'
    ? (prefersDark ? 'dark' : 'light')
    : theme;
  // Map to DaisyUI custom theme names
  const dataTheme = resolvedTheme === 'dark' ? 'corporateDark' : 'corporateLight';
  document.documentElement.setAttribute('data-theme', dataTheme);
})();
