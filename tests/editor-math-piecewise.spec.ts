import { test, expect } from '@playwright/test';

test.use({ storageState: { cookies: [], origins: [] } });

test('math templates insert an editable piecewise equation', async ({ page }) => {
  await page.request.post('http://localhost:8080/dev-login');
  await page.goto('http://localhost:8080/topic/add-problem?topic_id=3');

  await page.locator('#questionDiv .editor').click();
  await page.locator('#questionDiv .mathTemplateBtn').click();
  await page.getByRole('button', { name: 'Piecewise / Cases' }).click();

  const mathField = page.locator('#questionDiv math-field').first();
  await expect(mathField).toBeVisible();
  await expect.poll(() => mathField.evaluate((el: any) => el.getValue('latex'))).toContain('\\begin{cases}');

  await mathField.evaluate((el: any) => {
    el.setValue('f(x)=\\begin{cases}x^2+3x+a,&x\\le 1\\\\bx+2,&x>1\\end{cases}');
  });
  await mathField.focus();
  await page.keyboard.press('Enter');

  await expect(page.locator('#questionDiv .output mjx-container')).toBeVisible();
  await expect(page.locator('#questionDiv .editor')).toContainText('\\begin{cases}');
});
