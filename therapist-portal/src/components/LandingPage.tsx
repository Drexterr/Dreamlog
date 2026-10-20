'use client';

import { useEffect, useRef } from 'react';
import markup from '../content/landing-markup.json';
import { mountLanding } from './landing-interactions';
export default function LandingPage() {
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (root.current) return mountLanding(root.current);
  }, []);
  return <div className="ode-landing" ref={root} data-goal-theme="curious" dangerouslySetInnerHTML={{
    __html: markup
  }} />;
}
