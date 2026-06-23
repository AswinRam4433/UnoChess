import { useEffect, useRef, useCallback, useState } from 'react';
import { WS_BASE } from '../lib/constants';

const log = (...args) => console.log('[WS]', ...args);
const warn = (...args) => console.warn('[WS]', ...args);

export function useWebSocket(gameID, token) {
  const [gameState, setGameState] = useState(null);
  const [gameOver,  setGameOver]  = useState(null);
  const [wsStatus,  setWsStatus]  = useState('connecting');
  const [lastError, setLastError] = useState(null);
  const wsRef     = useRef(null);
  const msgCount  = useRef(0);

  useEffect(() => {
    if (!gameID || !token) return;

    const url = `${WS_BASE}/games/${gameID}/play?token=${token}`;
    log(`connecting → ${url}`);
    const ws = new WebSocket(url);
    wsRef.current = ws;
    setWsStatus('connecting');

    ws.onopen = () => {
      log('connection opened');
      setWsStatus('open');
    };

    ws.onclose = (e) => {
      log(`connection closed  code=${e.code}  wasClean=${e.wasClean}`);
      setWsStatus('closed');
    };

    ws.onerror = (e) => {
      warn('socket error', e);
      setWsStatus('closed');
    };

    ws.onmessage = (e) => {
      let msg;
      try { msg = JSON.parse(e.data); } catch (err) {
        warn('unparseable message:', e.data);
        return;
      }

      const n = ++msgCount.current;
      const ts = new Date().toISOString().slice(11, 23); // HH:MM:SS.mmm

      if (msg.type === 'state') {
        const { activeColor, phase, boardFEN, pendingCombo, yourHand, opponentHandCount } = msg;
        log(
          `#${n} [${ts}] STATE  active=${activeColor}  phase=${phase}` +
          `  movesLeft=${pendingCombo?.movesRemaining ?? '-'}` +
          `  handSize=${yourHand?.length ?? '?'}  oppHand=${opponentHandCount ?? '?'}`,
        );
        console.log(`        FEN: ${boardFEN}`);
        setGameState(msg);

      } else if (msg.type === 'game_over') {
        log(`#${n} [${ts}] GAME_OVER  winner=${msg.winner}  reason=${msg.reason}`);
        setGameOver(msg);

      } else if (msg.type === 'error') {
        warn(`#${n} [${ts}] ERROR  code=${msg.code}  message=${msg.message}`);
        setLastError(msg);

      } else {
        log(`#${n} [${ts}] UNKNOWN  type=${msg.type}`, msg);
      }
    };

    return () => {
      log('closing socket (cleanup)');
      ws.close();
    };
  }, [gameID, token]);

  const send = useCallback((msg) => {
    const ws = wsRef.current;
    if (ws?.readyState === WebSocket.OPEN) {
      log('→ SEND', JSON.stringify(msg));
      ws.send(JSON.stringify(msg));
    } else {
      warn('send attempted but socket not open  readyState=', ws?.readyState);
    }
  }, []);

  const clearError = useCallback(() => setLastError(null), []);

  return { gameState, gameOver, wsStatus, lastError, send, clearError };
}
