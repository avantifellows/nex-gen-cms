import { test, expect } from "@playwright/test";
import { dropdowns, HOME_PAGE_URL } from "./utils";
import { mockChaptersApi, mockDropdownApi } from "./mock";

test('verify that all table headers, add new chapter link are available and form is hidden', async ({ page }) => {

    await page.goto(HOME_PAGE_URL);

    const expectedHeaders = [
        'Code',
        'Name',
        'Topics',
        'Chapter Tests',
        'PSV',
        'MOD',
        'CV',
        'CT',
        'Status',
        'Actions'
    ];

    const headers = await page.locator('table thead tr th');
    await expect(headers).toHaveCount(expectedHeaders.length);
    for (let i = 0; i < expectedHeaders.length; i++) {
        await expect(headers.nth(i)).toHaveText(expectedHeaders[i]);
    }

    const addChapterLink = page.locator('#addChapterLink');
    await expect(addChapterLink).toBeVisible();
    await expect(addChapterLink).toHaveText("Add New Chapter");

    const addChapterForm = page.locator('#addChapterForm');
    await expect(addChapterForm).toBeHidden();
});

test('clicking add new chapter opens form and clicking it again hides it', async ({ page }) => {

    await page.goto(HOME_PAGE_URL);

    const addChapterLink = page.locator('#addChapterLink');
    await addChapterLink.click();

    const addChapterForm = page.locator('#addChapterForm');
    await expect(addChapterForm).toBeVisible();

    await addChapterLink.click();
    await expect(addChapterForm).toBeHidden();
});

test('verify correct fa-sort icon for columns', async ({ page }) => {

    const columns = [
        { headerLocator: 'th a[hx-get$="sortColumn=1"] i.fas', sortParam: '1' },
        { headerLocator: 'th a[hx-get$="sortColumn=2"] i.fas', sortParam: '2' },
        { headerLocator: 'th a[hx-get$="sortColumn=3"] i.fas', sortParam: '3' },
    ];

    page.goto(HOME_PAGE_URL);
    await mockChaptersApi(page);

    // Loop over each column header to perform the checks
    for (const column of columns) {
        const { headerLocator, sortParam } = column;
        const sortIcon = page.locator(headerLocator);

        // Verify default state - icon should be fa-sort (unsorted). 
        // \s* is added in rexexp to allow 0 or more white spaces at the end
        await expect(sortIcon).toHaveClass(/fa-sort\s*$/);

        // Click to sort this column in ascending and verify its icon changes to fa-sort-up 
        // and other column icons to fa-sort (not sorted state)
        await page.click(`th a[hx-get$="sortColumn=${sortParam}"]`);
        for (const column of columns) {
            if (sortParam == column.sortParam) {
                await expect(sortIcon).toHaveClass(/fa-sort-up\s*$/);
            } else {
                await expect(page.locator(column.headerLocator)).toHaveClass(/fa-sort\s*$/);
            }
        }

        // Click again to sort this column in descending and verify its icon changes to fa-sort-down
        // and other column icons to fa-sort (not sorted state)
        await page.click(`th a[hx-get$="sortColumn=${sortParam}"]`);
        for (const column of columns) {
            if (sortParam == column.sortParam) {
                await expect(sortIcon).toHaveClass(/fa-sort-down\s*$/);
            } else {
                await expect(page.locator(column.headerLocator)).toHaveClass(/fa-sort\s*$/);
            }
        }
    }
});