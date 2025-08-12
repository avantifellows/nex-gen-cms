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
        'pl-6'
    ],
    theme: {
        extend: {},
    },
    plugins: [],
}