import { motion, AnimatePresence } from 'framer-motion';
import UnoCard, { cardDims } from './UnoCard';

/**
 * DiscardPile — the hero. The top card is the single most important piece of
 * state (the card you must match), so it's large, tilted, and sits on a small
 * peeking stack. Always rendered in view (board edge on desktop, pile dock on
 * mobile). The flip animation on change is preserved.
 */
export default function DiscardPile({ discardTop, size = 'lg' }) {
  const { w, h } = cardDims(size);
  const r = Math.round(w * 0.13);

  const peek = {
    position: 'absolute',
    top: '50%', left: '50%',
    width: w, height: h, borderRadius: r,
    marginTop: -h / 2, marginLeft: -w / 2,
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10 }}>
      <span style={{
        fontFamily: 'Manrope, sans-serif', fontWeight: 800, fontSize: 10.5,
        letterSpacing: 1.6, textTransform: 'uppercase', color: 'var(--text-muted)',
        whiteSpace: 'nowrap',
      }}>
        Discard · match this
      </span>

      <div style={{ position: 'relative', width: w + 24, height: h + 24,
                    display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        {/* peeking stack behind the top card */}
        <div style={{ ...peek, transform: 'rotate(-9deg)', background: 'rgba(0,0,0,0.22)' }} />
        <div style={{ ...peek, transform: 'rotate(6deg)',
                      background: 'rgba(255,255,255,0.12)', border: '1px solid var(--border)' }} />

        <AnimatePresence mode="wait">
          {discardTop?.value ? (
            <motion.div
              key={discardTop.value + discardTop.color}
              style={{ position: 'relative', transform: 'rotate(-3deg)',
                       filter: 'drop-shadow(0 16px 38px rgba(0,0,0,0.5))' }}
              initial={{ rotateY: 90, opacity: 0 }}
              animate={{ rotateY: 0, opacity: 1 }}
              exit={{ rotateY: -90, opacity: 0 }}
              transition={{ duration: 0.25 }}
            >
              <UnoCard card={discardTop} playable={false} size={size} />
            </motion.div>
          ) : (
            <div style={{
              position: 'relative', width: w, height: h, borderRadius: r,
              border: '2px dashed var(--border)', color: 'var(--text-muted)',
              display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 22,
            }}>—</div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
