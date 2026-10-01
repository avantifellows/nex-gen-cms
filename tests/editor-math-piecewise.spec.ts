import { test, expect } from '@playwright/test';

test.use({ storageState: { cookies: [], origins: [] } });

test('math templates insert an editable piecewise equation', async ({ page }) => {
  await page.request.post('http://localhost:8080/dev-login');
  await page.goto('http://localhost:8080/topic/add-problem?topic_id=3');

  // #questionDiv holds one editor per language (en/hi/gu/ta); only "en" is visible by default.
  const questionEn = page.locator('#questionDiv .lang-content[data-lang="en"]');

  await questionEn.locator('.editor').click();
  await questionEn.locator('.mathTemplateBtn').click();
  // The dropdown row pairs a "number of rows" input with an "Insert" button; the
  // default row count (2) is enough to produce a \begin{cases} block.
  await questionEn.locator('.mathTemplateDropdown').getByTitle('Insert').click();

  const mathField = questionEn.locator('math-field').first();
  await expect(mathField).toBeVisible();
  await expect.poll(() => mathField.evaluate((el: any) => el.getValue('latex'))).toContain('\\begin{cases}');

  await mathField.evaluate((el: any) => {
    el.setValue('f(x)=\\begin{cases}x^2+3x+a,&x\\le 1\\\\bx+2,&x>1\\end{cases}');
  });
  await mathField.focus();
  await page.keyboard.press('Enter');

  await expect(questionEn.locator('.output mjx-container')).toBeVisible();
  await expect(questionEn.locator('.editor')).toContainText('\\begin{cases}');
});
