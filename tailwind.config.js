/** @type {import('tailwindcss').Config} */
module.exports = {
    content: [
        "./web/**/*.html",     // Scans HTML files
        "./web/**/*.js",          // Scans any static JS files (if you use Tailwind classes in JS)
    ],
    theme: {
        extend: {},
    },
    plugins: [],
}