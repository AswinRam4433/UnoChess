import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';

const PIECE_LABELS = { queen: '♛', rook: '♜', bishop: '♝', knight: '♞', pawn: '♟' };
const PIECE_NAMES  = Object.keys(PIECE_LABELS);

// Files a–h, ranks depending on player color
function ownHalfSquares(color) {
  const squares = [];
  const ranks = color === 'White' ? [1, 2, 3, 4] : [5, 6, 7, 8];
  for (const r of ranks)
    for (const f of ['a','b','c','d','e','f','g','h'])
      squares.push(`${f}${r}`);
  return squares;
}

export default function ResurrectionOverlay({ captured, allowedCount, yourColor, onConfirm }) {
  const [placements, setPlacements] = useState([]); // [{ piece, square }]
  const [selected,   setSelected]   = useState(null); // piece name waiting for a square

  const validSquares = ownHalfSquares(yourColor);
  const placed       = new Set(placements.map(p => p.square));
  const remaining    = allowedCount - placements.length;

  function selectPiece(piece) {
    setSelected(s => s === piece ? null : piece);
  }

  function selectSquare(sq) {
    if (!selected) return;
    if (placed.has(sq)) return;
    setPlacements(p => [...p, { piece: selected, square: sq }]);
    setSelected(null);
  }

  function removeAt(i) {
    setPlacements(p => p.filter((_, idx) => idx !== i));
  }

  function confirm() {
    onConfirm(placements);
  }

  return (
    <motion.div
      style={styles.backdrop}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      <motion.div
        style={styles.box}
        initial={{ y: 30, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        exit={{ y: 30, opacity: 0 }}
        transition={{ type: 'spring', damping: 24, stiffness: 300 }}
      >
        <h3 style={styles.title}>Resurrection</h3>
        <p style={styles.sub}>
          Place up to <strong>{allowedCount}</strong> captured piece{allowedCount > 1 ? 's' : ''} on your half.&nbsp;
          <span style={{ color: 'var(--active)' }}>{remaining} remaining</span>
        </p>

        {/* Captured piece tray */}
        <div style={styles.tray}>
          {PIECE_NAMES.filter(p => (captured[p] ?? 0) > 0).map(piece => (
            <motion.button
              key={piece}
              style={{
                ...styles.pieceBtn,
                background: selected === piece ? 'var(--active)' : 'var(--surface)',
                borderColor: selected === piece ? 'var(--active)' : 'var(--border)',
              }}
              onClick={() => selectPiece(piece)}
              whileTap={{ scale: 0.93 }}
            >
              <span style={styles.pieceGlyph}>{PIECE_LABELS[piece]}</span>
              <span style={styles.pieceCount}>×{captured[piece]}</span>
            </motion.button>
          ))}
        </div>

        {/* Square picker (simplified grid) */}
        {selected && (
          <div>
            <p style={styles.squareHint}>Click a square on your half to place the {selected}</p>
            <div style={styles.squareGrid}>
              {validSquares.map(sq => (
                <motion.button
                  key={sq}
                  style={{
                    ...styles.sqBtn,
                    background: placed.has(sq) ? '#333' : 'var(--surface-r)',
                    color: placed.has(sq) ? 'var(--text-muted)' : 'var(--text)',
                    cursor: placed.has(sq) ? 'not-allowed' : 'pointer',
                  }}
                  onClick={() => !placed.has(sq) && selectSquare(sq)}
                  whileHover={placed.has(sq) ? {} : { scale: 1.1 }}
                >
                  {sq}
                </motion.button>
              ))}
            </div>
          </div>
        )}

        {/* Chosen placements */}
        {placements.length > 0 && (
          <div style={styles.placementList}>
            <AnimatePresence>
              {placements.map((p, i) => (
                <motion.div
                  key={i}
                  style={styles.placementRow}
                  initial={{ opacity: 0, x: -10 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0 }}
                >
                  <span>{PIECE_LABELS[p.piece]} {p.piece} → {p.square}</span>
                  <button style={styles.removeBtn} onClick={() => removeAt(i)}>✕</button>
                </motion.div>
              ))}
            </AnimatePresence>
          </div>
        )}

        <div style={styles.actions}>
          <motion.button
            style={{ ...styles.btn, background: 'var(--active)' }}
            onClick={confirm}
            whileTap={{ scale: 0.97 }}
          >
            Confirm ({placements.length} piece{placements.length !== 1 ? 's' : ''})
          </motion.button>
          <motion.button
            style={{ ...styles.btn, background: 'var(--border)', color: 'var(--text-muted)' }}
            onClick={() => onConfirm([])}
            whileTap={{ scale: 0.97 }}
          >
            Skip resurrection
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
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 50,
    borderRadius: 'inherit',
  },
  box: {
    background: 'var(--surface-r)',
    border: '1px solid var(--border)',
    borderRadius: 14,
    padding: 24,
    display: 'flex',
    flexDirection: 'column',
    gap: 14,
    maxWidth: 420,
    width: '100%',
    maxHeight: '80vh',
    overflowY: 'auto',
  },
  title: {
    fontFamily: 'Cinzel, serif',
    fontSize: 17,
    fontWeight: 700,
    color: 'var(--text)',
  },
  sub: {
    fontSize: 13,
    color: 'var(--text-muted)',
    lineHeight: 1.5,
  },
  tray: {
    display: 'flex',
    gap: 8,
    flexWrap: 'wrap',
  },
  pieceBtn: {
    border: '1px solid',
    borderRadius: 8,
    padding: '8px 12px',
    display: 'flex',
    alignItems: 'center',
    gap: 6,
    cursor: 'pointer',
    transition: 'background 0.1s, border-color 0.1s',
  },
  pieceGlyph: { fontSize: 20, color: '#e8e8f0' },
  pieceCount: { fontSize: 12, color: 'var(--text-muted)' },
  squareHint: {
    fontSize: 12,
    color: 'var(--text-muted)',
    marginBottom: 8,
  },
  squareGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(8, 1fr)',
    gap: 4,
  },
  sqBtn: {
    border: '1px solid var(--border)',
    borderRadius: 4,
    padding: '5px 0',
    fontSize: 10,
    fontFamily: 'JetBrains Mono, monospace',
    textAlign: 'center',
    transition: 'background 0.1s',
  },
  placementList: {
    display: 'flex',
    flexDirection: 'column',
    gap: 4,
  },
  placementRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    background: 'var(--surface)',
    borderRadius: 6,
    padding: '6px 10px',
    fontSize: 13,
    color: 'var(--text)',
  },
  removeBtn: {
    background: 'transparent',
    border: 'none',
    color: 'var(--text-muted)',
    cursor: 'pointer',
    fontSize: 13,
    padding: '0 4px',
  },
  actions: {
    display: 'flex',
    gap: 8,
    flexWrap: 'wrap',
  },
  btn: {
    flex: 1,
    border: 'none',
    borderRadius: 8,
    padding: '10px 0',
    fontSize: 13,
    fontWeight: 600,
    color: '#fff',
    cursor: 'pointer',
    minWidth: 120,
  },
};
