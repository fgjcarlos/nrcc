/**
 * Design System Tokens
 * Centralized design token definitions for Node-RED Control Center
 * Includes spacing, colors, typography, borders, and shadows
 */

// ============================================================================
// SPACING SCALE (4px base grid)
// ============================================================================
export const spacing = {
  xs: '2px',    // Minimal spacing
  sm: '4px',    // 1 unit
  base: '8px',  // 2 units
  md: '16px',   // 4 units (standard section spacing)
  lg: '24px',   // 6 units (page spacing)
  xl: '32px',   // 8 units (large section spacing)
} as const;

// ============================================================================
// COLOR TOKENS
// ============================================================================

// Dark Mode Surface Colors (with clear differentiation)
export const darkColors = {
  background: '#09090b',        // Base background
  'background-alt': '#0d0d0f',  // Alternative background
  card: '#111113',              // Card background (slightly lighter)
  'card-hover': '#1a1a1d',      // Elevated card on hover
  surface: '#16161a',           // General surface
  'surface-hover': '#1f1f24',   // Surface on hover
  popover: '#202023',           // Popover/dropdown background
  
  // Text colors
  'text-primary': '#fafafa',    // Primary text (white)
  'text-secondary': '#a1a1aa',  // Secondary text (muted)
  'text-tertiary': '#71717a',   // Tertiary text (very muted)
  
  // Borders
  'border-default': 'rgba(113, 113, 122, 0.16)',
  'border-subtle': 'rgba(113, 113, 122, 0.08)',
  'border-hover': 'rgba(113, 113, 122, 0.24)',
} as const;

// Light Mode Surface Colors
export const lightColors = {
  background: '#fafafa',        // Base background
  'background-alt': '#f4f4f5',  // Alternative background
  card: '#ffffff',              // Card background (white)
  'card-hover': '#f9f9fb',      // Card on hover (very subtle)
  surface: '#f4f4f5',           // General surface
  'surface-hover': '#efefef',   // Surface on hover
  popover: '#ffffff',           // Popover/dropdown background
  
  // Text colors
  'text-primary': '#09090b',    // Primary text (black)
  'text-secondary': '#52525b',  // Secondary text (muted)
  'text-tertiary': '#a1a1aa',   // Tertiary text (very muted)
  
  // Borders
  'border-default': 'rgba(9, 9, 11, 0.1)',
  'border-subtle': 'rgba(9, 9, 11, 0.06)',
  'border-hover': 'rgba(9, 9, 11, 0.16)',
} as const;

// Semantic Status Colors (used in both themes)
export const statusColors = {
  success: '#22c55e',    // Success (green)
  warning: '#f59e0b',    // Warning (amber)
  error: '#ef4444',      // Error (red)
  info: '#3b82f6',       // Info (blue)
  'success-light': '#86efac',
  'warning-light': '#fcd34d',
  'error-light': '#fca5a5',
  'info-light': '#93c5fd',
} as const;

// ============================================================================
// TYPOGRAPHY SCALE
// ============================================================================
export const typography = {
  // Display/Page titles
  'display-lg': {
    size: '2.5rem',    // 40px
    weight: 700,
    lineHeight: 1.2,
    tracking: '-0.02em',
  },
  'display-md': {
    size: '2rem',      // 32px
    weight: 700,
    lineHeight: 1.25,
    tracking: '-0.02em',
  },
  'display-sm': {
    size: '1.875rem',  // 30px
    weight: 700,
    lineHeight: 1.2,
    tracking: '-0.02em',
  },
  
  // Page/Section titles
  'heading-lg': {
    size: '1.875rem',  // 30px
    weight: 700,
    lineHeight: 1.3,
    tracking: '-0.01em',
  },
  'heading-md': {
    size: '1.5rem',    // 24px
    weight: 700,
    lineHeight: 1.3,
    tracking: '-0.01em',
  },
  'heading-sm': {
    size: '1.25rem',   // 20px
    weight: 700,
    lineHeight: 1.4,
  },
  
  // Section titles
  'title-lg': {
    size: '1.125rem',  // 18px
    weight: 600,
    lineHeight: 1.4,
  },
  'title-md': {
    size: '1rem',      // 16px
    weight: 600,
    lineHeight: 1.5,
  },
  'title-sm': {
    size: '0.875rem',  // 14px
    weight: 600,
    lineHeight: 1.5,
  },
  
  // Body text
  'body-lg': {
    size: '1rem',      // 16px
    weight: 400,
    lineHeight: 1.6,
  },
  'body-md': {
    size: '0.95rem',   // 15px - custom, often used in this app
    weight: 400,
    lineHeight: 1.6,
  },
  'body-sm': {
    size: '0.875rem',  // 14px
    weight: 400,
    lineHeight: 1.5,
  },
  
  // Caption/metadata
  'caption-lg': {
    size: '0.8125rem', // 13px
    weight: 500,
    lineHeight: 1.4,
  },
  'caption-md': {
    size: '0.75rem',   // 12px
    weight: 500,
    lineHeight: 1.4,
  },
  'caption-sm': {
    size: '0.6875rem', // 11px
    weight: 500,
    lineHeight: 1.4,
  },
  
  // Mono/Code
  'mono-md': {
    size: '0.875rem',  // 14px
    weight: 400,
    lineHeight: 1.6,
    family: 'Berkeley Mono, IBM Plex Mono, monospace',
  },
  'mono-sm': {
    size: '0.8125rem', // 13px
    weight: 400,
    lineHeight: 1.5,
    family: 'Berkeley Mono, IBM Plex Mono, monospace',
  },
} as const;

// ============================================================================
// BORDER RADIUS TOKENS
// ============================================================================
export const borderRadius = {
  none: '0',
  'xs': '0.25rem',    // 4px
  'sm': '0.5rem',     // 8px
  'md': '0.75rem',    // 12px
  'lg': '1rem',       // 16px (default buttons, inputs)
  'xl': '1.25rem',    // 20px (cards, containers)
  'full': '9999px',   // Full border radius (badges, pills)
} as const;

// ============================================================================
// SHADOW TOKENS (Elevation system)
// ============================================================================
export const shadows = {
  // No shadow (base level)
  'none': 'none',
  
  // Subtle shadow (sm - cards, subtle elevation)
  'sm': '0 1px 2px rgba(0, 0, 0, 0.05)',
  
  // Standard shadow (md - standard cards, dropdowns)
  'md': '0 4px 6px rgba(0, 0, 0, 0.1), 0 2px 4px rgba(0, 0, 0, 0.06)',
  
  // Elevated shadow (lg - modal cards, popovers)
  'lg': '0 10px 15px rgba(0, 0, 0, 0.1), 0 4px 6px rgba(0, 0, 0, 0.05)',
  
  // Large shadow (xl - large modals, critical overlays)
  'xl': '0 20px 25px rgba(0, 0, 0, 0.1), 0 10px 10px rgba(0, 0, 0, 0.04)',
  
  // Extra large shadow (2xl - drawer, off-canvas)
  '2xl': '0 25px 50px rgba(0, 0, 0, 0.15)',
  
  // Glow effect (used in current styles)
  'glow': '0 20px 40px rgba(0, 0, 0, 0.5)',
  'glow-light': '0 20px 40px rgba(0, 0, 0, 0.06)',
} as const;

// ============================================================================
// COMPONENT-SPECIFIC TOKENS
// ============================================================================

// Card padding variants
export const cardPadding = {
  compact: '12px',    // p-3
  standard: '24px',   // p-6
  large: '32px',      // p-8
} as const;

// Button sizes
export const buttonSizes = {
  sm: {
    height: '2.25rem',
    padding: '0.5rem 0.75rem',
    fontSize: '0.875rem',
  },
  md: {
    height: '2.75rem',
    padding: '0.65rem 1rem',
    fontSize: '0.95rem',
  },
  lg: {
    height: '3.25rem',
    padding: '0.75rem 1.25rem',
    fontSize: '1rem',
  },
} as const;

// Input sizes
export const inputSizes = {
  default: {
    height: '3rem',
    padding: '0.75rem',
    fontSize: '0.95rem',
  },
  compact: {
    height: '2.75rem',
    padding: '0.65rem',
    fontSize: '0.875rem',
  },
} as const;

// ============================================================================
// EXPORT DEFAULT TOKENS OBJECT
// ============================================================================
export const tokens = {
  spacing,
  colors: {
    dark: darkColors,
    light: lightColors,
    status: statusColors,
  },
  typography,
  borderRadius,
  shadows,
  cardPadding,
  buttonSizes,
  inputSizes,
} as const;

export default tokens;
