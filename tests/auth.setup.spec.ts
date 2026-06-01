import { test as setup, chromium } from '@playwright/test';
import { HOME_PAGE_URL, LOGIN_PAGE_URL } from './utils';
import * as dotenv from 'dotenv';
import { existsSync, mkdirSync } from 'fs';

dotenv.config();

setup('@auth-setup authenticate', async ({ browser }) => {
  // Make sure the folder exists
  const storageDir = 'playwright/.auth';
  if (!existsSync(storageDir)) mkdirSync(storageDir, { recursive: true });

  // Create a fresh context without trying to read existing state
  const context = await browser.newContext({ storageState: undefined });
  const page = await context.newPage();

  // Go to login page
  await page.goto(LOGIN_PAGE_URL);

  // Fill credentials
  await page.fill('[name="username"]', process.env.CMS_USERNAME || '');
  await page.fill('[name="password"]', process.env.CMS_PASSWORD || '');

  // Submit login form
  await page.click('button[type="submit"]');

  // Wait for redirect
  await page.waitForURL(HOME_PAGE_URL);

  // Save authenticated storage state
  await context.storageState({ path: `${storageDir}/user.json` });

  await context.close();
});
