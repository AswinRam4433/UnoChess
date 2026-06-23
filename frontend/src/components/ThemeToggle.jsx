import { motion } from 'framer-motion';
import { useTheme } from '../hooks/useTheme';

/**
 * A compact pill that switches between the Walnut & Felt (dark) and
 * Linen & Terracotta (light) palettes. Persists via useTheme.
 */
export default function ThemeToggle({ size = 'md' }) {
  const { theme, toggle } = useTheme();
  const isLight = theme === 'light';
  const dim = size === 'sm' ? 30 : 34;

  return (
    <motion.button
      onClick={toggle}
      aria-label={isLight ? 'Switch to dark theme' : 'Switch to light theme'}
      title={isLight ? 'Walnut & Felt (dark)' : 'Linen & Terracotta (light)'}
      whileTap={{ scale: 0.9 }}
      whileHover={{ borderColor: 'var(--accent)', color: 'var(--accent)' }}
      style={{
        width: dim,
        height: dim,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'var(--panel)',
        border: '1px solid var(--border)',
        borderRadius: 9,
        color: 'var(--text-muted)',
        fontSize: 15,
        lineHeight: 1,
        flexShrink: 0,
      }}
    >
      {isLight ? '☾' : '☀'}
    </motion.button>
  );
}
