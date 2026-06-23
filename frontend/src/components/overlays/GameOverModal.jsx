import { motion } from 'framer-motion';

const CONFETTI_PIECES = Array.from({ length: 22 }, (_, i) => i);
const CONFETTI_COLORS = ['#c0392b','#2471a3','#1e8449','#d4ac0d','#7c6af7','#e8e8f0'];
const SYMBOLS = ['♟','♞','♝','♜','♛','♚'];

function rng(seed) { return ((seed * 1664525 + 1013904223) & 0xffffffff) >>> 0; }

export default function GameOverModal({ gameOver, yourColor, onRematch, onLobby }) {
  const won     = gameOver.winner === yourColor;
  const message = won ? 'Victory' : 'Defeat';
  const icon    = won ? '♔' : '♚';

  return (
    <motion.div
      style={styles.backdrop}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      {/* Confetti */}
      {CONFETTI_PIECES.map(i => {
        const s1 = rng(i * 7 + 1), s2 = rng(s1), s3 = rng(s2), s4 = rng(s3);
        const left    = (s1 % 100);
        const delay   = (s2 % 2000) / 1000;
        const dur     = 2 + (s3 % 2000) / 1000;
        const color   = CONFETTI_COLORS[s4 % CONFETTI_COLORS.length];
        const symbol  = SYMBOLS[rng(s4) % SYMBOLS.length];
        return (
          <motion.span
            key={i}
            style={{ ...styles.confetti, left: `${left}%`, color }}
            initial={{ y: -30, opacity: 1 }}
            animate={{ y: '110vh', opacity: 0 }}
            transition={{ delay, duration: dur, ease: 'linear' }}
          >
            {symbol}
          </motion.span>
        );
      })}

      <motion.div
        style={styles.modal}
        initial={{ scale: 0.8, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        exit={{ scale: 0.8, opacity: 0 }}
        transition={{ type: 'spring', damping: 22, stiffness: 280 }}
      >
        <motion.div
          style={styles.icon}
          animate={{ rotate: [0, -8, 8, -4, 4, 0] }}
          transition={{ delay: 0.4, duration: 0.6 }}
        >
          {icon}
        </motion.div>

        <div style={styles.titleRow}>
          <h1 style={{ ...styles.title, color: won ? '#7c6af7' : 'var(--text-muted)' }}>
            {message}
          </h1>
        </div>

        <p style={styles.winner}>
          {gameOver.winner} wins
        </p>

        <div style={styles.meta}>
          <div style={styles.metaItem}>
            <span style={styles.metaLabel}>Reason</span>
            <span style={styles.metaValue}>{gameOver.reason}</span>
          </div>
          <div style={styles.metaDivider} />
          <div style={styles.metaItem}>
            <span style={styles.metaLabel}>Turns</span>
            <span style={styles.metaValue}>{gameOver.turns}</span>
          </div>
        </div>

        <div style={styles.actions}>
          <motion.button
            style={{ ...styles.btn, background: 'var(--active)' }}
            onClick={onRematch}
            whileTap={{ scale: 0.97 }}
            whileHover={{ background: '#8e7ef9' }}
          >
            Play Again
          </motion.button>
          <motion.button
            style={{ ...styles.btn, background: 'var(--surface)' }}
            onClick={onLobby}
            whileTap={{ scale: 0.97 }}
          >
            Back to Lobby
          </motion.button>
        </div>
      </motion.div>
    </motion.div>
  );
}

const styles = {
  backdrop: {
    position: 'absolute',
    inset: 0,
    background: 'rgba(0,0,0,0.75)',
    backdropFilter: 'blur(6px)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 80,
    overflow: 'hidden',
    borderRadius: 'inherit',
  },
  confetti: {
    position: 'absolute',
    top: 0,
    fontSize: 20,
    userSelect: 'none',
    pointerEvents: 'none',
  },
  modal: {
    background: 'var(--surface-r)',
    border: '1px solid var(--border)',
    borderRadius: 20,
    padding: '36px 40px',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    gap: 16,
    minWidth: 320,
    position: 'relative',
    zIndex: 1,
  },
  icon: {
    fontSize: 56,
    lineHeight: 1,
  },
  titleRow: {
    textAlign: 'center',
  },
  title: {
    fontFamily: 'Cinzel, serif',
    fontSize: 34,
    fontWeight: 700,
    letterSpacing: 2,
  },
  winner: {
    fontFamily: 'Cinzel, serif',
    fontSize: 15,
    color: 'var(--text-muted)',
    letterSpacing: 1,
  },
  meta: {
    display: 'flex',
    gap: 20,
    alignItems: 'center',
    padding: '12px 20px',
    background: 'var(--bg)',
    borderRadius: 10,
    border: '1px solid var(--border)',
  },
  metaItem: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    gap: 4,
  },
  metaLabel: {
    fontSize: 11,
    color: 'var(--text-muted)',
    textTransform: 'uppercase',
    letterSpacing: 1,
  },
  metaValue: {
    fontSize: 15,
    fontWeight: 600,
    color: 'var(--text)',
    fontFamily: 'JetBrains Mono, monospace',
  },
  metaDivider: {
    width: 1,
    height: 32,
    background: 'var(--border)',
  },
  actions: {
    display: 'flex',
    gap: 10,
    width: '100%',
  },
  btn: {
    flex: 1,
    border: 'none',
    borderRadius: 10,
    padding: '12px 0',
    fontSize: 14,
    fontWeight: 600,
    color: '#fff',
    cursor: 'pointer',
    transition: 'background 0.15s',
  },
};
