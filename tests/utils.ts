// Base URL
export const BASE_URL = 'http://localhost:8080';

// Specific pages
export const LOGIN_PAGE_URL = `${BASE_URL}/login`;
export const HOME_PAGE_URL = `${BASE_URL}/home`;

export const dropdowns = [
    { name: 'curriculum-dropdown', urlPattern: /\/api\/curriculums/, content: ['1', '2'], key: 'selectedCurriculum', selectedVal: '2' },
    { name: 'grade-dropdown', urlPattern: /\/api\/grades/, content: ['3', '4'], key: 'selectedGrade', selectedVal: '4' },
    { name: 'subject-dropdown', urlPattern: /\/api\/subjects/, content: ['5', '6'], key: 'selectedSubject', selectedVal: '6' },
];
