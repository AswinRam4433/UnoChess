import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import UnoCard from './UnoCard';
import WildColorPicker from './overlays/WildColorPicker';
import { PHASE } from '../lib/constants';

function isPlayable(card, discardTop, phase) {
  if (phase !== PHASE.AWAITING_CARD) return false;
  if (!discardTop?.value) return true;
  if (card.color === 'WILD') return true;
  return card.color === discardTop.color || card.value === discardTop.value;
}

export default function UnoHand({ hand, discardTop, phase, isMyTurn, onPlayCard, onDraw, drawPending }) {
  const [selected,       setSelected]       = useState(null); // card index
  const [awaitingColor,  setAwaitingColor]  = useState(false);

  const canPlay = phase === PHASE.AWAITING_CARD && isMyTurn;
  // The server only permits a draw when you hold no playable card (DrawForTurn
  // returns ErrHasPlayableCard otherwise), so derive the affordance the same way
  // rather than tracking turn boundaries — the latter is unreliable because React
  // batches consecutive WS messages and can collapse the opponent's turn out of
  // the committed render, defeating any activeColor edge-detection.
  const hasPlayable = canPlay && hand.some(c => isPlayable(c, discardTop, phase));
  const canDraw = canPlay && !hasPlayable;

  function handleCardClick(i) {
    if (!canPlay) return;
    if (selected === i) { setSelected(null); return; }
    setSelected(i);
  }

  function handlePlay() {
    if (selected === null) return;
    const card = hand[selected];
    if (card.color === 'WILD') {
      setAwaitingColor(true);
    } else {
      onPlayCard(card, null);
      setSelected(null);
    }
  }

  function handleColorPick(color) {
    const card = hand[selected];
    setAwaitingColor(false);
    setSelected(null);
    onPlayCard(card, color);
  }

  return (
    <div style={styles.root}>
      {/* Cards row */}
      <div style={styles.handRow}>
        <AnimatePresence initial={false}>
          {hand.map((card, i) => {
            const playable = canPlay && isPlayable(card, discardTop, phase);
            return (
              <motion.div
                key={`${card.value}-${card.color}-${i}`}
                initial={{ x: 40, opacity: 0 }}
                animate={{ x: 0, opacity: 1 }}
                exit={{ y: -60, opacity: 0 }}
                transition={{ type: 'spring', stiffness: 300, damping: 28, delay: i * 0.03 }}
              >
                <UnoCard
                  card={card}
                  selected={selected === i}
                  playable={playable}
                  onClick={() => handleCardClick(i)}
                />
              </motion.div>
            );
          })}
        </AnimatePresence>
      </div>

      {/* Action bar */}
      <div style={styles.actions}>
        {selected !== null && canPlay && (
          <motion.button
            style={styles.playBtn}
            onClick={handlePlay}
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            whileTap={{ scale: 0.96 }}
          >
            Play card
          </motion.button>
        )}
        {canDraw && (
          <motion.button
            style={{ ...styles.drawBtn, ...(drawPending ? styles.drawBtnPending : {}) }}
            onClick={drawPending ? undefined : onDraw}
            disabled={drawPending}
            whileTap={drawPending ? {} : { scale: 0.96 }}
            whileHover={drawPending ? {} : { borderColor: 'var(--text-muted)' }}
          >
            {drawPending ? 'Drawing…' : 'Draw card'}
          </motion.button>
        )}
        {!isMyTurn && (
          <span style={styles.waitMsg}>Waiting for opponent…</span>
        )}
        {isMyTurn && phase === PHASE.IN_COMBO && (
          <span style={styles.comboMsg}>♟ Make your chess move on the board</span>
        )}
        {isMyTurn && phase === PHASE.AWAITING_RESURRECTION && (
          <span style={styles.comboMsg}>Resurrection triggered — see overlay</span>
        )}
      </div>

      {/* Wild color picker */}
      <AnimatePresence>
        {awaitingColor && (
          <div style={styles.pickerWrap}>
            <WildColorPicker onPick={handleColorPick} />
          </div>
        )}
      </AnimatePresence>
    </div>
  );
}

const styles = {
  root: {
    position: 'relative',
    display: 'flex',
    flexDirection: 'column',
    gap: 6,
    padding: '4px 0 2px',
  },
  handRow: {
    display: 'flex',
    flexWrap: 'nowrap',
    gap: 6,
    overflowX: 'auto',
    paddingBottom: 4,
    paddingTop: 16,
    paddingLeft: 4,
    scrollbarWidth: 'thin',
  },
  actions: {
    display: 'flex',
    gap: 8,
    alignItems: 'center',
    paddingLeft: 4,
    flexWrap: 'wrap',
  },
  playBtn: {
    background: 'var(--active)',
    color: '#fff',
    border: 'none',
    borderRadius: 8,
    padding: '8px 20px',
    fontWeight: 600,
    fontSize: 13,
    cursor: 'pointer',
  },
  drawBtn: {
    background: 'transparent',
    color: 'var(--text)',
    border: '1px solid var(--border)',
    borderRadius: 8,
    padding: '8px 18px',
    fontSize: 13,
    fontWeight: 500,
    cursor: 'pointer',
    transition: 'border-color 0.15s',
  },
  drawBtnPending: {
    opacity: 0.5,
    cursor: 'default',
  },
  waitMsg: {
    fontSize: 12,
    color: 'var(--text-muted)',
    fontStyle: 'italic',
  },
  comboMsg: {
    fontSize: 12,
    color: '#7c6af7',
    fontWeight: 500,
  },
  pickerWrap: {
    position: 'absolute',
    bottom: '110%',
    left: '50%',
    transform: 'translateX(-50%)',
    zIndex: 60,
  },
};
