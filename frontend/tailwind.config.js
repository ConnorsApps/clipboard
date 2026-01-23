/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        dark: {
          900: '#1a1a2e',
          800: '#16213e',
          700: '#0f3460',
        },
        accent: {
          500: '#e94560',
          600: '#d63d56',
        }
      }
    },
  },
  plugins: [],
}
