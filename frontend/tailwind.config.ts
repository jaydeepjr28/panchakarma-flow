import type { Config } from "tailwindcss";
import tailwindcssAnimate from "tailwindcss-animate";
import tailwindcssAspectRatio from "@tailwindcss/aspect-ratio";

export default {
  darkMode: ["class"],
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1400px",
      },
    },
    extend: {
      colors: {
        neu: {
          bg: "#F5F3EC",
          surface: "#F5F3EC",
          border: "#E2DFD2",
        },
        forest: {
          DEFAULT: "#315C49",
          dark: "#234737",
          light: "#417A61",
          muted: "rgba(49, 92, 73, 0.12)",
        },
        sage: {
          DEFAULT: "#A8B9A0",
          light: "#C7D4C1",
          dark: "#84987C",
          muted: "rgba(168, 185, 160, 0.2)",
        },
        terracotta: {
          DEFAULT: "#C98769",
          light: "#DBA389",
          dark: "#A9674A",
          muted: "rgba(201, 135, 105, 0.15)",
        },
        charcoal: {
          DEFAULT: "#27352D",
          soft: "#3A4B41",
        },
        slateneu: {
          DEFAULT: "#6F7B71",
          light: "#8E9A90",
        },
        neuSuccess: "#357A55",
        neuWarning: "#B7792B",
        neuError: "#B74747",

        // Radix compatibility tokens
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "#315C49",
          foreground: "#F5F3EC",
        },
        secondary: {
          DEFAULT: "#A8B9A0",
          foreground: "#27352D",
        },
        destructive: {
          DEFAULT: "#B74747",
          foreground: "#FFFFFF",
        },
        muted: {
          DEFAULT: "#E7E4DC",
          foreground: "#6F7B71",
        },
        accent: {
          DEFAULT: "#C98769",
          foreground: "#FFFFFF",
        },
        card: {
          DEFAULT: "#F5F3EC",
          foreground: "#27352D",
        },
      },
      borderRadius: {
        neu: "18px",
        "neu-sm": "12px",
        "neu-lg": "24px",
        "neu-full": "9999px",
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      boxShadow: {
        'neu-raised-xs': '2px 2px 4px rgba(190, 184, 168, 0.5), -2px -2px 4px rgba(255, 255, 255, 0.95)',
        'neu-raised-sm': '3px 3px 7px rgba(190, 184, 168, 0.55), -3px -3px 7px rgba(255, 255, 255, 0.95)',
        'neu-raised': '6px 6px 14px rgba(190, 184, 168, 0.55), -6px -6px 14px rgba(255, 255, 255, 0.95)',
        'neu-raised-lg': '10px 10px 24px rgba(190, 184, 168, 0.6), -10px -10px 24px rgba(255, 255, 255, 0.95)',
        'neu-inset-sm': 'inset 2px 2px 4px rgba(190, 184, 168, 0.5), inset -2px -2px 4px rgba(255, 255, 255, 0.9)',
        'neu-inset': 'inset 3px 3px 7px rgba(190, 184, 168, 0.55), inset -3px -3px 7px rgba(255, 255, 255, 0.95)',
        'neu-inset-deep': 'inset 4px 4px 10px rgba(190, 184, 168, 0.6), inset -4px -4px 10px rgba(255, 255, 255, 0.95)',
        'neu-pressed': 'inset 2px 2px 5px rgba(190, 184, 168, 0.65), inset -2px -2px 5px rgba(255, 255, 255, 0.9)',
        'neu-glow-forest': '0 0 16px rgba(49, 92, 73, 0.3), 4px 4px 10px rgba(190, 184, 168, 0.45)',
        'neu-glow-terracotta': '0 0 16px rgba(201, 135, 105, 0.35), 4px 4px 10px rgba(190, 184, 168, 0.45)',
      },
      keyframes: {
        "accordion-down": {
          from: { height: "0" },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: "0" },
        },
        pulseGlow: {
          "0%, 100%": { opacity: "1" },
          "50%": { opacity: "0.6" },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
        "pulse-glow": "pulseGlow 2.5s infinite ease-in-out",
      },
    },
  },
  plugins: [tailwindcssAnimate, tailwindcssAspectRatio],
} satisfies Config;
