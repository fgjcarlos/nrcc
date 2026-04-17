import { motion } from 'framer-motion'
import { useLocation } from 'react-router-dom'

/**
 * PageTransition wraps page content with smooth fade-in and optional slide-up animations.
 * Automatically triggers on route changes.
 */
export function PageTransition({ children }: { children: React.ReactNode }) {
  const location = useLocation()

  return (
    <motion.div
      key={location.pathname}
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0 }}
      transition={{
        duration: 0.25,
      }}
    >
      {children}
    </motion.div>
  )
}
