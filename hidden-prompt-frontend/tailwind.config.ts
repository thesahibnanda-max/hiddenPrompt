import type { Config } from "tailwindcss";

export default {
  darkMode: ["class"],
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      screens: {
        // Tailwind defaults kept: sm 640, md 768, lg 1024, xl 1280, 2xl 1536
        "3xl": "1920px", // large monitor / entry TV
        tv: "2560px", // 10-foot UI: bigger base font, bigger touch targets, wider safe margins
      },
      colors: {
        void: {
          DEFAULT: "#0a0118",
          2: "#160a2e",
        },
        neon: {
          magenta: "#ff2ec4",
          cyan: "#22e8f0",
          purple: "#9d4dff",
          pink: "#ff6ec7",
          amber: "#ffb347",
        },
        chrome: {
          light: "#f4f4f8",
          mid: "#b8b8c8",
          dark: "#4a4a5e",
        },
        border: "rgba(184,184,200,0.18)",
        input: "rgba(184,184,200,0.18)",
        ring: "#22e8f0",
        background: "#0a0118",
        foreground: "#f4f4f8",
        primary: {
          DEFAULT: "#ff2ec4",
          foreground: "#0a0118",
        },
        secondary: {
          DEFAULT: "#22e8f0",
          foreground: "#0a0118",
        },
        destructive: {
          DEFAULT: "#ff4d6d",
          foreground: "#f4f4f8",
        },
        muted: {
          DEFAULT: "#160a2e",
          foreground: "#b8b8c8",
        },
        accent: {
          DEFAULT: "#9d4dff",
          foreground: "#f4f4f8",
        },
        popover: {
          DEFAULT: "#160a2e",
          foreground: "#f4f4f8",
        },
        card: {
          DEFAULT: "rgba(22,10,46,0.6)",
          foreground: "#f4f4f8",
        },
      },
      backgroundImage: {
        "grid-horizon":
          "linear-gradient(transparent 0%, rgba(10,1,24,0.92) 88%), repeating-linear-gradient(0deg, rgba(34,232,240,0.15) 0px, rgba(34,232,240,0.15) 1px, transparent 1px, transparent 40px), repeating-linear-gradient(90deg, rgba(255,46,196,0.12) 0px, rgba(255,46,196,0.12) 1px, transparent 1px, transparent 40px)",
        "chrome-text":
          "linear-gradient(180deg, #ffffff 0%, #b8b8c8 40%, #ff2ec4 55%, #22e8f0 75%, #9d4dff 100%)",
        "sun-gradient":
          "linear-gradient(180deg, #ffb347 0%, #ff6ec7 45%, #9d4dff 100%)",
      },
      fontFamily: {
        display: ["var(--font-display)", "sans-serif"],
        body: ["var(--font-body)", "sans-serif"],
      },
      borderRadius: {
        lg: "0.75rem",
        md: "0.5rem",
        sm: "0.375rem",
      },
      keyframes: {
        scanline: {
          "0%": { transform: "translateY(-100%)" },
          "100%": { transform: "translateY(100%)" },
        },
        "grid-scroll": {
          "0%": { backgroundPosition: "0 0" },
          "100%": { backgroundPosition: "0 40px" },
        },
        "glow-pulse": {
          "0%, 100%": {
            opacity: "0.85",
            filter: "drop-shadow(0 0 6px currentColor)",
          },
          "50%": {
            opacity: "1",
            filter: "drop-shadow(0 0 16px currentColor)",
          },
        },
        shake: {
          "0%, 100%": { transform: "translateX(0)" },
          "20%": { transform: "translateX(-8px)" },
          "40%": { transform: "translateX(8px)" },
          "60%": { transform: "translateX(-8px)" },
          "80%": { transform: "translateX(8px)" },
        },
      },
      animation: {
        scanline: "scanline 8s linear infinite",
        "grid-scroll": "grid-scroll 3s linear infinite",
        "glow-pulse": "glow-pulse 2.4s ease-in-out infinite",
        shake: "shake 0.4s ease-in-out",
      },
      boxShadow: {
        neon: "0 0 8px rgba(34,232,240,0.8), 0 0 24px rgba(34,232,240,0.4)",
        "neon-magenta":
          "0 0 8px rgba(255,46,196,0.8), 0 0 24px rgba(255,46,196,0.4)",
        "neon-purple":
          "0 0 8px rgba(157,77,255,0.8), 0 0 24px rgba(157,77,255,0.4)",
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
} satisfies Config;
