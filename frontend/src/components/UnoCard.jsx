import { motion } from 'framer-motion';

const BG = {
  RED:    '#c0392b',
  BLUE:   '#2471a3',
  GREEN:  '#1e8449',
  YELLOW: '#d4ac0d',
  WILD:   null,
};

const LABEL = {
  Skip:    '⊘',
  Reverse: '⇄',
  '+2':    '+2',
  '+4':    '+4',
  Wild:    '★',
};

export default function UnoCard({ card, selected, playable, onClick, size = 'md' }) {
  const isWild = card.color === 'WILD';
  const label  = LABEL[card.value] ?? card.value;
  const dim    = size === 'sm' ? { w: 46, h: 68, fs: 13 }
               : size === 'lg' ? { w: 80, h: 116, fs: 22 }
               :                 { w: 60, h: 88, fs: 17 };

  const bg = isWild
    ? 'linear-gradient(135deg, #c0392b 0%, #2471a3 33%, #1e8449 66%, #d4ac0d 100%)'
    : BG[card.color];

  return (
    <motion.div
      role="button"
      tabIndex={playable ? 0 : -1}
      style={{
        width: dim.w,
        height: dim.h,
        borderRadius: 8,
        background: bg,
        border: selected
          ? '2.5px solid #fff'
          : '2px solid rgba(255,255,255,0.12)',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        cursor: playable ? 'pointer' : 'default',
        opacity: playable ? 1 : 0.38,
        userSelect: 'none',
        flexShrink: 0,
        boxShadow: selected
          ? '0 0 0 3px var(--active-glow), 0 4px 16px rgba(0,0,0,0.5)'
          : '0 2px 8px rgba(0,0,0,0.4)',
        position: 'relative',
        overflow: 'hidden',
      }}
      animate={{
        y: selected ? -20 : 0,
        scale: selected ? 1.05 : 1,
      }}
      whileHover={playable ? { y: selected ? -20 : -12, scale: 1.03 } : {}}
      transition={{ type: 'spring', stiffness: 400, damping: 28 }}
      onClick={playable ? onClick : undefined}
    >
      {/* Inner oval */}
      <div style={{
        width: '72%',
        height: '82%',
        borderRadius: '50%',
        border: '2px solid rgba(255,255,255,0.25)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}>
        <span style={{
          fontSize: dim.fs,
          fontWeight: 700,
          color: '#fff',
          fontFamily: 'Cinzel, serif',
          textShadow: '0 1px 4px rgba(0,0,0,0.5)',
          lineHeight: 1,
        }}>
          {label}
        </span>
      </div>
      {/* Corner labels */}
      <span style={{ position:'absolute', top:4, left:6, fontSize:9, color:'rgba(255,255,255,0.8)', fontFamily:'Cinzel,serif', fontWeight:700 }}>
        {label}
      </span>
      <span style={{ position:'absolute', bottom:4, right:6, fontSize:9, color:'rgba(255,255,255,0.8)', fontFamily:'Cinzel,serif', fontWeight:700, transform:'rotate(180deg)' }}>
        {label}
      </span>
    </motion.div>
  );
}
