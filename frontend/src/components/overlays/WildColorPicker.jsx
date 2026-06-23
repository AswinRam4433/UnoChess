import { motion } from 'framer-motion';
import { UNO_COLORS, COLOR_HEX } from '../../lib/constants';

export default function WildColorPicker({ onPick }) {
  return (
    <motion.div
      style={styles.backdrop}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      <motion.div
        style={styles.box}
        initial={{ scale: 0.85, opacity: 0 }}
        animate={{ scale: 1, opacity: 1 }}
        exit={{ scale: 0.85, opacity: 0 }}
        transition={{ type: 'spring', damping: 20, stiffness: 300 }}
      >
        <h3 style={styles.title}>Choose a color</h3>
        <div style={styles.grid}>
          {UNO_COLORS.map(color => (
            <motion.button
              key={color}
              style={{ ...styles.swatch, background: COLOR_HEX[color] }}
              onClick={() => onPick(color)}
              whileHover={{ scale: 1.08 }}
              whileTap={{ scale: 0.94 }}
            >
              {color.charAt(0) + color.slice(1).toLowerCase()}
            </motion.button>
          ))}
        </div>
      </motion.div>
    </motion.div>
  );
}

const styles = {
  backdrop: {
    position: 'absolute',
    inset: 0,
    background: 'rgba(0,0,0,0.7)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 50,
    borderRadius: 'inherit',
  },
  box: {
    background: 'var(--surface-r)',
    border: '1px solid var(--border)',
    borderRadius: 14,
    padding: '24px 28px',
    display: 'flex',
    flexDirection: 'column',
    gap: 18,
    alignItems: 'center',
  },
  title: {
    fontFamily: 'Cinzel, serif',
    fontSize: 15,
    fontWeight: 600,
    color: 'var(--text)',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: 12,
  },
  swatch: {
    width: 110,
    height: 72,
    borderRadius: 10,
    border: '2px solid rgba(255,255,255,0.15)',
    color: '#fff',
    fontFamily: 'Cinzel, serif',
    fontWeight: 600,
    fontSize: 15,
    cursor: 'pointer',
    letterSpacing: 0.5,
  },
};
