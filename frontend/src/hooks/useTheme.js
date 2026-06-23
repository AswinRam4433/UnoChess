import { useState, useEffect } from 'react';

const KEY = 'unochess-theme';

export function getInitialTheme() {
  try {
    const t = localStorage.getItem(KEY);
    if (t === 'light' || t === 'dark') return t;
  } catch { /* ignore */ }
  return 'dark'; // Walnut & Felt is the default
}

export function applyTheme(theme) {
  if (theme === 'light') document.documentElement.setAttribute('data-theme', 'light');
  else document.documentElement.removeAttribute('data-theme');
}

/**
 * useTheme — reads the persisted theme, keeps <html data-theme> in sync,
 * and persists every change. Returns { theme, setTheme, toggle }.
 */
export function useTheme() {
  const [theme, setTheme] = useState(getInitialTheme);

  useEffect(() => {
    applyTheme(theme);
    try { localStorage.setItem(KEY, theme); } catch { /* ignore */ }
  }, [theme]);

  const toggle = () => setTheme(t => (t === 'light' ? 'dark' : 'light'));
  return { theme, setTheme, toggle };
}
