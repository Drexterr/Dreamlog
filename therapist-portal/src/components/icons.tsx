// Thin-stroke line icons drawn for DreamLog. Inherit color from parent via currentColor.

interface IconProps {
  size?: number;
  strokeWidth?: number;
}

function Svg({ size = 22, strokeWidth = 1.5, children }: IconProps & { children: React.ReactNode }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {children}
    </svg>
  );
}

export function Leaf(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M5 15.5C5 9 10.5 4.5 18.5 4.5c0 8-4.5 13-11 13" />
      <path d="M4 20c2.5-5.5 6.5-9.5 12-12.5" />
    </Svg>
  );
}

export function Mic(props: IconProps) {
  return (
    <Svg {...props}>
      <rect x="9" y="3" width="6" height="11" rx="3" />
      <path d="M5.5 11.5a6.5 6.5 0 0 0 13 0" />
      <path d="M12 18v3" />
    </Svg>
  );
}

export function Bubble(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M4.5 7A3.5 3.5 0 0 1 8 3.5h8A3.5 3.5 0 0 1 19.5 7v5a3.5 3.5 0 0 1-3.5 3.5H9.8L5.5 19.7v-4.5A3.5 3.5 0 0 1 4.5 12z" />
      <path d="M10.3 8.2a1.8 1.8 0 1 1 2.5 1.9c-.6.3-.8.6-.8 1.2" />
      <path d="M12 13.4v.05" />
    </Svg>
  );
}

export function Chart(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M4 19.5h16" />
      <path d="M5 15.5l4.2-5.3 3.8 3 5.5-7.2" />
    </Svg>
  );
}

export function Lock(props: IconProps) {
  return (
    <Svg {...props}>
      <rect x="5" y="10.5" width="14" height="9.5" rx="2.5" />
      <path d="M8 10.5V8a4 4 0 0 1 8 0v2.5" />
      <path d="M12 14.5v2" />
    </Svg>
  );
}

export function Sprout(props: IconProps) {
  return (
    <Svg {...props}>
      <path d="M12 21v-7.5" />
      <path d="M12 13.5c0-4 3-6.5 7.5-6.5 0 4-3 6.5-7.5 6.5z" />
      <path d="M12 11.5C12 8.3 9.7 6.5 6 6.5c0 3.2 2.3 5 6 5z" />
    </Svg>
  );
}
