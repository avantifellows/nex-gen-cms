export const HOME_PAGE_URL = 'http://localhost:8080';

export const dropdowns = [
    { name: 'curriculum-dropdown', urlPattern: /\/api\/curriculums/, content: ['c1', 'c2'], key: 'selectedCurriculum', selectedVal: 'c2' },
    { name: 'grade-dropdown', urlPattern: /\/api\/grades/, content: ['g1', 'g2'], key: 'selectedGrade', selectedVal: 'g2' },
    { name: 'subject-dropdown', urlPattern: /\/api\/subjects/, content: ['s1', 's2'], key: 'selectedSubject', selectedVal: 's2' },
];
