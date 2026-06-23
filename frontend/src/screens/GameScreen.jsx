import { useState, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useWebSocket } from '../hooks/useWebSocket';
import ChessBoardWrapper from '../components/ChessBoardWrapper';
import UnoHand from '../components/UnoHand';
import OpponentArea from '../components/OpponentArea';
import GameInfo from '../components/GameInfo';
import ThemeToggle from '../components/ThemeToggle';
import ResurrectionOverlay from '../components/overlays/ResurrectionOverlay';
import GameOverModal from '../components/overlays/GameOverModal';
import { PHASE } from '../lib/constants';

const gsLog = (...args) => console.log('[GAME]', ...args);

const PHASE_LABELS = {
  [PHASE.AWAITING_CARD]:         'Play a card',
  [PHASE.IN_COMBO]:              'Make your chess move',
  [PHASE.AWAITING_RESURRECTION]: 'Resurrection',
  [PHASE.TURN_COMPLETE]:         'Advancing…',
  [PHASE.GAME_OVER]:             'Game over',
};

// Simple responsive hook — single breakpoint at 820px.
function useIsMobile(bp = 820) {
  const [m, setM] = useState(() => typeof window !== 'undefined' && window.innerWidth < bp);
  useEffect(() => {
    const onResize = () => setM(window.innerWidth < bp);
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, [bp]);
  return m;
}

export default function GameScreen({ gameData, onRematch, onLobby }) {
  const { gameID, token, color: yourColor } = gameData;
  const { gameState, gameOver, wsStatus, lastError, send, clearError } = useWebSocket(gameID, token);
  const [drawPending, setDrawPending] = useState(false);
  const renderCount = useRef(0);
  const isMobile = useIsMobile();

  useEffect(() => {
    if (!gameState) return;
    const n = ++renderCount.current;
    gsLog(`render #${n}  active=${gameState.activeColor}  phase=${gameState.phase}` +
      `  movesLeft=${gameState.pendingCombo?.movesRemaining ?? '-'}`);
    gsLog(`  FEN: ${gameState.boardFEN}`);
  }, [gameState]);

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
  const gameEnded     = gameState.phase === PHASE.GAME_OVER;

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
  function sendResurrection(placements) { send({ type: 'play_resurrection', placements }); }

  const board = (
    <div style={{ position: 'relative', width: '100%' }}>
      <div style={styles.boardFrame}>
        <ChessBoardWrapper
          fen={gameState.boardFEN}
          phase={gameState.phase}
          isMyTurn={isMyTurn}
          yourColor={yourColor}
          onSubMove={sendSubMove}
          pendingCombo={gameState.pendingCombo}
        />
      </div>
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
  );

  const hand = (
    <UnoHand
      hand={gameState.yourHand}
      discardTop={gameState.discardTop}
      phase={gameState.phase}
      isMyTurn={isMyTurn}
      onPlayCard={sendPlayCard}
      onDraw={sendDraw}
      drawPending={drawPending}
    />
  );

  return (
    <motion.div
      style={styles.root}
      initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
    >
      {/* ── Topbar ─────────────────────────────── */}
      <header style={styles.nav}>
        <div style={styles.navBrand}>
          <img src="/logo.svg" alt="" style={styles.navLogo} />
          {!isMobile && <span style={styles.navTitle}>UN♟CHESS</span>}
        </div>

        {!isMobile && (
          <div style={styles.navMeta}>
            <PlayerChip color={yourColor} label="You" />
            <span style={styles.navSep}>vs</span>
            <PlayerChip color={opponentColor} label="Opponent" />
          </div>
        )}

        <div style={styles.navRight}>
          <PhaseBadge
            phase={gameState.phase}
            isMyTurn={isMyTurn}
            wsStatus={wsStatus}
            ended={gameEnded}
          />
          <ThemeToggle />
          <button style={styles.leaveBtn} onClick={onLobby}>Leave</button>
        </div>
      </header>

      {/* ── Error toast ────────────────────────── */}
      <AnimatePresence>
        {lastError && (
          <motion.div
            style={styles.toast}
            initial={{ opacity: 0, y: -20 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -20 }}
            onClick={clearError}
          >
            ⚠ {lastError.message}
          </motion.div>
        )}
      </AnimatePresence>

      {/* ── Felt play surface ──────────────────── */}
      <div style={isMobile ? styles.bodyMobile : styles.body}>
        {isMobile ? (
          <div style={styles.stack}>
            <OpponentArea
              opponentHandCount={gameState.opponentHandCount}
              opponentColor={opponentColor}
              isTheirTurn={!isMyTurn && !gameEnded}
            />
            {board}
            <GameInfo gameState={gameState} layout="bar" />
            {hand}
          </div>
        ) : (
          <div style={styles.content}>
            <div style={styles.boardCol}>
              <OpponentArea
                opponentHandCount={gameState.opponentHandCount}
                opponentColor={opponentColor}
                isTheirTurn={!isMyTurn && !gameEnded}
              />
              {board}
              {hand}
            </div>
            <aside style={styles.dock}>
              <GameInfo gameState={gameState} layout="rail" />
            </aside>
          </div>
        )}
      </div>

      {/* ── Game over ──────────────────────────── */}
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
        width: 11, height: 11, borderRadius: '50%', display: 'inline-block',
        background: isWhite ? '#f1ead9' : '#1a130c',
        border: isWhite ? '1px solid rgba(0,0,0,0.15)' : '1px solid rgba(255,255,255,0.25)',
        flexShrink: 0,
      }} />
      {label} — {color}
    </span>
  );
}

function PhaseBadge({ phase, isMyTurn, wsStatus, ended }) {
  const disconnected = wsStatus !== 'open';
  const active = isMyTurn && !ended && !disconnected;
  const label = disconnected
    ? (wsStatus === 'connecting' ? 'Connecting…' : 'Disconnected')
    : ended ? 'Game over'
    : isMyTurn ? (PHASE_LABELS[phase] ?? phase)
    : 'Opponent’s turn';

  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 8,
      padding: '6px 14px', borderRadius: 20,
      background: active ? 'var(--accent)' : 'var(--panel)',
      border: active ? '1px solid var(--accent)' : '1px solid var(--border)',
      color: active ? 'var(--accent-ink)' : disconnected ? 'var(--uno-red)' : 'var(--text-muted)',
      fontWeight: 700, fontSize: 12.5, whiteSpace: 'nowrap',
    }}>
      <motion.span
        animate={active ? { scale: [1, 1.4, 1] } : {}}
        transition={{ repeat: Infinity, duration: 1.4 }}
        style={{
          width: 7, height: 7, borderRadius: '50%', display: 'inline-block',
          background: active ? 'var(--accent-ink)' : disconnected ? 'var(--uno-red)' : 'var(--text-muted)',
        }}
      />
      {label}
    </div>
  );
}

const styles = {
  root: {
    minHeight: '100dvh',
    display: 'flex',
    flexDirection: 'column',
    background: 'var(--bg)',
    position: 'relative',
  },

  loading: {
    minHeight: '100dvh',
    display: 'flex', flexDirection: 'column',
    alignItems: 'center', justifyContent: 'center',
    gap: 16, background: 'var(--bg)',
  },
  spinner: {
    width: 36, height: 36,
    border: '3px solid var(--border)',
    borderTop: '3px solid var(--accent)',
    borderRadius: '50%',
  },
  loadingText: { color: 'var(--text-muted)', fontSize: 14 },

  /* ── Topbar ──────────────────────────────── */
  nav: {
    minHeight: 56,
    background: 'var(--bar)',
    borderBottom: '1px solid var(--border)',
    display: 'flex', alignItems: 'center',
    padding: '0 16px', gap: 14, flexShrink: 0,
  },
  navBrand: { display: 'flex', alignItems: 'center', gap: 9 },
  navLogo:  { width: 26, height: 32 },
  navTitle: {
    fontFamily: 'Cinzel, serif', fontSize: 17, fontWeight: 600,
    color: 'var(--text)', letterSpacing: 0.5,
  },
  navMeta: { flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 10 },
  navPlayer: { display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, color: 'var(--text)' },
  navSep: { fontSize: 12, color: 'var(--text-muted)' },
  navRight: { marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 10 },
  leaveBtn: {
    background: 'transparent', border: '1px solid var(--border)',
    color: 'var(--text-muted)', borderRadius: 9, padding: '7px 14px',
    fontSize: 12.5, fontWeight: 600,
  },

  toast: {
    position: 'absolute', top: 66, left: '50%', transform: 'translateX(-50%)',
    background: 'var(--uno-red)', color: '#fff',
    borderRadius: 10, padding: '9px 18px', fontSize: 13, fontWeight: 600,
    zIndex: 90, cursor: 'pointer', whiteSpace: 'nowrap',
    boxShadow: '0 8px 24px rgba(0,0,0,0.4)',
  },

  /* ── Felt body (desktop) ─────────────────── */
  body: {
    flex: 1,
    display: 'flex', justifyContent: 'center', alignItems: 'flex-start',
    padding: '26px 24px',
    background: 'var(--felt)',
    boxShadow: 'inset 0 0 150px rgba(0,0,0,0.42), inset 0 0 0 1px var(--border)',
    overflow: 'hidden',
  },
  content: { display: 'flex', gap: 30, alignItems: 'flex-start' },
  boardCol: {
    display: 'flex', flexDirection: 'column', gap: 10,
    width: 'clamp(300px, calc(100vh - 240px), 460px)',
    flexShrink: 0,
    overflow: 'visible',
  },
  dock: {
    width: 168, flexShrink: 0,
    display: 'flex', flexDirection: 'column',
    paddingTop: 52,
  },

  /* ── Felt body (mobile) ──────────────────── */
  bodyMobile: {
    flex: 1,
    background: 'var(--felt)',
    boxShadow: 'inset 0 0 120px rgba(0,0,0,0.42)',
    padding: '14px 12px 22px',
    overflowY: 'auto',
  },
  stack: {
    display: 'flex', flexDirection: 'column', gap: 14,
    maxWidth: 520, margin: '0 auto',
  },

  /* Wooden frame around the board */
  boardFrame: {
    background: 'var(--board-frame)',
    padding: '2.8%',
    borderRadius: 14,
    boxShadow: '0 14px 36px rgba(0,0,0,0.45), inset 0 0 0 1px rgba(255,255,255,0.06)',
  },

  gameOverLayer: {
    position: 'absolute', inset: 0, zIndex: 80,
    display: 'flex', alignItems: 'center', justifyContent: 'center',
  },
};
