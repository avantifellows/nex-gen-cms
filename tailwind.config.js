/** @type {import('tailwindcss').Config} */
module.exports = {
    content: [
        "./web/**/*.html",     // Scans HTML files
        "./web/**/*.js",          // Scans any static JS files (if you use Tailwind classes in JS)
    ],
    /**
     * Safelist ensures that certain classes are always included in the final CSS
     * even if they are not found in the content files.
     * This is useful for dynamic classes or classes that might not be present in the scanned files.
     * eg - question paper pdf instructions which are fetched from the server use these classes
     */
    safelist: [
        'list-disc',
        'list-decimal',
        'pl-6',
        'htmx-request'
    ],
    theme: {
        extend: {
            // Avanti Fellows brand palette — Warm Professional.
            // Mirrors the af_lms UI Style Guide tokens so the two CMS apps share a look.
            colors: {
                accent: {
                    DEFAULT: '#ad2f2f',   // AF maroon
                    hover:   '#8a2525',
                },
                'text-on-accent': '#ffffff',
                bg: {
                    DEFAULT:  '#f5efe8',  // warm beige page background
                    card:     '#fffaf5',  // cream card background
                    'card-alt': '#f3ece5', // table head / alt row
                    input:    '#ffffff',
                    hover:    'rgba(173, 47, 47, 0.06)',
                },
                ink: {
                    DEFAULT:   '#261410', // dark brown — primary text
                    secondary: '#685851', // taupe — muted text
                    muted:     '#685851',
                },
                border: {
                    DEFAULT: 'rgba(38, 20, 16, 0.15)',
                    accent:  '#ad2f2f',
                },
                danger: {
                    DEFAULT: '#ad2f2f',
                    bg:      'rgba(173, 47, 47, 0.08)',
                },
                success: {
                    DEFAULT: '#1e6b4b',
                    bg:      'rgba(30, 107, 75, 0.12)',
                },
                warning: {
                    DEFAULT: '#8c5a1d',
                    bg:      'rgba(140, 90, 29, 0.08)',
                    border:  '#8c5a1d',
                },
                info: {
                    DEFAULT: '#9AC4FA',
                    bg:      'rgba(154, 196, 250, 0.15)',
                },
                brand: {
                    coral:  '#E96D57',
                    gold:   '#FFD063',
                    amber:  '#FFB763',
                    blue:   '#9AC4FA',
                    salmon: '#FF9683',
                    orange: '#D77C11',
                    'coral-bg': 'rgba(233, 109, 87, 0.10)',
                    'gold-bg':  'rgba(255, 208, 99, 0.15)',
                    'amber-bg': 'rgba(255, 183, 99, 0.12)',
                    'blue-bg':  'rgba(154, 196, 250, 0.15)',
                },
            },
            fontFamily: {
                sans: ['Inter', 'system-ui', '-apple-system', 'Segoe UI', 'Roboto', 'Arial', 'sans-serif'],
                mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace'],
            },
            // Make a bare `border` class use our brand border tint so legacy templates
            // pick up the warm-professional palette without an explicit border-border-…
            borderColor: {
                DEFAULT: 'rgba(38, 20, 16, 0.15)',
            },
            ringColor: {
                DEFAULT: 'rgba(173, 47, 47, 0.40)',
            },
            boxShadow: {
                card: '0 1px 2px 0 rgba(38, 20, 16, 0.05), 0 1px 3px 0 rgba(38, 20, 16, 0.08)',
            },
        },
    },
    plugins: [],
}
