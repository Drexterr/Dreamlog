'use client';

import { useEffect, useState } from 'react';
import { motion, useScroll, useSpring, useTransform, useReducedMotion } from 'motion/react';

/**
 * The Breath Line — Ode's brand mark. One continuous stroke that trembles
 * like speech, decays, and settles into a single point of rest.
 * Master paths from the identity spec (docs/brand/ode-breath-line.html).
 */
export const BREATH_LINE_D =
  'M 8 32 C 18 10, 28 54, 38 32 C 47 15, 56 49, 65 32 C 73 20, 81 44, 89 32 ' +
  'C 96 25, 103 39, 110 32 C 117 28.5, 124 35.5, 131 32 C 141 30, 153 34, 164 32';

// Icon crop — two expressive waves + settle + dot (drawn on a 96-wide canvas,
// vertical extent 24–72). Used bare in lockups; boxed only where a chip is
// unavoidable (favicons, app icon).
const ICON_CROP_D =
  'M 8 48 C 17 24, 27 72, 37 48 C 45 30, 54 64, 62 48 C 67 42, 72 52, 76 48';

// Vertical variant of the full mark: x/y swapped, so the tremor decays
// top-to-bottom and settles into the dot at the base. viewBox 0 0 64 192.
const BREATH_LINE_VERTICAL_D =
  'M 32 8 C 10 18, 54 28, 32 38 C 15 47, 49 56, 32 65 C 20 73, 44 81, 32 89 ' +
  'C 25 96, 39 103, 32 110 C 28.5 117, 35.5 124, 32 131 C 30 141, 34 153, 32 164';

const AMBER = '#c8955a';
const AMBER_HI = '#d4a870';

/**
 * The bare brand mark for lockups — no chip, no border. The line sits
 * directly on the surface, the way the wordmark's letters do.
 */
export function OdeMark({ size = 28 }: { size?: number }) {
  // Wide crop: just the mark's own bounding box (96 × 48), so the squiggle
  // reads at wordmark scale instead of floating inside a square.
  return (
    <svg
      width={size * 1.35}
      height={size * 0.675}
      viewBox="0 24 96 48"
      aria-hidden="true"
      style={{ display: 'block', flexShrink: 0 }}
    >
      <path d={ICON_CROP_D} fill="none" stroke={AMBER} strokeWidth="7.5" strokeLinecap="round" />
      <circle cx="90" cy="48" r="6.5" fill={AMBER_HI} />
    </svg>
  );
}

/**
 * Fixed scroll rail: the Breath Line standing upright at the page's right
 * edge, drawing itself in step with scroll. Progress is anchored to the hero
 * headline — the line stays undrawn while "Your thoughts, out loud." is on
 * screen, starts trembling forward the moment the headline scrolls away, and
 * the dot lands at the very end of the page. Speak → steady → settle,
 * mapped onto reading the site.
 */
export function BreathRail({ startSelector = '#main-content h1' }: { startSelector?: string }) {
  const reduce = useReducedMotion();

  // Fraction of total scrollable height at which the hero headline starts
  // leaving the viewport — the rail's zero point.
  const [startFrac, setStartFrac] = useState(0);
  useEffect(() => {
    const measure = () => {
      const el = document.querySelector(startSelector);
      const scrollable = document.documentElement.scrollHeight - window.innerHeight;
      if (!el || scrollable <= 0) return;
      const headlineTop = el.getBoundingClientRect().top + window.scrollY;
      // Drawing spans from "headline begins to exit" to page end. Cap well
      // below 1 so a short page can never produce an empty range.
      setStartFrac(Math.min(0.5, Math.max(0, headlineTop / scrollable)));
    };
    measure();
    window.addEventListener('resize', measure);
    // Fonts/images settling shift layout after first paint — re-measure once.
    const t = setTimeout(measure, 1200);
    return () => { window.removeEventListener('resize', measure); clearTimeout(t); };
  }, [startSelector]);

  // Global page progress, re-based to the headline anchor.
  const { scrollYProgress } = useScroll();
  const anchored = useTransform(scrollYProgress, [startFrac, 1], [0, 1]);
  const eased = useSpring(anchored, { stiffness: 90, damping: 24, mass: 0.5 });
  const dotR = useTransform(eased, [0.92, 1], [0, 5]);
  const dotOpacity = useTransform(eased, [0.92, 1], [0, 1]);

  // Static under reduced motion — a quiet full mark, no scroll coupling.
  return (
    <div aria-hidden="true" className="breath-rail">
      <svg viewBox="0 0 64 192" style={{ width: '100%', height: '100%', overflow: 'visible' }}>
        {/* Ghost track: the full path at whisper opacity, so the rail reads
            as an indicator with a destination, not a floating fragment */}
        <path
          d={BREATH_LINE_VERTICAL_D}
          fill="none"
          stroke="rgba(200,149,90,0.13)"
          strokeWidth="2"
          strokeLinecap="round"
        />
        {reduce ? (
          <>
            <path d={BREATH_LINE_VERTICAL_D} fill="none" stroke="rgba(200,149,90,0.6)" strokeWidth="2" strokeLinecap="round" />
            <circle cx="32" cy="179" r="5" fill={AMBER_HI} opacity="0.8" />
          </>
        ) : (
          <>
            <motion.path
              d={BREATH_LINE_VERTICAL_D}
              fill="none"
              stroke="rgba(200,149,90,0.65)"
              strokeWidth="2.4"
              strokeLinecap="round"
              style={{ pathLength: eased }}
            />
            <motion.circle cx="32" cy="179" r={dotR} fill={AMBER_HI} style={{ opacity: dotOpacity }} />
          </>
        )}
      </svg>
      <style>{`
        .breath-rail {
          position: fixed;
          right: 18px;
          top: 50%;
          transform: translateY(-50%);
          height: min(62vh, 560px);
          width: 34px;
          z-index: 40;
          pointer-events: none;
        }
        /* The rail needs quiet margin; below 1100px it would sit on content */
        @media (max-width: 1100px) {
          .breath-rail { display: none; }
        }
      `}</style>
    </div>
  );
}
