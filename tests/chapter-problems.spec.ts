import { test, expect, Page } from '@playwright/test';
import { BASE_URL, dropdowns } from './utils';
import * as mock from './mock';

// Chapter 3 has to exist on the server these specs run against, like the chapter used by the editor specs.
const CHAPTER_ID = '3';

test.use({ storageState: { cookies: [], origins: [] } });

async function mockHeaderDropdowns(page: Page) {
  for (const { urlPattern, content } of dropdowns) {
    await mock.mockDropdownApi(page, urlPattern, content);
  }
}

// Only the topic requests of the screen under test are answered; everything else reaches the server.
async function mockTopicsApi(page: Page, view: string, body: string) {
  await page.route(/\/api\/topics/, async route => {
    if (route.request().url().includes(`view=${view}`)) {
      await route.fulfill({ status: 200, contentType: 'text/html', body });
    } else {
      await route.continue();
    }
  });
}

test('the chapter Problems tab lists all problems and narrows them locally by an optional topic', async ({ page }) => {
  await page.request.post(`${BASE_URL}/dev-login`);
  await mockHeaderDropdowns(page);
  await mockTopicsApi(page, 'dropdown-all',
    '<option value="" selected>All Topics</option><option value="7">Kinematics</option>');

  const problemRequests: string[] = [];
  await page.route(/\/api\/chapter\/problems/, async route => {
    problemRequests.push(route.request().url());
    await route.fulfill({
      status: 200,
      contentType: 'text/html',
      body: `<tr data-topic-ids="7"><td><input type="checkbox" name="select-problem" value="1"></td><td>P1</td><td>First</td><td></td></tr>
             <tr data-topic-ids=""><td><input type="checkbox" name="select-problem" value="2"></td><td>P2</td><td>Second</td><td></td></tr>`,
    });
  });

  await page.goto(`${BASE_URL}/chapter?id=${CHAPTER_ID}&curriculum-dropdown=1&grade-dropdown=3&subject-dropdown=5`);
  await page.click('#problems-sub-tab');

  await expect(page.locator('#problemTableBody tr')).toHaveCount(2);
  // the chapter alone loads the problems: no topic is needed
  expect(problemRequests[0]).toContain(`chapter-dropdown=${CHAPTER_ID}`);
  // one unpaged request, with no topic
  expect(problemRequests[0]).not.toContain('limit=');
  expect(problemRequests[0]).not.toContain('topic-dropdown');

  // the first row of controls: the topic filter, then the add button
  const controls = page.locator('#problems-topic-filter, button:has-text("+ Add New Problem")');
  await expect(controls.first()).toHaveAttribute('id', 'problems-topic-filter');

  // the Move button needs a selection
  await expect(page.locator('#move-problems-btn')).toBeDisabled();
  await page.locator('input[name="select-problem"]').first().check();
  await expect(page.locator('#move-problems-btn')).toBeEnabled();

  // the topic filter hides rows in the browser, and sends no request
  await page.selectOption('#problems-topic-filter', '7');
  await expect(page.locator('#problemTableBody tr:not(.hidden)')).toHaveCount(1);
  await page.selectOption('#problems-topic-filter', '');
  await expect(page.locator('#problemTableBody tr:not(.hidden)')).toHaveCount(2);
  expect(problemRequests.length).toBe(1);
});

test('adding a problem picks topics from a checklist, with no concepts field', async ({ page }) => {
  await page.request.post(`${BASE_URL}/dev-login`);
  await mockHeaderDropdowns(page);
  await mockTopicsApi(page, 'checklist', `
    <label><input type="checkbox" name="topic-ids" value="7"><span>Kinematics</span></label>
    <label><input type="checkbox" name="topic-ids" value="8"><span>Dynamics</span></label>`);

  await page.goto(`${BASE_URL}/chapter/add-problem?chapter_id=${CHAPTER_ID}`);

  await expect(page.locator('#topics-label')).toHaveText('Topics');
  await expect(page.locator('#topics-checklist input[name="topic-ids"]')).toHaveCount(2);
  await page.locator('#topics-checklist input[value="8"]').check();
  await expect(page.locator('#topics-checklist input[name="topic-ids"]:checked')).toHaveCount(1);

  // topics replaced concepts on this screen
  await expect(page.locator('#concepts-dropdown')).toHaveCount(0);
  await expect(page.getByText('Concepts', { exact: true })).toHaveCount(0);

  // topics of other chapters come from a picker dialog
  await page.getByText('+ Add topic from other chapter').first().click();
  await expect(page.locator('#add-topic-modal')).toBeVisible();
  await expect(page.locator('#add-topic-modal h2')).toHaveText('Add Topics from Another Chapter');
  await expect(page.locator('#add-topic-btn')).toBeDisabled();
});
