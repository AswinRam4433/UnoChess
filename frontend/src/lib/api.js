import { API_BASE } from './constants';

async function req(path, method = 'GET', body = null) {
  const opts = { method, headers: { 'Content-Type': 'application/json' } };
  if (body) opts.body = JSON.stringify(body);
  const r = await fetch(API_BASE + path, opts);
  const data = await r.json();
  if (!r.ok) throw Object.assign(new Error(data.message ?? 'Request failed'), { code: data.code, status: r.status });
  return data;
}

export const api = {
  createGame: (opponent) => req('/games', 'POST', { opponent }),
  joinGame:   (gameID)   => req(`/games/${gameID}/join`, 'POST', {}),
  getGame:    (gameID)   => req(`/games/${gameID}`),
  listGames:  ()         => req('/games'),
};
