import { test, expect } from "@playwright/test";
import { dropdowns, HOME_PAGE_URL } from "./utils";
import { mockChaptersApi, mockCreateChapterApi, mockDropdownApi } from './mock';

test.describe('Add Chapter', () => {

    test.beforeEach(async ({ page }) => {

        // intercept request to mock response
        dropdowns.forEach(async function ({ urlPattern, content }) {
            await mockDropdownApi(page, urlPattern, content);
        });

        // mock get chapters api response
        await mockChaptersApi(page);

        await page.goto(HOME_PAGE_URL);
        const addChapterLink = page.locator('#addChapterLink');
        await addChapterLink.click();
    });

    test('Submitting new chapter form posts data to server', async ({ page }) => {
        const chptrCode = 'CHAP001';
        const chptrName = 'New Chapter';

        // mock create chapter api response
        await mockCreateChapterApi(page, chptrCode, chptrName);

        const inputCode = page.locator('[name="code"]');
        const inputName = page.locator('[name="name"]');

        await inputCode.fill(chptrCode);
        await inputName.fill(chptrName);

        // Intercept and wait for the POST request
        const apiReqPromise = page.waitForRequest((request) =>
            request.url().includes('/create-chapter') && request.method() === 'POST'
        );

        await page.click('button[type="submit"]');

        // Await the API request and verify it was made
        const apiRequest = await apiReqPromise;
        expect(apiRequest).toBeTruthy();

        // verify the request payload
        const postData = apiRequest.postDataJSON();
        expect(postData).toMatchObject({
            code: chptrCode,
            name: chptrName,
        });
        // Check for additional parameters, verifying they exist but not checking values
        expect(postData).toHaveProperty('curriculum-dropdown');
        expect(postData).toHaveProperty('grade-dropdown');
        expect(postData).toHaveProperty('subject-dropdown');

        // Verify that the new row was added to the table as the last row
        const lastRow = page.locator('#chapterTableBody > tr:last-child');

        // Verify the text within the last row contains the expected values
        await expect(lastRow).toContainText(chptrCode);
        await expect(lastRow).toContainText(chptrName);

        // Verify that input fields are reset after successful creation of new chapter
        await expect(inputCode).toBeEmpty();
        await expect(inputName).toBeEmpty();
    });
});