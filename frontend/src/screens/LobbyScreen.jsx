import { useState, useEffect, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { api } from '../lib/api';
import ShareDialog from '../components/overlays/ShareDialog';

export default function LobbyScreen({ onGameReady }) {
  const [opponent, setOpponent]       = useState('human');
  const [joinID,   setJoinID]         = useState('');
  const [creating, setCreating]       = useState(false);
  const [joining,  setJoining]        = useState(false);
  const [error,    setError]          = useState(null);
  const [openGames, setOpenGames]     = useState([]);
  const [shareData, setShareData]     = useState(null); // { gameID, token }

  // Poll open games every 5 s
  const fetchOpenGames = useCallback(async () => {
    try {
      const { games } = await api.listGames();
      setOpenGames(games ?? []);
    } catch { /* ignore */ }
  }, []);

  useEffect(() => {
    fetchOpenGames();
    const id = setInterval(fetchOpenGames, 5000);
    return () => clearInterval(id);
  }, [fetchOpenGames]);

  async function handleCreate() {
    setCreating(true);
    setError(null);
    try {
      const data = await api.createGame(opponent);
      if (opponent === 'bot') {
        onGameReady({ gameID: data.gameID, token: data.playerToken, color: data.playerColor });
      } else {
        setShareData({ gameID: data.gameID, token: data.playerToken, color: data.playerColor });
      }
    } catch (e) {
      setError(e.message);
    } finally {
      setCreating(false);
    }
  }

  async function handleJoin(id) {
    const gameID = id ?? joinID.trim();
    if (!gameID) return;
    setJoining(true);
    setError(null);
    try {
      const data = await api.joinGame(gameID);
      onGameReady({ gameID, token: data.playerToken, color: data.playerColor });
    } catch (e) {
      setError(e.message);
    } finally {
      setJoining(false);
    }
  }

  function handleShareConfirm(gameData) {
    // White waits for Black to join; navigate to game immediately
    onGameReady(gameData);
  }

  return (
    <motion.div
      style={styles.root}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      {/* Nav */}
      <header style={styles.nav}>
        <div style={styles.navBrand}>
          <img src="/logo.svg" alt="" style={styles.navLogo} />
          <span style={{ ...styles.navTitle, fontFamily: 'Cinzel, serif' }}>UN♟CHESS</span>
        </div>
      </header>

      <main style={styles.main}>
        <div style={styles.cards}>

          {/* Create game card */}
          <motion.div
            style={styles.card}
            whileHover={{ borderColor: 'var(--active)', boxShadow: '0 0 0 1px var(--active-glow)' }}
            transition={{ duration: 0.15 }}
          >
            <h2 style={styles.cardTitle}>New Game</h2>

            <div style={styles.radioGroup}>
              {['human', 'bot'].map(opt => (
                <label key={opt} style={styles.radioLabel}>
                  <input
                    type="radio"
                    name="opponent"
                    value={opt}
                    checked={opponent === opt}
                    onChange={() => setOpponent(opt)}
                    style={styles.radioInput}
                  />
                  <span style={styles.radioText}>
                    vs {opt.charAt(0).toUpperCase() + opt.slice(1)}
                  </span>
                </label>
              ))}
            </div>

            <motion.button
              style={styles.btn}
              onClick={handleCreate}
              disabled={creating}
              whileTap={{ scale: 0.97 }}
              whileHover={{ background: '#8e7ef9' }}
            >
              {creating ? 'Creating…' : 'Create Game'}
            </motion.button>
          </motion.div>

          {/* Join game card */}
          <motion.div
            style={styles.card}
            whileHover={{ borderColor: 'var(--active)', boxShadow: '0 0 0 1px var(--active-glow)' }}
            transition={{ duration: 0.15 }}
          >
            <h2 style={styles.cardTitle}>Join Game</h2>

            <input
              style={styles.input}
              placeholder="Paste game ID…"
              value={joinID}
              onChange={e => setJoinID(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleJoin()}
              spellCheck={false}
            />

            <motion.button
              style={styles.btn}
              onClick={() => handleJoin()}
              disabled={joining || !joinID.trim()}
              whileTap={{ scale: 0.97 }}
              whileHover={{ background: '#8e7ef9' }}
            >
              {joining ? 'Joining…' : 'Join →'}
            </motion.button>
          </motion.div>
        </div>

        {/* Error */}
        <AnimatePresence>
          {error && (
            <motion.p
              style={styles.error}
              initial={{ opacity: 0, y: -8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0 }}
            >
              {error}
            </motion.p>
          )}
        </AnimatePresence>

        {/* Open games list */}
        {openGames.length > 0 && (
          <div style={styles.openGames}>
            <h3 style={styles.openTitle}>Open Games</h3>
            <div style={styles.openList}>
              <AnimatePresence initial={false}>
                {openGames.map(id => (
                  <motion.div
                    key={id}
                    style={styles.openRow}
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0 }}
                    layout
                  >
                    <span style={styles.openID}>{id}</span>
                    <span style={styles.openTag}>Waiting for opponent</span>
                    <motion.button
                      style={styles.joinBtn}
                      onClick={() => handleJoin(id)}
                      whileTap={{ scale: 0.97 }}
                    >
                      Join →
                    </motion.button>
                  </motion.div>
                ))}
              </AnimatePresence>
            </div>
          </div>
        )}
      </main>

      {/* Share dialog */}
      <AnimatePresence>
        {shareData && (
          <ShareDialog
            gameID={shareData.gameID}
            onConfirm={() => handleShareConfirm(shareData)}
            onClose={() => setShareData(null)}
          />
        )}
      </AnimatePresence>
    </motion.div>
  );
}

const styles = {
  root: {
    height: '100vh',
    display: 'flex',
    flexDirection: 'column',
    background: 'var(--bg)',
    overflow: 'auto',
  },
  nav: {
    height: 52,
    borderBottom: '1px solid var(--border)',
    display: 'flex',
    alignItems: 'center',
    padding: '0 24px',
    flexShrink: 0,
  },
  navBrand: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
  },
  navLogo: {
    width: 28,
    height: 35,
  },
  navTitle: {
    fontSize: 18,
    fontWeight: 600,
    color: 'var(--text)',
    letterSpacing: 1,
  },
  main: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 32,
    gap: 28,
  },
  cards: {
    display: 'flex',
    gap: 20,
    flexWrap: 'wrap',
    justifyContent: 'center',
  },
  card: {
    background: 'var(--surface)',
    border: '1px solid var(--border)',
    borderRadius: 12,
    padding: 28,
    width: 240,
    display: 'flex',
    flexDirection: 'column',
    gap: 16,
    transition: 'border-color 0.15s, box-shadow 0.15s',
  },
  cardTitle: {
    fontFamily: 'Cinzel, serif',
    fontSize: 16,
    fontWeight: 600,
    color: 'var(--text)',
    marginBottom: 4,
  },
  radioGroup: {
    display: 'flex',
    flexDirection: 'column',
    gap: 10,
  },
  radioLabel: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    cursor: 'pointer',
  },
  radioInput: {
    accentColor: 'var(--active)',
    width: 16,
    height: 16,
  },
  radioText: {
    color: 'var(--text)',
    fontSize: 14,
  },
  btn: {
    background: 'var(--active)',
    color: '#fff',
    border: 'none',
    borderRadius: 8,
    padding: '10px 0',
    fontSize: 14,
    fontWeight: 600,
    width: '100%',
    transition: 'background 0.15s',
    cursor: 'pointer',
  },
  input: {
    background: 'var(--bg)',
    border: '1px solid var(--border)',
    borderRadius: 6,
    padding: '8px 12px',
    color: 'var(--text)',
    fontSize: 13,
    fontFamily: 'JetBrains Mono, monospace',
    width: '100%',
  },
  error: {
    color: '#e74c3c',
    fontSize: 13,
    textAlign: 'center',
  },
  openGames: {
    width: '100%',
    maxWidth: 520,
  },
  openTitle: {
    fontFamily: 'Cinzel, serif',
    fontSize: 13,
    fontWeight: 600,
    color: 'var(--text-muted)',
    letterSpacing: 1,
    textTransform: 'uppercase',
    marginBottom: 10,
  },
  openList: {
    display: 'flex',
    flexDirection: 'column',
    gap: 6,
  },
  openRow: {
    background: 'var(--surface)',
    border: '1px solid var(--border)',
    borderRadius: 8,
    padding: '10px 16px',
    display: 'flex',
    alignItems: 'center',
    gap: 12,
  },
  openID: {
    fontFamily: 'JetBrains Mono, monospace',
    fontSize: 12,
    color: 'var(--text-muted)',
    flex: 1,
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  },
  openTag: {
    fontSize: 12,
    color: 'var(--text-muted)',
  },
  joinBtn: {
    background: 'transparent',
    border: '1px solid var(--active)',
    borderRadius: 6,
    color: 'var(--active)',
    padding: '4px 12px',
    fontSize: 12,
    fontWeight: 600,
    cursor: 'pointer',
  },
};
