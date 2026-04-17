import { motion } from 'framer-motion'

interface AnimatedCardProps {
  children: React.ReactNode
  className?: string
  delay?: number
  onClick?: () => void
}

/**
 * AnimatedCard wraps content with entrance animation and hover effects.
 * Designed to be used within a StaggerContainer for coordinated animations.
 */
export function AnimatedCard({
  children,
  className = '',
  delay = 0,
  onClick,
}: AnimatedCardProps) {
  return (
    <motion.div
      className={`interactive-card ${className}`}
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{
        duration: 0.3,
        delay,
        ease: 'easeOut',
      }}
      whileHover={{
        y: -2,
        scale: 1.005,
        transition: { duration: 0.2 },
      }}
      onClick={onClick}
    >
      {children}
    </motion.div>
  )
}
