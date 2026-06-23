import { motion, AnimatePresence } from 'framer-motion';
import UnoCard from './UnoCard';
import { PHASE } from '../lib/constants';

const PHASE_LABELS = {
  [PHASE.AWAITING_CARD]:         'Play a card',
  [PHASE.IN_COMBO]:              'Chess combo',
  [PHASE.AWAITING_RESURRECTION]: 'Resurrection',
  [PHASE.TURN_COMPLETE]:         'Advancing…',
  [PHASE.GAME_OVER]:             'Game over',
};

const PHASE_COLORS = {
  [PHASE.AWAITING_CARD]:         '#7c6af7',
  [PHASE.IN_COMBO]:              '#d4ac0d',
  [PHASE.AWAITING_RESURRECTION]: '#1e8449',
  [PHASE.TURN_COMPLETE]:         'var(--text-muted)',
  [PHASE.GAME_OVER]:             '#c0392b',
};

export default function GameInfo({ gameState, yourColor, wsStatus }) {
  if (!gameState) return null;

  const {
    phase, activeColor, discardTop, drawPileSize,
    pendingCombo, history, yourHand,
  } = gameState;

  const isYourTurn = activeColor === yourColor;
  const phaseColor = PHASE_COLORS[phase] ?? 'var(--text)';

  return (
    <div style={styles.root}>
      {/* Connection status */}
      <div style={styles.connRow}>
        <span style={{
          ...styles.connDot,
          background: wsStatus === 'open' ? '#1e8449' : '#c0392b',
        }} />
        <span style={styles.connLabel}>
          {wsStatus === 'open' ? 'Connected' : wsStatus === 'connecting' ? 'Connecting…' : 'Disconnected'}
        </span>
      </div>

      {/* Discard pile */}
      <div style={styles.section}>
        <span style={styles.sectionLabel}>Discard</span>
        <AnimatePresence mode="wait">
          {discardTop?.value ? (
            <motion.div
              key={discardTop.value + discardTop.color}
              initial={{ rotateY: 90, opacity: 0 }}
              animate={{ rotateY: 0, opacity: 1 }}
              exit={{ rotateY: -90, opacity: 0 }}
              transition={{ duration: 0.25 }}
            >
              <UnoCard card={discardTop} playable={false} size="sm" />
            </motion.div>
          ) : (
            <div style={styles.emptyPile}>—</div>
          )}
        </AnimatePresence>
      </div>

      {/* Draw pile */}
      <div style={styles.section}>
        <span style={styles.sectionLabel}>Draw pile</span>
        <div style={styles.drawPile}>
          <div style={styles.drawFaceDown} />
          <span style={styles.drawCount}>{drawPileSize}</span>
        </div>
      </div>

      {/* Phase */}
      <div style={styles.section}>
        <span style={styles.sectionLabel}>Phase</span>
        <AnimatePresence mode="wait">
          <motion.span
            key={phase}
            style={{ ...styles.phaseText, color: phaseColor }}
            initial={{ opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
          >
            {PHASE_LABELS[phase] ?? phase}
          </motion.span>
        </AnimatePresence>
      </div>

      {/* Combo counter */}
      <AnimatePresence>
        {pendingCombo && (
          <motion.div
            style={styles.comboBox}
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.9 }}
          >
            <span style={styles.comboLabel}>Moves left</span>
            <motion.span
              key={pendingCombo.movesRemaining}
              style={styles.comboCount}
              animate={{ scale: [1.4, 1] }}
              transition={{ duration: 0.2 }}
            >
              {pendingCombo.movesRemaining}
            </motion.span>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Your turn indicator */}
      <AnimatePresence>
        {isYourTurn && phase !== PHASE.GAME_OVER && (
          <motion.div
            style={styles.yourTurn}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <motion.span
              animate={{ scale: [1, 1.15, 1] }}
              transition={{ repeat: Infinity, duration: 1.4 }}
              style={styles.turnDot}
            >
              ●
            </motion.span>
            <span style={styles.yourTurnText}>Your turn</span>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Turn count */}
      <div style={styles.turnCount}>
        <span style={styles.sectionLabel}>Turn</span>
        <span style={styles.turnNum}>{history?.length ?? 0}</span>
      </div>

      {/* Your hand count */}
      <div style={styles.turnCount}>
        <span style={styles.sectionLabel}>Your hand</span>
        <span style={styles.turnNum}>{yourHand?.length ?? 0}</span>
      </div>
    </div>
  );
}

const styles = {
  root: {
    display: 'flex',
    flexDirection: 'column',
    gap: 10,
    padding: '0 4px',
    minWidth: 0,
  },
  connRow: {
    display: 'flex',
    alignItems: 'center',
    gap: 6,
  },
  connDot: {
    width: 7,
    height: 7,
    borderRadius: '50%',
    flexShrink: 0,
  },
  connLabel: {
    fontSize: 11,
    color: 'var(--text-muted)',
  },
  section: {
    display: 'flex',
    flexDirection: 'column',
    gap: 6,
  },
  sectionLabel: {
    fontSize: 10,
    color: 'var(--text-muted)',
    textTransform: 'uppercase',
    letterSpacing: 1,
    fontWeight: 600,
  },
  emptyPile: {
    color: 'var(--text-muted)',
    fontSize: 20,
  },
  drawPile: {
    position: 'relative',
    width: 46,
    height: 68,
  },
  drawFaceDown: {
    width: 46,
    height: 68,
    borderRadius: 6,
    background: 'linear-gradient(135deg, #1e1e2e, #2a2a3f)',
    border: '2px solid var(--border)',
    boxShadow: '0 2px 8px rgba(0,0,0,0.4)',
  },
  drawCount: {
    position: 'absolute',
    bottom: -16,
    left: '50%',
    transform: 'translateX(-50%)',
    fontFamily: 'JetBrains Mono, monospace',
    fontSize: 11,
    color: 'var(--text-muted)',
  },
  phaseText: {
    fontSize: 14,
    fontWeight: 600,
  },
  comboBox: {
    background: 'rgba(212,172,13,0.12)',
    border: '1px solid #d4ac0d',
    borderRadius: 8,
    padding: '8px 14px',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  comboLabel: {
    fontSize: 12,
    color: '#d4ac0d',
  },
  comboCount: {
    fontSize: 22,
    fontWeight: 700,
    color: '#d4ac0d',
    fontFamily: 'Cinzel, serif',
  },
  yourTurn: {
    display: 'flex',
    alignItems: 'center',
    gap: 6,
    background: 'var(--active-glow)',
    border: '1px solid var(--active)',
    borderRadius: 8,
    padding: '8px 12px',
  },
  turnDot: {
    color: 'var(--active)',
    fontSize: 10,
    display: 'inline-block',
  },
  yourTurnText: {
    color: 'var(--active)',
    fontSize: 13,
    fontWeight: 600,
  },
  turnCount: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  turnNum: {
    fontFamily: 'JetBrains Mono, monospace',
    fontSize: 14,
    color: 'var(--text)',
    fontWeight: 600,
  },
};
