import { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useWebSocket } from '../hooks/useWebSocket';
import ChessBoardWrapper from '../components/ChessBoardWrapper';
import UnoHand from '../components/UnoHand';
import OpponentArea from '../components/OpponentArea';
import GameInfo from '../components/GameInfo';
import ResurrectionOverlay from '../components/overlays/ResurrectionOverlay';
import GameOverModal from '../components/overlays/GameOverModal';
import { PHASE } from '../lib/constants';

const gsLog = (...args) => console.log('[GAME]', ...args);

export default function GameScreen({ gameData, onRematch, onLobby }) {
  const { gameID, token, color: yourColor } = gameData;
  const { gameState, gameOver, wsStatus, lastError, send, clearError } = useWebSocket(gameID, token);
  // drawPending disables the Draw button between a click and the next server
  // state, to swallow accidental double-draws. It resets on *every* new gameState
  // (keyed on object identity), so batched WS messages can't strand it.
  const [drawPending, setDrawPending] = useState(false);
  const renderCount = useRef(0);

  // Log every React re-render driven by gameState.
  // If the [GAME] render count is lower than the [WS] STATE count, React is
  // batching messages and we're losing intermediate bot positions.
  useEffect(() => {
    if (!gameState) return;
    const n = ++renderCount.current;
    gsLog(
      `render #${n}  active=${gameState.activeColor}  phase=${gameState.phase}` +
      `  movesLeft=${gameState.pendingCombo?.movesRemaining ?? '-'}`,
    );
    gsLog(`  FEN: ${gameState.boardFEN}`);
  }, [gameState]);

  // Any fresh state from the server clears the transient draw lock.
  useEffect(() => { setDrawPending(false); }, [gameState]);

  if (!gameState) {
    return (
      <div style={styles.loading}>
        <motion.div
          animate={{ rotate: 360 }}
          transition={{ repeat: Infinity, duration: 1, ease: 'linear' }}
          style={styles.spinner}
        />
        <span style={styles.loadingText}>
          {wsStatus === 'connecting' ? 'Connecting…' : wsStatus === 'closed' ? 'Disconnected' : 'Waiting for game…'}
        </span>
      </div>
    );
  }

  const isMyTurn      = gameState.activeColor === yourColor;
  const opponentColor = yourColor === 'White' ? 'Black' : 'White';
  const needsResurrection = gameState.phase === PHASE.AWAITING_RESURRECTION && isMyTurn;

  const capturedPool  = { queen: 1, rook: 2, bishop: 2, knight: 2, pawn: 8 };
  const allowedCount  = gameState.phase === PHASE.AWAITING_RESURRECTION
    ? (gameState.discardTop?.value === '+4' ? 4 : 2) : 0;

  function sendPlayCard(card, declaredColor) {
    const cmd = { type: 'play_card', card };
    if (declaredColor) cmd.declaredColor = declaredColor;
    send(cmd);
  }
  function sendDraw() { setDrawPending(true); send({ type: 'draw_for_turn' }); }
  function sendSubMove(uci) { send({ type: 'play_sub_move', uci }); }
  function sendResurrection(placements) {
    send({ type: 'play_resurrection', placements });
  }

  return (
    <motion.div
      style={styles.root}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      {/* ── Nav ─────────────────────────────────── */}
      <header style={styles.nav}>
        <div style={styles.navBrand}>
          <img src="/logo.svg" alt="" style={styles.navLogo} />
          <span style={styles.navTitle}>UN♟CHESS</span>
        </div>

        <div style={styles.navMeta}>
          <PlayerChip color={yourColor} label="You" />
          <span style={styles.navSep}>vs</span>
          <PlayerChip color={opponentColor} label="Opponent" />
        </div>

        <button style={styles.leaveBtn} onClick={onLobby}>Leave</button>
      </header>

      {/* ── Error toast ─────────────────────────── */}
      <AnimatePresence>
        {lastError && (
          <motion.div
            style={styles.toast}
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            onClick={clearError}
          >
            ⚠ {lastError.message}
          </motion.div>
        )}
      </AnimatePresence>

      {/* ── Body ────────────────────────────────── */}
      {/*
        Two-column layout, horizontally centered in the viewport.
        boardCol width drives everything: the ResizeObserver inside
        ChessBoardWrapper reads it and passes it to <Chessboard boardWidth>.
        clamp keeps the board from being too large on tall screens or
        too small on short ones.
      */}
      <div style={styles.body}>
        <div style={styles.content}>

          {/* Left column: opponent → board → hand */}
          <div style={styles.boardCol}>
            <OpponentArea
              opponentHandCount={gameState.opponentHandCount}
              opponentColor={opponentColor}
              isTheirTurn={!isMyTurn && gameState.phase !== PHASE.GAME_OVER}
            />

            <div style={{ position: 'relative', width: '100%' }}>
              <ChessBoardWrapper
                fen={gameState.boardFEN}
                phase={gameState.phase}
                isMyTurn={isMyTurn}
                yourColor={yourColor}
                onSubMove={sendSubMove}
                pendingCombo={gameState.pendingCombo}
              />
              <AnimatePresence>
                {needsResurrection && (
                  <ResurrectionOverlay
                    captured={capturedPool}
                    allowedCount={allowedCount}
                    yourColor={yourColor}
                    onConfirm={sendResurrection}
                  />
                )}
              </AnimatePresence>
            </div>

            <UnoHand
              hand={gameState.yourHand}
              discardTop={gameState.discardTop}
              phase={gameState.phase}
              isMyTurn={isMyTurn}
              onPlayCard={sendPlayCard}
              onDraw={sendDraw}
              drawPending={drawPending}
            />
          </div>

          {/* Right column: game info */}
          <div style={styles.infoCol}>
            <GameInfo
              gameState={gameState}
              yourColor={yourColor}
              wsStatus={wsStatus}
            />
          </div>

        </div>
      </div>

      {/* ── Game over ───────────────────────────── */}
      <AnimatePresence>
        {gameOver && (
          <div style={styles.gameOverLayer}>
            <GameOverModal
              gameOver={gameOver}
              yourColor={yourColor}
              onRematch={onRematch}
              onLobby={onLobby}
            />
          </div>
        )}
      </AnimatePresence>
    </motion.div>
  );
}

function PlayerChip({ color, label }) {
  const isWhite = color === 'White';
  return (
    <span style={styles.navPlayer}>
      <span style={{
        width: 10, height: 10, borderRadius: '50%', display: 'inline-block',
        background: isWhite ? '#ddd' : '#111',
        border: isWhite ? 'none' : '1px solid #555',
        flexShrink: 0,
      }} />
      {label} — {color}
    </span>
  );
}

const styles = {
  root: {
    height: '100vh',
    display: 'flex',
    flexDirection: 'column',
    background: 'var(--bg)',
    overflow: 'hidden',
    position: 'relative',
  },

  /* ── Loading ──────────────────────────────── */
  loading: {
    height: '100vh',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 16,
    background: 'var(--bg)',
  },
  spinner: {
    width: 36, height: 36,
    border: '3px solid var(--border)',
    borderTop: '3px solid var(--active)',
    borderRadius: '50%',
  },
  loadingText: { color: 'var(--text-muted)', fontSize: 14 },

  /* ── Nav ──────────────────────────────────── */
  nav: {
    height: 50,
    borderBottom: '1px solid var(--border)',
    display: 'flex',
    alignItems: 'center',
    padding: '0 20px',
    gap: 16,
    flexShrink: 0,
  },
  navBrand: { display: 'flex', alignItems: 'center', gap: 8 },
  navLogo:  { width: 24, height: 30 },
  navTitle: {
    fontFamily: 'Cinzel, serif',
    fontSize: 16, fontWeight: 600,
    color: 'var(--text)',
  },
  navMeta: {
    flex: 1,
    display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 10,
  },
  navPlayer: {
    display: 'flex', alignItems: 'center', gap: 6,
    fontSize: 13, color: 'var(--text)',
  },
  navSep: { fontSize: 12, color: 'var(--text-muted)' },
  leaveBtn: {
    background: 'transparent',
    border: '1px solid var(--border)',
    color: 'var(--text-muted)',
    borderRadius: 6, padding: '4px 12px', fontSize: 12, cursor: 'pointer',
  },

  /* ── Toast ────────────────────────────────── */
  toast: {
    position: 'absolute', top: 58, left: '50%', transform: 'translateX(-50%)',
    background: '#6a1f1f', border: '1px solid #c0392b', color: '#fff',
    borderRadius: 8, padding: '8px 16px', fontSize: 13,
    zIndex: 90, cursor: 'pointer', whiteSpace: 'nowrap',
  },

  /* ── Body ─────────────────────────────────── */
  body: {
    flex: 1,
    display: 'flex',
    justifyContent: 'center',   // center the content horizontally
    alignItems: 'flex-start',   // anchor to top
    padding: '12px 20px',
    overflow: 'hidden',
  },

  /* The natural-width wrapper that gets centered inside body */
  content: {
    display: 'flex',
    gap: 20,
    alignItems: 'flex-start',
  },

  /* Left column. Its width = board width (picked up by ResizeObserver). */
  boardCol: {
    display: 'flex',
    flexDirection: 'column',
    gap: 8,
    // clamp(min, preferred, max)
    // preferred = available height minus fixed chrome, so the board fits
    // without scrolling. nav=50, bodyPad=24, opponent=56, hand=106, gaps=24
    // → preferred = 100vh - 260px (=board height ≈ board width for square board)
    width: 'clamp(280px, calc(100vh - 260px), 440px)',
    flexShrink: 0,
    overflow: 'visible',        // allow cards to lift on hover
  },

  /* Right column: fixed, sits flush with the top of the board */
  infoCol: {
    width: 168,
    flexShrink: 0,
    display: 'flex',
    flexDirection: 'column',
    gap: 4,
    paddingTop: 56,             // visual alignment: skip opponent row height
    overflowY: 'auto',
  },

  /* ── Game over ────────────────────────────── */
  gameOverLayer: {
    position: 'absolute', inset: 0, zIndex: 80,
    display: 'flex', alignItems: 'center', justifyContent: 'center',
  },
};
