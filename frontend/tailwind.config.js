/** @type {import('tailwindcss').Config} */
import daisyui from 'daisyui'
import { spacing, shadows, borderRadius } from './src/tokens.ts'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  // Enable dark mode based on data-theme attribute selector
  darkMode: ['selector', '[data-theme="dark"]'],
  theme: {
    extend: {
      // Spacing scale from design tokens
      spacing: {
        'xs': spacing.xs,
        'sm': spacing.sm,
        'base': spacing.base,
        'md': spacing.md,
        'lg': spacing.lg,
        'xl': spacing.xl,
      },
      
      // Shadow elevation system
      boxShadow: {
        'sm': shadows.sm,
        'md': shadows.md,
        'lg': shadows.lg,
        'xl': shadows.xl,
        '2xl': shadows['2xl'],
        'glow': shadows.glow,
        'glow-light': shadows['glow-light'],
      },
      
      // Border radius tokens
      borderRadius: {
        'xs': borderRadius.xs,
        'sm': borderRadius.sm,
        'md': borderRadius.md,
        'lg': borderRadius.lg,
        'xl': borderRadius.xl,
        'full': borderRadius.full,
      },
      
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', '-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'sans-serif'],
        mono: ['"Berkeley Mono"', '"IBM Plex Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', '"Liberation Mono"', '"Courier New"', 'monospace'],
      },
    },
  },
  plugins: [daisyui],
  daisyui: {
    themes: [
      {
        dark: {
          primary: '#ef4444',
          'primary-content': '#ffffff',
          secondary: '#18181b',
          'secondary-content': '#fafafa',
          accent: '#a1a1aa',
          'accent-content': '#fafafa',
          'base-100': '#09090b',
          'base-200': '#0a0a0c',
          'base-300': '#111113',
          'base-content': '#fafafa',
          success: '#22c55e',
          'success-content': '#ffffff',
          warning: '#f59e0b',
          'warning-content': '#ffffff',
          error: '#ef4444',
          'error-content': '#ffffff',
          info: '#3b82f6',
          'info-content': '#ffffff',
        },
        light: {
          primary: '#dc2626',
          'primary-content': '#ffffff',
          secondary: '#f4f4f5',
          'secondary-content': '#09090b',
          accent: '#52525b',
          'accent-content': '#fafafa',
          'base-100': '#fafafa',
          'base-200': '#f4f4f5',
          'base-300': '#e4e4e7',
          'base-content': '#09090b',
          success: '#16a34a',
          'success-content': '#ffffff',
          warning: '#d97706',
          'warning-content': '#ffffff',
          error: '#dc2626',
          'error-content': '#ffffff',
          info: '#2563eb',
          'info-content': '#ffffff',
        },
      },
    ],
    logs: false,
  },
}
