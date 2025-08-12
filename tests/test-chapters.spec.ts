import { test, expect } from "@playwright/test";
import { dropdowns, HOME_PAGE_URL } from "./utils";
import * as mock from "./mock";

test('verify that all table headers, add new chapter link are available and form is hidden', async ({ page }) => {

    await page.goto(HOME_PAGE_URL);

    // Click on the "Chapters" tab
    await page.click('#chapters-tab');

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

    // Click on the "Chapters" tab
    await page.click('#chapters-tab');
    
    const addChapterLink = page.locator('#addChapterLink');
    await addChapterLink.click();

    const addChapterForm = page.locator('#addChapterForm');
    await expect(addChapterForm).toBeVisible();

    await addChapterLink.click();
    await expect(addChapterForm).toBeHidden();
});

// TODO: Update following test once sorting state management is shifted from server to client
/*test('verify correct fa-sort icon for columns', async ({ page }) => {

    const columns = [
        { headerLocator: 'th a[hx-get$="sortColumn=1"] i.fas', sortParam: '1' },
        { headerLocator: 'th a[hx-get$="sortColumn=2"] i.fas', sortParam: '2' },
        { headerLocator: 'th a[hx-get$="sortColumn=3"] i.fas', sortParam: '3' },
    ];

    await page.goto(HOME_PAGE_URL);
    // Click on the "Chapters" tab
    await page.click('#chapters-tab');

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
});*/

test.describe('Chapter list Row', () => {

    const chapterObj = {
        id: '1',
        name: 'n1',
        code: 'c1'
    };

    test.beforeEach(async ({ page }) => {
        // mock dropdown apis to trigger chapter list loading
        dropdowns.forEach(async function ({ urlPattern, content }) {
            await mock.mockDropdownApi(page, urlPattern, content);
        });
        // mock chapter list api
        await mock.mockChaptersApiUsingHtml(page, 'web/html/chapter_row.html', chapterObj);

        await page.goto(HOME_PAGE_URL);

        // Click on the "Chapters" tab
        await page.click('#chapters-tab');
    });

    test('verify delete chapter functionality', async ({ page }) => {

        // Set up a listener to capture the dialog
        page.once('dialog', async dialog => {
            // verify the dialog message
            expect(dialog.message()).toBe(`Are you sure you want to delete chapter ${chapterObj.name}?`);

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

    test('verify edit chapter functionality', async ({ page }) => {
        const row = page.locator('tbody tr');
        // Verify initial chapter code & name on chapter list screen row
        await expect(row).toContainText(chapterObj.code);
        await expect(row).toContainText(chapterObj.name);

        mock.mockEditChapterUsingHtml(page, 'web/html/edit_chapter.html', 'web/html/home.html', chapterObj);

        let apiReqPromise = page.waitForRequest((request) =>
            request.url().includes('/edit-chapter') && request.method() === 'GET'
        );

        // locate edit button for the chapter row (we have one row only in mocked response, 
        // hence no need to select specific row)
        const editBtn = page.locator('tbody tr td button[hx-get]');
        await editBtn.click();

        // Await the API request and verify it was made
        let apiRequest = await apiReqPromise;
        expect(apiRequest).toBeTruthy();

        // verify default chapter code and name in edit screen
        const inputName = page.locator('#name');
        await expect(inputName).toBeVisible();
        const nameValue = await inputName.inputValue();
        expect(nameValue).toBe(chapterObj.name);

        const inputCode = page.locator('#code');
        await expect(inputCode).toBeVisible();
        const codeValue = await inputCode.inputValue();
        expect(codeValue).toBe(chapterObj.code);

        mock.mockUpdateChapterUsingHtml(page, 'web/html/update_success.html');

        // update name and code
        chapterObj.name = 'new name';
        chapterObj.code = 'new code';
        await inputName.fill(chapterObj.name);
        await inputCode.fill(chapterObj.code);

        // Intercept and wait for the PATCH request
        apiReqPromise = page.waitForRequest(request =>
            request.url().includes('/update-chapter') && request.method() === 'PATCH'
        );
        await page.click('button[type="submit"]');
        
        // Await the API request and verify it was made
        apiRequest = await apiReqPromise;
        expect(apiRequest).toBeTruthy();

        // verify the request payload
        const postData = apiRequest.postDataJSON();
        expect(postData).toMatchObject({
            code: chapterObj.code,
            name: chapterObj.name
        });

        // mock chapters api again to have updated chapter values
        await mock.mockChaptersApiUsingHtml(page, 'web/html/chapter_row.html', chapterObj);

        // Verify updated chapter code & name on chapter list screen row
        await expect(row).toContainText(chapterObj.code);
        await expect(row).toContainText(chapterObj.name);
    });
});