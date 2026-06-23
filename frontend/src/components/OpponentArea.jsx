import { motion } from 'framer-motion';

// Renders face-down cards as a visual count for the opponent
export default function OpponentArea({ opponentHandCount, opponentColor, isTheirTurn }) {
  const shown = Math.min(opponentHandCount, 10);

  return (
    <div style={styles.root}>
      <div style={styles.info}>
        <span style={{
          ...styles.tag,
          background: opponentColor === 'White' ? '#ddd' : '#1a1a1a',
          color:      opponentColor === 'White' ? '#111' : '#ccc',
          border:     opponentColor === 'Black' ? '1px solid #444' : 'none',
        }}>
          {opponentColor}
        </span>
        <span style={styles.label}>Opponent</span>
        {isTheirTurn && (
          <motion.span
            style={styles.theirTurn}
            animate={{ opacity: [1, 0.4, 1] }}
            transition={{ repeat: Infinity, duration: 1.2 }}
          >
            ● thinking…
          </motion.span>
        )}
      </div>

      <div style={styles.cardRow}>
        {Array.from({ length: shown }).map((_, i) => (
          <motion.div
            key={i}
            style={{
              ...styles.faceDown,
              marginLeft: i === 0 ? 0 : -28,
              zIndex: i,
            }}
            initial={{ x: 20, opacity: 0 }}
            animate={{ x: 0, opacity: 1 }}
            transition={{ delay: i * 0.02 }}
          />
        ))}
        {opponentHandCount > 10 && (
          <span style={styles.more}>+{opponentHandCount - 10}</span>
        )}
      </div>

      <span style={styles.count}>{opponentHandCount} card{opponentHandCount !== 1 ? 's' : ''}</span>
    </div>
  );
}

const styles = {
  root: {
    display: 'flex',
    flexDirection: 'column',
    gap: 6,
    padding: '8px 4px',
  },
  info: {
    display: 'flex',
    alignItems: 'center',
    gap: 8,
  },
  tag: {
    padding: '2px 8px',
    borderRadius: 4,
    fontSize: 11,
    fontWeight: 700,
    letterSpacing: 0.5,
  },
  label: {
    fontSize: 12,
    color: 'var(--text-muted)',
  },
  theirTurn: {
    fontSize: 11,
    color: '#7c6af7',
    fontWeight: 500,
  },
  cardRow: {
    display: 'flex',
    alignItems: 'center',
    height: 38,
    paddingLeft: 4,
  },
  faceDown: {
    width: 26,
    height: 38,
    borderRadius: 5,
    background: 'linear-gradient(135deg, #1e1e2e 0%, #2a2a3f 100%)',
    border: '1.5px solid var(--border)',
    boxShadow: '1px 1px 4px rgba(0,0,0,0.5)',
    position: 'relative',
    flexShrink: 0,
  },
  more: {
    marginLeft: 8,
    fontSize: 12,
    color: 'var(--text-muted)',
    fontFamily: 'JetBrains Mono, monospace',
  },
  count: {
    fontSize: 11,
    color: 'var(--text-muted)',
  },
};
