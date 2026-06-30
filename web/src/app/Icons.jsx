// Shared Lucide-style outline icons for the chartpress UI (~1.7px stroke).
// Substituting Lucide-style glyphs for Radix Icons (see design readme ICONOGRAPHY).
import React from "react";

const Icon = ({ d, size = 18, sw = 1.7, children, ...p }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
       strokeWidth={sw} strokeLinecap="round" strokeLinejoin="round" {...p}>
    {children || <path d={d} />}
  </svg>
);

export const Plus = (p) => <Icon {...p}><path d="M12 5v14M5 12h14" /></Icon>;
export const X = (p) => <Icon {...p}><path d="M18 6 6 18M6 6l12 12" /></Icon>;
export const Form = (p) => <Icon {...p}><rect x="3" y="3" width="18" height="18" rx="2" /><path d="M8 8h8M8 12h8M8 16h5" /></Icon>;
export const Sparkle = (p) => <Icon {...p}><path d="M12 3v4M12 17v4M3 12h4M17 12h4M6.3 6.3l2.5 2.5M15.2 15.2l2.5 2.5M17.7 6.3l-2.5 2.5M8.8 15.2l-2.5 2.5" /></Icon>;
export const Download = (p) => <Icon {...p}><path d="M12 3v12m0 0 4-4m-4 4-4-4M5 21h14" /></Icon>;
export const ArrowLeft = (p) => <Icon {...p}><path d="M19 12H5m0 0 6 6m-6-6 6-6" /></Icon>;
export const ArrowRight = (p) => <Icon {...p}><path d="M5 12h14m0 0-6-6m6 6-6 6" /></Icon>;
export const Package = (p) => <Icon {...p}><path d="M21 8 12 3 3 8m18 0-9 5m9-5v8l-9 5m0-8L3 8m9 5v8M3 8v8l9 5" /></Icon>;
export const Layers = (p) => <Icon {...p}><path d="m12 2 9 5-9 5-9-5 9-5Zm9 10-9 5-9-5m18 5-9 5-9-5" /></Icon>;
export const Search = (p) => <Icon {...p}><circle cx="11" cy="11" r="7" /><path d="m21 21-4.3-4.3" /></Icon>;
export const Github = (p) => <Icon {...p}><path d="M15 22v-4a4 4 0 0 0-1-3c3 0 6-2 6-6 0-1.5-.5-3-1.5-4 .5-1.5 0-3-.5-3.5-1.5 0-3 1-3.5 1.5a12 12 0 0 0-6 0C8 3 6.5 2 5 2c-.5.5-1 2-.5 3.5C3.5 6.5 3 8 3 9.5c0 4 3 6 6 6a4 4 0 0 0-1 3v4" /></Icon>;
export const Check = (p) => <Icon {...p}><path d="M20 6 9 17l-5-5" /></Icon>;
