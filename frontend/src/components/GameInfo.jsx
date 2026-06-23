import { motion, AnimatePresence } from 'framer-motion';
import DiscardPile from './DiscardPile';
import { PHASE } from '../lib/constants';

/**
 * GameInfo is now the board-side DOCK. It owns three things:
 *   1. the hero Discard pile        (most important state)
 *   2. the face-down Draw pile      (with a brass count pill)
 *   3. a big Turn / Moves-left number (Cormorant display)
 * Connection status + "your turn" live in the GameScreen topbar now.
 *
 * layout="rail"  → vertical, sits beside the board on desktop
 * layout="bar"   → horizontal felt-inset strip, between board and hand on mobile
 */
export default function GameInfo({ gameState, layout = 'rail' }) {
  if (!gameState) return null;

  const { phase, discardTop, drawPileSize, pendingCombo, history } = gameState;
  const inCombo  = phase === PHASE.IN_COMBO && pendingCombo;
  const isBar    = layout === 'bar';
  const cardSize = isBar ? 'md' : 'lg';

  const wrap = {
    display: 'flex',
    flexDirection: isBar ? 'row' : 'column',
    alignItems: 'center',
    justifyContent: isBar ? 'space-around' : 'flex-start',
    gap: isBar ? 14 : 22,
    ...(isBar ? {
      width: '100%',
      padding: '12px 14px',
      background: 'var(--felt-inset)',
      border: '1px solid var(--border)',
      borderRadius: 14,
    } : {
      paddingTop: 4,
    }),
  };

  return (
    <div style={wrap}>
      <DiscardPile discardTop={discardTop} size={cardSize} />

      <DrawPile count={drawPileSize} size={cardSize} />

      {/* Big readout: moves-left during a combo, otherwise the turn number */}
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2 }}>
        <span style={styles.miniLabel}>{inCombo ? 'Moves left' : 'Turn'}</span>
        <AnimatePresence mode="wait">
          <motion.span
            key={inCombo ? `c${pendingCombo.movesRemaining}` : `t${history?.length ?? 0}`}
            style={{
              ...styles.bigNum,
              color: inCombo ? 'var(--accent)' : 'var(--text)',
            }}
            initial={{ scale: 1.35, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
          >
            {inCombo ? pendingCombo.movesRemaining : (history?.length ?? 0)}
          </motion.span>
        </AnimatePresence>
      </div>
    </div>
  );
}

function DrawPile({ count, size = 'lg' }) {
  const w = size === 'md' ? 56 : 78;
  const h = size === 'md' ? 82 : 114;
  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10 }}>
      <span style={styles.miniLabel}>Draw pile</span>
      <div style={{ position: 'relative', width: w, height: h }}>
        <div style={{
          width: w, height: h, borderRadius: Math.round(w * 0.13),
          background: 'var(--card-back)',
          border: '2px solid var(--card-back-edge)',
          boxShadow: '0 6px 18px rgba(0,0,0,0.4)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <div style={{
            width: '64%', height: '64%', borderRadius: '50%',
            border: '1.5px solid var(--accent-glow)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            color: 'var(--accent)', fontSize: Math.round(w * 0.34),
          }}>♟</div>
        </div>
        <span style={{
          position: 'absolute', bottom: -8, right: -6,
          background: 'var(--accent)', color: 'var(--accent-ink)',
          fontFamily: 'Manrope, sans-serif', fontWeight: 800, fontSize: 11,
          padding: '2px 7px', borderRadius: 20,
          boxShadow: '0 2px 6px rgba(0,0,0,0.35)',
        }}>×{count}</span>
      </div>
    </div>
  );
}

const styles = {
  miniLabel: {
    fontFamily: 'Manrope, sans-serif', fontWeight: 800, fontSize: 10.5,
    letterSpacing: 1.6, textTransform: 'uppercase', color: 'var(--text-muted)',
    whiteSpace: 'nowrap',
  },
  bigNum: {
    fontFamily: '"Cormorant Garamond", serif', fontWeight: 700,
    fontSize: 44, lineHeight: 1,
  },
};
