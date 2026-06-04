import { test as setup } from '@playwright/test';
import { HOME_PAGE_URL, LOGIN_PAGE_URL } from './utils';
import * as dotenv from 'dotenv';
import { existsSync, mkdirSync } from 'fs';

dotenv.config();

setup('@auth-setup authenticate', async ({ browser }) => {
  const storageDir = 'playwright/.auth';
  if (!existsSync(storageDir)) mkdirSync(storageDir, { recursive: true });

  const context = await browser.newContext({ storageState: undefined });
  const page = await context.newPage();

  // We can't drive a real Google OAuth flow in Playwright. Tests rely on the dev-login bypass,
  // which is enabled by setting DEV_LOGIN_EMAIL on the server. The user named there must already
  // exist in cms_user_permission with is_active = true.
  await page.goto(LOGIN_PAGE_URL);

  const response = await page.request.post(`${LOGIN_PAGE_URL.replace(/\/login$/, '')}/dev-login`);
  if (!response.ok()) {
    throw new Error(`Dev login failed (${response.status()}). Set DEV_LOGIN_EMAIL on the server and ensure the user exists in cms_user_permission.`);
  }

  await page.goto(HOME_PAGE_URL);
  await page.waitForURL(HOME_PAGE_URL);

  await context.storageState({ path: `${storageDir}/user.json` });

  await context.close();
});
