import { useEffect } from 'react';
import { motion } from 'framer-motion';

const LETTERS_UNO   = ['U', 'N', 'O'];
const LETTERS_CHESS = ['C', 'H', 'E', 'S', 'S'];

export default function LoadingScreen({ onDone }) {
  // Navigate to lobby after the animation completes
  useEffect(() => {
    const t = setTimeout(onDone, 2800);
    return () => clearTimeout(t);
  }, [onDone]);

  return (
    <motion.div
      style={styles.root}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      transition={{ duration: 0.4 }}
    >
      {/* Logo icon */}
      <motion.img
        src="/logo.svg"
        alt="UnoChess logo"
        style={styles.icon}
        initial={{ scale: 0.7, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        transition={{ duration: 0.6, ease: 'easeOut' }}
      />

      {/* Logotype */}
      <div style={styles.logotype}>
        <div style={styles.wordRow}>
          {LETTERS_UNO.map((ch, i) => (
            <motion.span
              key={i}
              style={{ ...styles.letter, ...styles.letterUno }}
              initial={{ y: 20, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              transition={{ delay: 0.5 + i * 0.07, duration: 0.4, ease: 'easeOut' }}
            >
              {ch === 'O' ? '♟' : ch}
            </motion.span>
          ))}
        </div>
        <div style={styles.wordRow}>
          {LETTERS_CHESS.map((ch, i) => (
            <motion.span
              key={i}
              style={{ ...styles.letter, ...styles.letterChess }}
              initial={{ y: 20, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              transition={{ delay: 0.7 + i * 0.06, duration: 0.4, ease: 'easeOut' }}
            >
              {ch}
            </motion.span>
          ))}
        </div>
      </div>

      {/* Progress bar */}
      <motion.div
        style={styles.barTrack}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 1.1 }}
      >
        <motion.div
          style={styles.barFill}
          initial={{ width: '0%' }}
          animate={{ width: '100%' }}
          transition={{ delay: 1.2, duration: 1.4, ease: 'easeInOut' }}
        />
      </motion.div>

      <motion.p
        style={styles.subtitle}
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 1.3 }}
      >
        Loading…
      </motion.p>
    </motion.div>
  );
}

const styles = {
  root: {
    height: '100vh',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 16,
    background: 'var(--bg)',
  },
  icon: {
    width: 90,
    height: 112,
    marginBottom: 8,
  },
  logotype: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    gap: 2,
  },
  wordRow: {
    display: 'flex',
    gap: 2,
  },
  letter: {
    fontFamily: 'Cinzel, serif',
    lineHeight: 1,
    display: 'inline-block',
  },
  letterUno: {
    fontSize: 40,
    fontWeight: 700,
    color: '#e8e8f0',
  },
  letterChess: {
    fontSize: 22,
    fontWeight: 400,
    color: '#7c6af7',
    letterSpacing: 6,
  },
  barTrack: {
    marginTop: 20,
    width: 220,
    height: 3,
    borderRadius: 2,
    background: 'var(--border)',
    overflow: 'hidden',
  },
  barFill: {
    height: '100%',
    borderRadius: 2,
    background: 'linear-gradient(90deg, #c0392b, #2471a3, #1e8449, #d4ac0d)',
  },
  subtitle: {
    marginTop: 10,
    fontSize: 12,
    color: 'var(--text-muted)',
    fontFamily: 'Inter, sans-serif',
    letterSpacing: 1,
  },
};
