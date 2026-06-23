import { useState, useCallback } from 'react';
import { AnimatePresence } from 'framer-motion';
import LoadingScreen from './screens/LoadingScreen';
import LobbyScreen from './screens/LobbyScreen';
import GameScreen from './screens/GameScreen';

export default function App() {
  const [screen,   setScreen]   = useState('loading');
  const [gameData, setGameData] = useState(null);

  const goLobby   = useCallback(() => { setGameData(null); setScreen('lobby'); }, []);
  const goGame    = useCallback((data) => { setGameData(data); setScreen('game'); }, []);
  const goRematch = useCallback(() => { setGameData(null); setScreen('lobby'); }, []);

  return (
    <AnimatePresence mode="wait">
      {screen === 'loading' && (
        <LoadingScreen key="loading" onDone={() => setScreen('lobby')} />
      )}
      {screen === 'lobby' && (
        <LobbyScreen key="lobby" onGameReady={goGame} />
      )}
      {screen === 'game' && gameData && (
        <GameScreen
          key="game"
          gameData={gameData}
          onRematch={goRematch}
          onLobby={goLobby}
        />
      )}
    </AnimatePresence>
  );
}
