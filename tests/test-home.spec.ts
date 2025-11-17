import { test, expect } from '@playwright/test';
import { dropdowns, HOME_PAGE_URL } from './utils';
import { mockDropdownApi, mockTabContentApi } from './mock';

test.describe('Homepage Load', () => {

    test.beforeEach(async ({ page }) => {
        // intercept request to mock response
        dropdowns.forEach(async function ({ urlPattern, content }) {
            await mockDropdownApi(page, urlPattern, content);
        });
        await page.goto(HOME_PAGE_URL);
    });

    test('should display all 2 tabs', async ({ page }) => {
        // Check if all four tabs are visible
        await expect(page.locator('#chapters-tab')).toBeVisible();
        await expect(page.locator('#tests-tab')).toBeVisible();
    });

    for (const { name, urlPattern, content } of dropdowns) {
        test(`${name} is populated with options retrieved by calling ${urlPattern}`, async ({ page }) => {
            const dropdown = page.locator(`#${name}`);
            const options = dropdown.locator('option');

            await expect(options).toHaveCount(2);

            for (let index = 0; index < content.length; index++) {
                await expect(options.nth(index)).toContainText(content[index]);
            }
        });
    }

    for (const { name, urlPattern, content, key, selectedVal } of dropdowns) {
        test(`${name} triggers onLoaded event & maintains its state from sessionStorage`, async ({ browser }) => {
            const context = await browser.newContext();

            await context.addInitScript(([key, selectedVal]) => {
                // Set sessionStorage before page load
                sessionStorage.setItem(key, selectedVal);
            }, [key, selectedVal]);

            // Poll for HTMX, and delay the trigger function once HTMX is available
            await context.addInitScript(() => {
                const pollForHTMX = () => {
                    if (window.htmx && window.htmx.trigger) {
                        // Save original trigger function
                        const originalTrigger = window.htmx.trigger;

                        // Override the htmx.trigger method to introduce a delay
                        window.htmx.trigger = (...args) => {
                            setTimeout(() => {
                                originalTrigger.apply(window.htmx, args);
                            }, 1000); // Delay of 1000ms
                        };
                        console.log("HTMX trigger delayed by 1000ms.");
                    } else {
                        // Retry after a short delay if HTMX is not yet loaded
                        setTimeout(pollForHTMX, 30);
                    }
                };
                pollForHTMX();
            });

            const page = await context.newPage();

            // mock response for new page creted above
            await mockDropdownApi(page, urlPattern, content);

            await page.goto(HOME_PAGE_URL);

            // Wait for the custom 'onLoaded' event to trigger on the page
            const eventTriggered = await page.evaluate((id) => {
                return new Promise((resolve) => {
                    const element = document.getElementById(id);
                    if (element) {
                        element.addEventListener('onLoaded', () => {
                            resolve(true);
                        });
                    }
                });
            }, name);

            // Ensure the event was triggered
            expect(eventTriggered).toBe(true);

            // verify that dropdown selected value matches with its sessionStorage value
            const dropdownValue = await page.$eval(`#${name}`, (dropdown: HTMLSelectElement) => dropdown.value);
            expect(dropdownValue).toBe(selectedVal);
        });
    }
});

test.describe('Tabs', () => {

    test('Clicking Chapters tab makes GET api call to /chapters', async ({ page }) => {
        // Function to check if the request is for /chapters
        const isChaptersRequest = (request) =>
            request.url().includes('/chapters') && request.method() === 'GET';

        await page.goto(HOME_PAGE_URL);

        // Start waiting for request before clicking
        let reqPromise = page.waitForRequest(isChaptersRequest);
        // Click on the "Chapters" tab
        await page.click('#chapters-tab');
        // Wait for the request to be made
        const request = await reqPromise;

        // Verify that the request was made
        expect(request.url()).toContain('/chapters');
    });

    const tabs = [
        { name: 'chapters', urlPattern: /\/chapters/, content: 'Chapters content loaded' },
        { name: 'tests', urlPattern: /\/tests/, content: 'Tests content loaded' }
    ];

    tabs.forEach(function ({ name, urlPattern, content }) {
        test(`Clicking ${name} tab loads ${urlPattern} api response in div having id content`, async ({ page }) => {
            // Intercept the HTMX GET request to mock response
            await mockTabContentApi(page, urlPattern, content);

            await page.goto(HOME_PAGE_URL);

            // Click on the tab
            await page.click(`#${name}-tab`);

            // Verify that the content is updated in the target element
            await expect(page.locator('#content')).toHaveText(content);
        });
    });
});

dropdowns.forEach(function ({ name, key, selectedVal }) {

    test(`on changing selection in the ${name}, it is updated in session storage and chapters are reloaded`, async ({ page }) => {
        // mock response for dropdown values apis
        dropdowns.forEach(async function ({ urlPattern, content }) {
            await mockDropdownApi(page, urlPattern, content);
        });

        // Set up the network interceptor to capture the request to /api/chapters after changing the dropdown
        const apiRequestPromise = page.waitForRequest(request => {
            if (request.url().includes('/api/chapters') && request.method() === 'GET') {
                const url = new URL(request.url());
                const params = url.searchParams;
                // return true if changed parameter value is present
                return params.get(name) == selectedVal;
            }
            return false;
        });
        // Navigate to the page with the dropdown
        await page.goto(HOME_PAGE_URL);

        // Click on the "Chapters" tab
        await page.click('#chapters-tab');
        /**
         * wait for api is required here; otherwise even before finishing api call it changes dropdown value, resetting
         * session storage CHAPTERS_LOADED_KEY to false. But after this first /api/chapters call due to chapters tab click 
         * finishes resetting flag again to true, hence preventing actual call required on changing dropdown value
         */
        await page.waitForResponse(resp =>
            resp.url().includes('/api/chapters') && resp.status() === 200
        );

        const dropdown = page.locator(`#${name}`);
        // Select a specific option by value
        await dropdown.selectOption({ value: selectedVal });

        // Verify if the correct option is selected
        expect(dropdown).toHaveValue(selectedVal);

        // Check sessionStorage value for corresponding key
        const storedSelectedVal = await page.evaluate((key) => sessionStorage.getItem(key), key);
        expect(storedSelectedVal).toBe(selectedVal);
        
        const apiReq = await apiRequestPromise;
        
        // Verify the API request was made
        expect(apiReq).toBeTruthy();
    });
});