import { test, expect } from "@playwright/test";
import { dropdowns, HOME_PAGE_URL } from "./utils";
import * as mock from "./mock";

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

test('verify delete chapter functionality', async ({ page }) => {
    // mock dropdown apis to trigger chapter list loading
    dropdowns.forEach(async function ({ urlPattern, content }) {
        await mock.mockDropdownApi(page, urlPattern, content);
    });
    // mock chapter list api
    await mock.mockChaptersApiUsingHtml(page, 'web/html/chapter_row.html');

    page.goto(HOME_PAGE_URL);

    // Set up a listener to capture the dialog
    page.once('dialog', async dialog => {
        // verify the dialog message
        expect(dialog.message()).toBe('Are you sure you want to delete chapter {{.Name}}?');

        // dismiss dialog
        await dialog.dismiss();
    });

    // locate delete button for the chapter row (we have one row only in mocked response, 
    // hence no need to select specific row)
    const deleteBtn = page.locator('tbody tr td button[hx-delete]');
    await deleteBtn.click();

    // verify row is not deleted
    let row = page.locator('tbody tr');
    await expect(row).toBeVisible();

    // Set up a listener again to capture the dialog, but this time press positive button
    page.once('dialog', async dialog => {
        // press ok
        await dialog.accept();
    });
    mock.mockDeleteChapterApi(page);
    const apiReqPromise = page.waitForRequest((request) =>
        request.url().includes('/delete-chapter') && request.method() === 'DELETE'
    );

    await deleteBtn.click();
    // Await the API request and verify it was made
    const apiRequest = await apiReqPromise;
    expect(apiRequest).toBeTruthy();

    // verify that row is deleted
    row = page.locator('tbody tr');
    await expect(row).toBeHidden();
});