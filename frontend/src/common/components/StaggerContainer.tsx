import { motion } from 'framer-motion'

interface StaggerContainerProps {
  children: React.ReactNode
  staggerDelay?: number
  delay?: number
}

/**
 * StaggerContainer animates children in sequence with a stagger effect.
 * Each child appears after the previous one with a configurable delay.
 */
export function StaggerContainer({
  children,
  staggerDelay = 0.05,
  delay = 0,
}: StaggerContainerProps) {
  const childrenArray = Array.isArray(children)
    ? children
    : children
      ? [children]
      : []

  const containerVariants = {
    hidden: { opacity: 0 },
    visible: {
      opacity: 1,
      transition: {
        staggerChildren: staggerDelay,
        delayChildren: delay,
      },
    },
  }

  const itemVariants = {
    hidden: { opacity: 0, y: 12 },
    visible: {
      opacity: 1,
      y: 0,
      transition: {
        duration: 0.3,
      },
    },
  }

  return (
    <motion.div
      variants={containerVariants}
      initial="hidden"
      animate="visible"
    >
      {childrenArray.map((child, index) => (
        <motion.div key={index} variants={itemVariants}>
          {child}
        </motion.div>
      ))}
    </motion.div>
  )
}
