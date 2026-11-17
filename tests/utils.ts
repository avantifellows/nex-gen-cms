import { Page } from '@playwright/test';

// Base URL
export const BASE_URL = 'http://localhost:8080';

// Specific pages
export const LOGIN_PAGE_URL = `${BASE_URL}/login`;
export const HOME_PAGE_URL = `${BASE_URL}/chapters`;

export const dropdowns = [
    { name: 'curriculum-dropdown', urlPattern: /\/api\/curriculums/, content: ['1', '2'], key: 'selectedCurriculum', selectedVal: '2', placeholder: 'Select curriculum' },
    { name: 'grade-dropdown', urlPattern: /\/api\/grades/, content: ['3', '4'], key: 'selectedGrade', selectedVal: '4', placeholder: 'Select grade' },
    { name: 'subject-dropdown', urlPattern: /\/api\/subjects/, content: ['5', '6'], key: 'selectedSubject', selectedVal: '6', placeholder: 'Select subject' },
];

/**
 * The header dropdowns show "Select ..." placeholders until a value is chosen, and chapters/tests are not loaded
 * until all of them have a value. Specs that need the chapters list to load call this before navigating: it
 * stores the first option of each (mocked) dropdown in sessionStorage, which the app restores on load.
 */
export async function preselectDropdowns(page: Page) {
    const selections = dropdowns.map(({ key, content }) => [key, content[0]]);
    await page.addInitScript((pairs) => {
        pairs.forEach(([key, value]) => sessionStorage.setItem(key, value));
    }, selections);
}
