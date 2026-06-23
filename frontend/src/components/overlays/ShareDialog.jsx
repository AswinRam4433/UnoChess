import { useState } from 'react';
import { motion } from 'framer-motion';

export default function ShareDialog({ gameID, onConfirm, onClose }) {
  const [copied, setCopied] = useState(false);

  function copy() {
    navigator.clipboard.writeText(gameID).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <motion.div
      style={styles.backdrop}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      onClick={onClose}
    >
      <motion.div
        style={styles.sheet}
        initial={{ y: 80, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        exit={{ y: 80, opacity: 0 }}
        transition={{ type: 'spring', damping: 24, stiffness: 300 }}
        onClick={e => e.stopPropagation()}
      >
        <h2 style={styles.title}>Share with your opponent</h2>
        <p style={styles.sub}>Send them this Game ID to join</p>

        <div style={styles.idRow}>
          <span style={styles.idText}>{gameID}</span>
          <motion.button
            style={styles.copyBtn}
            onClick={copy}
            whileTap={{ scale: 0.95 }}
          >
            {copied ? 'Copied ✓' : 'Copy'}
          </motion.button>
        </div>

        <motion.button
          style={styles.playBtn}
          onClick={onConfirm}
          whileTap={{ scale: 0.97 }}
          whileHover={{ filter: 'brightness(1.08)' }}
        >
          Open game → (waiting for opponent)
        </motion.button>
      </motion.div>
    </motion.div>
  );
}

const styles = {
  backdrop: {
    position: 'fixed',
    inset: 0,
    background: 'rgba(0,0,0,0.6)',
    display: 'flex',
    alignItems: 'flex-end',
    justifyContent: 'center',
    zIndex: 100,
  },
  sheet: {
    background: 'var(--surface-r)',
    border: '1px solid var(--border)',
    borderRadius: '16px 16px 0 0',
    padding: '28px 32px 40px',
    width: '100%',
    maxWidth: 480,
    display: 'flex',
    flexDirection: 'column',
    gap: 16,
  },
  title: {
    fontFamily: 'Cinzel, serif',
    fontSize: 18,
    fontWeight: 600,
    color: 'var(--text)',
  },
  sub: {
    fontSize: 13,
    color: 'var(--text-muted)',
    marginTop: -8,
  },
  idRow: {
    background: 'var(--bg)',
    border: '1px solid var(--border)',
    borderRadius: 8,
    padding: '10px 14px',
    display: 'flex',
    alignItems: 'center',
    gap: 12,
  },
  idText: {
    fontFamily: 'JetBrains Mono, monospace',
    fontSize: 13,
    color: 'var(--text)',
    flex: 1,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  copyBtn: {
    background: 'var(--border)',
    color: 'var(--text)',
    border: 'none',
    borderRadius: 6,
    padding: '5px 14px',
    fontSize: 12,
    fontWeight: 600,
    cursor: 'pointer',
    flexShrink: 0,
  },
  playBtn: {
    background: 'var(--accent)',
    color: 'var(--accent-ink)',
    border: 'none',
    borderRadius: 10,
    padding: '13px 0',
    fontSize: 14,
    fontWeight: 700,
    width: '100%',
    cursor: 'pointer',
    minHeight: 44,
  },
};
