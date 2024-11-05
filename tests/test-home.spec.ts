import { test, expect } from '@playwright/test';

const HOME_PAGE_URL = 'http://localhost:8080';

test.describe('Homepage Load', () => {

    const dropdowns = [
        { name: 'curriculum-dropdown', urlPattern: /\/api\/curriculums/, content: ['c1', 'c2'], key: 'selectedCurriculum', selectedVal: 'c2' },
        { name: 'grade-dropdown', urlPattern: /\/api\/grades/, content: ['g1', 'g2'], key: 'selectedGrade', selectedVal: 'g1' },
        { name: 'subject-dropdown', urlPattern: /\/api\/subjects/, content: ['s1', 's2'], key: 'selectedSubject', selectedVal: 's2' },
    ];

    test.beforeEach(async ({ page }) => {
        // intercept request to mock response
        dropdowns.forEach(async function ({ urlPattern, content }) {
            await page.route(urlPattern, async route => {
                await route.fulfill({
                    status: 200,
                    body: `<option value="${content[0]}">${content[0]}</option><option value="${content[1]}">${content[1]}</option>`,
                    headers: { 'Content-Type': 'text/html' }
                })
            })
        });

        await page.goto(HOME_PAGE_URL);
    });

    test('should display all 4 tabs with the first tab selected by default', async ({ page }) => {
        const chaptersTab = page.locator('#chapters-tab');

        // Check if all four tabs are visible
        await expect(chaptersTab).toBeVisible();
        await expect(page.locator('#modules-tab')).toBeVisible();
        await expect(page.locator('#books-tab')).toBeVisible();
        await expect(page.locator('#major-tests-tab')).toBeVisible();

        // Check that the first tab is selected by default
        await expect(chaptersTab).toHaveClass(/active/);
    });

    test('Page load makes GET api call to /chapters', async ({ page }) => {
        // Wait for the initial request to complete (on page load)
        const requestPromise = page.waitForRequest(request =>
            request.url().includes('/chapters') && request.method() === 'GET');

        const request = await requestPromise;

        // Verify that the request was made
        expect(request.url()).toContain('/chapters');
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
                        console.log('poll if');
                        // Save original trigger function
                        const originalTrigger = window.htmx.trigger;

                        // Override the htmx.trigger method to introduce a delay
                        window.htmx.trigger = (...args) => {
                            setTimeout(() => {
                                originalTrigger.apply(window.htmx, args);
                            }, 100); // Delay of 100ms
                        };
                        console.log("HTMX trigger delayed by 100ms.");
                    } else {
                        // Retry after a short delay if HTMX is not yet loaded
                        setTimeout(pollForHTMX, 30);
                    }
                };
                pollForHTMX();
            });

            const page = await context.newPage();

            // mock response for new page creted above
            await page.route(urlPattern, async route => {
                await route.fulfill({
                    status: 200,
                    body: `<option value="${content[0]}">${content[0]}</option><option value="${content[1]}">${content[1]}</option>`,
                    headers: { 'Content-Type': 'text/html' }
                })
            })

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
    test.beforeEach(async ({ page }) => {
        await page.goto(HOME_PAGE_URL);

        // Wait for the initial request to complete (on page load)
        page.waitForRequest(request =>
            request.url().includes('/chapters') && request.method() === 'GET');
    });

    test('Clicking Chapters tab makes GET api call to /chapters', async ({ page }) => {
        // Function to check if the request is for /chapters
        const isChaptersRequest = (request) =>
            request.url().includes('/chapters') && request.method() === 'GET';

        // Click on the "Chapters" tab
        await page.click('#chapters-tab');

        // Wait for the 2nd request to be made
        const requestPromise = page.waitForRequest(isChaptersRequest);
        const request = await requestPromise;

        // Verify that the request was made
        expect(request.url()).toContain('/chapters');
    });

    const tabs = [
        { name: 'chapters', urlPattern: /\/chapters/, content: 'Chapters content loaded' },
        { name: 'books', urlPattern: /\/books/, content: 'Books content loaded' },
        { name: 'modules', urlPattern: /\/modules/, content: 'Modules content loaded' },
        { name: 'major-tests', urlPattern: /\/major-tests/, content: 'Major tests content loaded' }
    ];

    tabs.forEach(function ({ name, urlPattern, content }) {
        test(`Clicking ${name} tab loads ${urlPattern} api response in div having id content`, async ({ page }) => {
            // Intercept the HTMX GET request to mock response
            await page.route(urlPattern, async route => {
                await route.fulfill({
                    status: 200,
                    body: `<p>${content}</p>`,
                    headers: { 'Content-Type': 'text/html' }
                });
            });

            // Click on the tab
            await page.click(`#${name}-tab`);

            // Verify that the content is updated in the target element
            await expect(page.locator('#content')).toHaveText(content);
        });
    });
});