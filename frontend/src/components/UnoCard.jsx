import { motion } from 'framer-motion';

// Card body colour (the printed face)
const BG = {
  RED:    'var(--uno-red)',
  BLUE:   'var(--uno-blue)',
  GREEN:  'var(--uno-green)',
  YELLOW: 'var(--uno-yellow)',
  WILD:   null,
};

// Number/symbol colour printed on the cream oval. Yellow needs a darker
// ink than its body, or it's illegible on cream.
const INK = {
  RED:    'var(--uno-red)',
  BLUE:   'var(--uno-blue)',
  GREEN:  'var(--uno-green)',
  YELLOW: 'var(--uno-yellow-ink)',
  WILD:   '#3a3942',
};

const LABEL = {
  Skip:    '⊘',
  Reverse: '⇄',
  '+2':    '+2',
  '+4':    '+4',
  Wild:    '★',
};

// w/h define the card stock; everything else scales off w.
const SIZES = {
  sm: { w: 46,  h: 68  },
  md: { w: 62,  h: 92  },
  lg: { w: 104, h: 152 },
  xl: { w: 120, h: 176 },
};

export function cardDims(size = 'md') { return SIZES[size] ?? SIZES.md; }

export default function UnoCard({ card, selected, playable, onClick, size = 'md' }) {
  const isWild = card.color === 'WILD';
  const label  = LABEL[card.value] ?? card.value;
  const { w, h } = cardDims(size);
  const fs     = Math.round(w * 0.46);
  const corner = Math.max(8, Math.round(w * 0.16));

  const bg  = isWild ? 'var(--wild-grad)' : BG[card.color];
  const ink = INK[card.color];

  return (
    <motion.div
      role="button"
      tabIndex={playable ? 0 : -1}
      style={{
        width: w,
        height: h,
        borderRadius: Math.round(w * 0.13),
        background: bg,
        border: `${Math.max(2, Math.round(w * 0.045))}px solid var(--card-cream)`,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        cursor: playable ? 'pointer' : 'default',
        opacity: playable ? 1 : 0.4,
        userSelect: 'none',
        flexShrink: 0,
        position: 'relative',
        overflow: 'hidden',
        boxShadow: selected
          ? '0 0 0 3px var(--accent-glow), 0 12px 28px rgba(0,0,0,0.5)'
          : '0 3px 12px rgba(0,0,0,0.4)',
      }}
      animate={{ y: selected ? -20 : 0, scale: selected ? 1.05 : 1 }}
      whileHover={playable ? { y: selected ? -20 : -12, scale: 1.04 } : {}}
      transition={{ type: 'spring', stiffness: 400, damping: 28 }}
      onClick={playable ? onClick : undefined}
    >
      {/* gloss */}
      <div style={{
        position: 'absolute', inset: 0, borderRadius: 'inherit', pointerEvents: 'none',
        background: 'linear-gradient(180deg, rgba(255,255,255,0.24), transparent 46%)',
      }} />

      {/* tilted cream oval */}
      <div style={{
        width: '82%', height: '66%', borderRadius: '50%',
        background: 'var(--card-cream)', transform: 'rotate(-22deg)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        boxShadow: 'inset 0 1px 3px rgba(0,0,0,0.12)',
      }}>
        <span style={{
          transform: 'rotate(22deg)',
          fontFamily: 'Manrope, sans-serif', fontWeight: 800,
          fontSize: fs, color: ink, lineHeight: 1,
        }}>
          {label}
        </span>
      </div>

      {/* corner labels */}
      <span style={{
        position: 'absolute', top: Math.round(corner * 0.35), left: Math.round(corner * 0.5),
        fontSize: corner, fontWeight: 800, fontFamily: 'Manrope, sans-serif', color: '#fff',
        textShadow: '0 1px 2px rgba(0,0,0,0.4)',
      }}>{label}</span>
      <span style={{
        position: 'absolute', bottom: Math.round(corner * 0.35), right: Math.round(corner * 0.5),
        fontSize: corner, fontWeight: 800, fontFamily: 'Manrope, sans-serif', color: '#fff',
        textShadow: '0 1px 2px rgba(0,0,0,0.4)', transform: 'rotate(180deg)',
      }}>{label}</span>
    </motion.div>
  );
}
