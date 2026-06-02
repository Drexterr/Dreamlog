import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50:  '#f3f0ff',
          100: '#e8e0ff',
          300: '#b39ddb',
          500: '#7c5cbf',
          700: '#4a2d82',
          900: '#1a1625',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        serif: ['Georgia', 'serif'],
      },
    },
  },
  plugins: [],
};

export default config;
