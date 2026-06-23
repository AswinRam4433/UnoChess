import { useState, useRef, useEffect } from 'react';
import { Chessboard } from 'react-chessboard';
import { Chess } from 'chess.js';
import { motion, AnimatePresence } from 'framer-motion';
import { PHASE } from '../lib/constants';

/*
 * react-chessboard v5: all props go inside a single `options` object.
 * Board sizes itself to 100% of its CSS container — no boardWidth prop.
 *
 * Key design: we keep a localFen ref + displayFen state so the board
 * updates after *each* sub-move without waiting for the server round-trip.
 * Legal-move highlights for move N+1 are computed from the post-move-N
 * position, not the stale server FEN.
 *
 * localFenRef  – mutable, used inside callbacks (no stale-closure problem)
 * displayFen   – React state, drives the board render
 *
 * When the server sends a new authoritative FEN (phase change, opponent move,
 * combo end) both are synced from the `fen` prop via useEffect.
 */
const boardLog = (...args) => console.log('[BOARD]', ...args);


export default function ChessBoardWrapper({ fen, phase, isMyTurn, yourColor, onSubMove, pendingCombo }) {
  const [selectedSq,   setSelectedSq]   = useState(null);
  const [legalSquares, setLegalSquares] = useState({});
  const [promotion,    setPromotion]    = useState(null); // { from, to }
  const [shaking,      setShaking]      = useState(false);
  const [displayFen,   setDisplayFen]   = useState(fen);
  const localFenRef = useRef(fen);

  // Sync local position from server whenever the authoritative FEN changes
  // (turn ends, opponent plays, combo finishes, etc.)
  useEffect(() => {
    boardLog(
      `server FEN updated  phase=${phase}  isMyTurn=${isMyTurn}` +
      `  movesLeft=${pendingCombo?.movesRemaining ?? '-'}`,
    );
    boardLog(`  old localFen: ${localFenRef.current}`);
    boardLog(`  new serverFen: ${fen}`);
    localFenRef.current = fen;
    setDisplayFen(fen);
  }, [fen]);

  const inCombo      = phase === PHASE.IN_COMBO && isMyTurn;
  const myChessColor = yourColor === 'White' ? 'w' : 'b';
  const boardOrientation = yourColor?.toLowerCase() === 'black' ? 'black' : 'white';

  // Always build from the current LOCAL position so move-N+1 is legal
  function getChess() {
    try { return new Chess(localFenRef.current); } catch { return null; }
  }

  // Apply a sub-move optimistically: update local board immediately,
  // then tell the server. Returns true on success, false on illegal move.
  //
  // If chess.js can't parse the current FEN (both kings must be present),
  // that's an invalid game state — the server should have already sent game_over.
  // We block moves and let the server resolve the situation.
  function applySubMove(from, to, promotionPiece) {
    boardLog(`applySubMove  ${from}→${to}${promotionPiece ? `=${promotionPiece}` : ''}  localFen="${localFenRef.current}"`);
    const chess = getChess();
    if (!chess) {
      boardLog('  BLOCKED: chess.js cannot parse this FEN (king missing?) — waiting for server game_over');
      return false;
    }

    try {
      chess.move({ from, to, ...(promotionPiece ? { promotion: promotionPiece } : {}) });
    } catch (err) {
      boardLog(`  FAIL: chess.move threw → ${err.message}`);
      // King-capture exception: chess.js (standard chess) rejects king captures
      // because the game normally ends at checkmate first. In UnoChess, capturing
      // the king during a multi-move combo is a winning move — forward it to the
      // server for validation rather than blocking it here.
      const movingPiece = chess.get(from);
      const targetPiece = chess.get(to);
      if (movingPiece?.color === myChessColor && targetPiece?.type === 'k') {
        boardLog(`  → king capture: forwarding to server`);
        onSubMove(`${from}${to}${promotionPiece ?? ''}`);
        return true;
      }
      return false;
    }

    // Pin active color so the same player can keep moving during a combo.
    // chess.fen() flips to the opponent's turn after each move, but the server
    // expects moves from the same player for the rest of the combo.
    const fenParts = chess.fen().split(' ');
    fenParts[1] = myChessColor;
    const newFen = fenParts.join(' ');
    boardLog(`  OK: newFen="${newFen}"`);
    localFenRef.current = newFen;
    setDisplayFen(newFen);
    const uci = `${from}${to}${promotionPiece ?? ''}`;
    boardLog(`  → sending sub-move to server: ${uci}`);
    onSubMove(uci);
    return true;
  }

  function shake() {
    setShaking(true);
    setTimeout(() => setShaking(false), 350);
  }

  // Clear selection when we leave combo phase
  useEffect(() => {
    if (!inCombo) {
      setSelectedSq(null);
      setLegalSquares({});
    }
  }, [inCombo]);

  // ── Drag-and-drop ────────────────────────────────────────────────────────────
  function canDragPiece({ piece: { pieceType } }) {
    return inCombo && pieceType[0] === myChessColor;
  }

  function handlePieceDrop({ sourceSquare, targetSquare }) {
    boardLog(`onPieceDrop  ${sourceSquare}→${targetSquare}  inCombo=${inCombo}`);
    if (!inCombo) {
      boardLog('  rejected: not in combo');
      return false;
    }

    const chess = getChess();
    const piece = chess?.get(sourceSquare);
    boardLog(`  piece at ${sourceSquare}:`, piece);

    // Pawn promotion — show picker, don't move yet
    if (piece?.type === 'p' &&
      ((piece.color === 'w' && targetSquare[1] === '8') ||
       (piece.color === 'b' && targetSquare[1] === '1'))
    ) {
      setPromotion({ from: sourceSquare, to: targetSquare });
      setSelectedSq(null);
      setLegalSquares({});
      return false;
    }

    const ok = applySubMove(sourceSquare, targetSquare);
    if (!ok) { shake(); return false; }
    setSelectedSq(null);
    setLegalSquares({});
    return true;
  }

  // ── Click-to-move ────────────────────────────────────────────────────────────
  function handleSquareClick({ square }) {
    boardLog(`onSquareClick  square=${square}  selectedSq=${selectedSq}  inCombo=${inCombo}`);
    if (!inCombo) return;
    const chess = getChess(); // may be null in non-standard positions

    if (selectedSq) {
      const from  = selectedSq;
      const to    = square;
      const piece = chess?.get(from);

      // Pawn promotion
      if (piece?.type === 'p' &&
        ((piece.color === 'w' && to[1] === '8') ||
         (piece.color === 'b' && to[1] === '1'))
      ) {
        setPromotion({ from, to });
        setSelectedSq(null);
        setLegalSquares({});
        return;
      }

      const ok = applySubMove(from, to);
      if (ok) {
        setSelectedSq(null);
        setLegalSquares({});
      } else {
        selectSquare(square, chess);
      }
    } else {
      selectSquare(square, chess);
    }
  }

  function selectSquare(sq, chess) {
    const piece = chess?.get(sq);
    if (!piece || piece.color !== myChessColor) {
      setSelectedSq(null);
      setLegalSquares({});
      return;
    }
    const moves = chess.moves({ square: sq, verbose: true });
    if (moves.length === 0) { shake(); return; }

    setSelectedSq(sq);
    const highlights = { [sq]: { backgroundColor: 'color-mix(in srgb, var(--accent) 42%, transparent)' } };
    moves.forEach(m => {
      highlights[m.to] = chess.get(m.to)
        ? { boxShadow: 'inset 0 0 0 4px color-mix(in srgb, var(--accent) 70%, transparent)' }
        : { background: 'radial-gradient(circle, color-mix(in srgb, var(--accent) 55%, transparent) 28%, transparent 30%)' };
    });
    setLegalSquares(highlights);
  }

  // ── Promotion ────────────────────────────────────────────────────────────────
  function handlePromotionPiece(p) {
    if (!promotion) return;
    applySubMove(promotion.from, promotion.to, p);
    setPromotion(null);
  }

  return (
    <div style={{ position: 'relative', width: '100%' }}>
      {/* Turn / combo ring */}
      <motion.div
        style={{
          borderRadius: 10,
          transition: 'box-shadow 0.3s ease',
          boxShadow: inCombo
            ? '0 0 0 3px rgba(216,180,94,0.6)'
            : isMyTurn
              ? '0 0 0 3px var(--accent-glow)'
              : '0 0 0 1px var(--border)',
        }}
        animate={isMyTurn && !inCombo ? {
          boxShadow: [
            '0 0 0 3px rgba(216,180,94,0.5)',
            '0 0 0 6px rgba(216,180,94,0.08)',
            '0 0 0 3px rgba(216,180,94,0.5)',
          ],
        } : {}}
        transition={{ repeat: Infinity, duration: 2 }}
      >
        <div className={shaking ? 'shake' : undefined}>
          <Chessboard
            options={{
              position: displayFen,
              boardOrientation,
              allowDragging: inCombo,
              canDragPiece,
              onPieceDrop: handlePieceDrop,
              onSquareClick: inCombo ? handleSquareClick : undefined,
              squareStyles: legalSquares,
              darkSquareStyle:  { backgroundColor: 'var(--board-dark)' },
              lightSquareStyle: { backgroundColor: 'var(--board-light)' },
              boardStyle: {
                borderRadius: 8,
                boxShadow: '0 4px 32px rgba(0,0,0,0.5)',
                width: '100%',
                aspectRatio: '1 / 1',
              },
              showNotation: true,
            }}
          />
        </div>
      </motion.div>

      {/* Combo counter badge */}
      <AnimatePresence>
        {inCombo && pendingCombo && (
          <motion.div
            style={styles.comboBadge}
            initial={{ opacity: 0, y: 6 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
          >
            ♟ Drag or click a piece —{' '}
            <strong>{pendingCombo.movesRemaining}</strong>{' '}
            move{pendingCombo.movesRemaining !== 1 ? 's' : ''} left
          </motion.div>
        )}
      </AnimatePresence>

      {/* Promotion picker */}
      <AnimatePresence>
        {promotion && (
          <motion.div style={styles.promoBackdrop} initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
            <motion.div style={styles.promoBox} initial={{ scale: 0.85 }} animate={{ scale: 1 }}>
              <p style={styles.promoTitle}>Promote pawn to:</p>
              <div style={styles.promoRow}>
                {[['q','♛'],['r','♜'],['b','♝'],['n','♞']].map(([p, glyph]) => (
                  <motion.button
                    key={p}
                    style={styles.promoBtn}
                    onClick={() => handlePromotionPiece(p)}
                    whileHover={{ background: 'var(--accent)' }}
                    whileTap={{ scale: 0.93 }}
                  >
                    {glyph}
                  </motion.button>
                ))}
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

const styles = {
  comboBadge: {
    marginTop: 8,
    textAlign: 'center',
    fontSize: 12,
    color: '#d4ac0d',
    fontWeight: 500,
  },
  promoBackdrop: {
    position: 'absolute', inset: 0,
    background: 'rgba(0,0,0,0.72)',
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    borderRadius: 8, zIndex: 40,
  },
  promoBox: {
    background: 'var(--surface-r)',
    border: '1px solid var(--border)',
    borderRadius: 12,
    padding: '20px 24px',
    display: 'flex', flexDirection: 'column', gap: 14, alignItems: 'center',
  },
  promoTitle: { fontFamily: 'Cinzel, serif', fontSize: 14, color: 'var(--text)' },
  promoRow:   { display: 'flex', gap: 10 },
  promoBtn: {
    width: 52, height: 52,
    borderRadius: 8,
    background: 'var(--surface)',
    border: '1px solid var(--border)',
    fontSize: 28, cursor: 'pointer', color: 'var(--text)',
    transition: 'background 0.15s',
  },
};
