import { test, expect, Page } from '@playwright/test';
import { BASE_URL } from './utils';

// The Test Configuration dialog normally opens over the edit screen. Here it opens over the home screen, with the
// two edit-screen functions it calls (and the rule check of the problems) replaced by recorders.
test.use({ storageState: { cookies: [], origins: [] } });

const SAVED_PAIRS = '[{"curriculum_id":1,"grade_id":3},{"curriculum_id":1,"grade_id":4}]';

async function mockOptions(page: Page, urlPattern: RegExp, values: number[]) {
  await page.route(urlPattern, async route => {
    await route.fulfill({
      status: 200,
      contentType: 'text/html',
      body: values.map(v => `<option value="${v}">${v}</option>`).join(''),
    });
  });
}

async function openDialog(page: Page, ruleBody: object = { TestRule: { config: {} }, jee_advanced_exam_id: 5 }) {
  await page.request.post(`${BASE_URL}/dev-login`);
  await mockOptions(page, /\/api\/curriculums/, [1, 2, 3]);
  await mockOptions(page, /\/api\/grades/, [1, 2, 3, 4, 5]);
  await mockOptions(page, /\/api\/exams/, [1, 2]);

  const ruleChecks = { count: 0 };
  await page.route(/\/tests\/validate-test/, async route => {
    ruleChecks.count++;
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(ruleBody) });
  });

  await page.goto(`${BASE_URL}/home`);
  await page.evaluate((query) => {
    const w = window as any;
    const host = document.createElement('div');
    host.id = 'cfg-host';
    document.body.appendChild(host);
    w.__applied = null;
    // Defined by the edit screen, which the dialog relies on
    w.readResponseOrThrow = async (res: Response) => {
      if (!res.ok) throw new Error(await res.text());
      return res.text();
    };
    w.validateProblemRows = () => [];
    w.applyTestConfiguration = (configuration: unknown, ruleChange: unknown) => {
      w.__applied = { configuration, ruleChange };
    };
    return w.htmx.ajax('GET', `/tests/add-test-dialog?${query}`, { target: '#cfg-host', swap: 'innerHTML' });
  }, `mode=configuration&subtype=chapter_test&exam_id=1&curriculum_grades=${encodeURIComponent(SAVED_PAIRS)}`);

  // wait until every select has picked its saved option
  const grades = page.locator("select[name='grade[]']");
  await expect(grades.nth(0)).toHaveValue('3');
  await expect(grades.nth(1)).toHaveValue('4');
  await expect(page.locator("select[name='curriculum[]']").nth(0)).toHaveValue('1');
  await expect(page.locator('#modal-examType')).toHaveValue('1');
  await expect(page.locator('#modal-testType')).toHaveValue('chapter_test');
  return ruleChecks;
}

const confirm = (page: Page) => page.locator('#configuration-confirm-btn');
const applied = (page: Page) => page.evaluate(() => (window as any).__applied);

test('Confirm starts disabled and follows whether the choice differs from what the dialog opened with', async ({ page }) => {
  await openDialog(page);
  await expect(confirm(page)).toBeDisabled();

  const firstGrade = page.locator("select[name='grade[]']").nth(0);
  await firstGrade.selectOption('5');
  await expect(confirm(page)).toBeEnabled();

  await firstGrade.selectOption('3');
  await expect(confirm(page)).toBeDisabled();
});

test('the blank test type is not offered', async ({ page }) => {
  await openDialog(page);
  await expect(page.locator('#modal-testType option[value=""]')).toHaveCount(0);
});

test('a curriculum-grade change is applied without checking a rule', async ({ page }) => {
  const ruleChecks = await openDialog(page);

  await page.locator("select[name='grade[]']").nth(0).selectOption('5');
  await confirm(page).click();

  expect(await applied(page)).toEqual({
    configuration: {
      subtype: 'chapter_test',
      examId: 1,
      curriculumGrades: [{ curriculum_id: 1, grade_id: 5 }, { curriculum_id: 1, grade_id: 4 }],
    },
    ruleChange: null,
  });
  expect(ruleChecks.count).toBe(0);
  await expect(page.locator('#addTestModal')).toHaveCount(0);
});

test('an exam that fits is checked, then applied with its rule', async ({ page }) => {
  const ruleChecks = await openDialog(page);

  await page.selectOption('#modal-examType', '2');
  await expect(page.locator('#configuration-rule-status')).toContainText('The test fits this exam and test type.');
  await expect(confirm(page)).toBeEnabled();
  await confirm(page).click();

  expect(ruleChecks.count).toBe(1);
  expect(await applied(page)).toEqual({
    configuration: {
      subtype: 'chapter_test',
      examId: 2,
      curriculumGrades: [{ curriculum_id: 1, grade_id: 3 }, { curriculum_id: 1, grade_id: 4 }],
    },
    ruleChange: { rule: { config: {} }, jeeAdvancedExamId: 5 },
  });
  await expect(page.locator('#addTestModal')).toHaveCount(0);
});

test('an exam without a rule restricts nothing and is applied', async ({ page }) => {
  await openDialog(page, { TestRule: null, jee_advanced_exam_id: 0 });

  await page.selectOption('#modal-examType', '2');
  await expect(page.locator('#configuration-rule-status')).toContainText('No rule is configured');
  await expect(confirm(page)).toBeEnabled();
  await confirm(page).click();

  expect((await applied(page)).ruleChange).toEqual({ rule: null, jeeAdvancedExamId: 0 });
});

test('problems that do not fit the exam block Confirm and are named, until the exam is set back', async ({ page }) => {
  await openDialog(page);
  await page.evaluate(() => {
    (window as any).validateProblemRows = () => {
      const row = document.createElement('tr');
      row.dataset.subject = 'subject-5';
      row.dataset.subtype = 'subtype-mcq';
      row.innerHTML = '<td><a href="/problem?id=1">P-001</a></td>';
      return [{ row, result: { isValid: false, reason: 'Subject is not allowed' } }];
    };
  });

  await page.selectOption('#modal-examType', '2');
  const status = page.locator('#configuration-rule-status');
  await expect(status).toContainText('cannot move to this configuration');
  await expect(status).toContainText('Subject 5');
  await expect(status).toContainText('1 question(s) - Subject is not allowed (P-001)');
  await expect(confirm(page)).toBeDisabled();

  // back on the exam it opened with, there is nothing to check and nothing to confirm
  await page.selectOption('#modal-examType', '1');
  await expect(status).toBeEmpty();
  await expect(confirm(page)).toBeDisabled();
  expect(await applied(page)).toBeNull();
});

test('duplicate curriculum-grade pairs are refused when Confirm is clicked', async ({ page }) => {
  await openDialog(page);

  // the second row becomes a copy of the first
  await page.locator("select[name='grade[]']").nth(1).selectOption('3');
  await expect(confirm(page)).toBeEnabled();

  let message = '';
  page.once('dialog', async dialog => {
    message = dialog.message();
    await dialog.accept();
  });
  await confirm(page).click();

  expect(message).toContain('Duplicate Curriculum');
  expect(await applied(page)).toBeNull();
  await expect(page.locator('#addTestModal')).toHaveCount(1);
});
