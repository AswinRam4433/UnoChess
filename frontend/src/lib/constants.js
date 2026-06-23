export const API_BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';
export const WS_BASE  = API_BASE.replace(/^http/, 'ws');

export const UNO_COLORS = ['RED', 'BLUE', 'GREEN', 'YELLOW'];

export const COLOR_HEX = {
  RED:    '#c0392b',
  BLUE:   '#2471a3',
  GREEN:  '#1e8449',
  YELLOW: '#d4ac0d',
  WILD:   null,
};

export const PHASE = {
  AWAITING_CARD:        'AwaitingCard',
  IN_COMBO:             'InCombo',
  AWAITING_RESURRECTION:'AwaitingResurrection',
  TURN_COMPLETE:        'TurnComplete',
  GAME_OVER:            'GameOver',
};
