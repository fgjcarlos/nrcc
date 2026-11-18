import { useEffect, useState } from 'react';

// Tailwind's `lg` breakpoint (min-width: 1024px; no custom screens in
// tailwind.config.js). Below it the rail becomes a drawer (#966).
const LG_BREAKPOINT = 1024;

/** True while the viewport is narrower than Tailwind's `lg` breakpoint. */
export function useBelowLg(): boolean {
  const [below, setBelow] = useState(() => window.innerWidth < LG_BREAKPOINT);
  useEffect(() => {
    const update = () => setBelow(window.innerWidth < LG_BREAKPOINT);
    window.addEventListener('resize', update);
    return () => window.removeEventListener('resize', update);
  }, []);
  return below;
}
