export const API_BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';
export const WS_BASE  = API_BASE.replace(/^http/, 'ws');

export const UNO_COLORS = ['RED', 'BLUE', 'GREEN', 'YELLOW'];

export const COLOR_HEX = {
  RED:    'var(--uno-red)',
  BLUE:   'var(--uno-blue)',
  GREEN:  'var(--uno-green)',
  YELLOW: 'var(--uno-yellow)',
  WILD:   null,
};

export const PHASE = {
  AWAITING_CARD:        'AwaitingCard',
  IN_COMBO:             'InCombo',
  AWAITING_RESURRECTION:'AwaitingResurrection',
  TURN_COMPLETE:        'TurnComplete',
  GAME_OVER:            'GameOver',
};
