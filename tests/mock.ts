import { Page } from "@playwright/test"
import * as fs from "fs";

export async function mockDropdownApi(page: Page, urlPattern: RegExp, content: string[]) {
    await page.route(urlPattern, async route => {
        await route.fulfill({
            status: 200,
            body: `<option value="${content[0]}">${content[0]}</option><option value="${content[1]}">${content[1]}</option>`,
            headers: { 'Content-Type': 'text/html' }
        })
    })
}

export async function mockTabContentApi(page: Page, urlPattern: RegExp, content: string) {
    await page.route(urlPattern, async route => {
        await route.fulfill({
            status: 200,
            body: `<p>${content}</p>`,
            headers: { 'Content-Type': 'text/html' }
        });
    });
}

export async function mockChaptersApi(page: Page) {
    await page.route(/\/api\/chapters/, async route => {
        await route.fulfill({
            status: 200,
            body: `<tr><td>c1</td><td>n1</td><td>1</td></tr>
            <tr><td>c3</td><td>n3</td><td>3</td></tr>
            <tr><td>c2</td><td>n2</td><td>2</td></tr>`,
            headers: { 'Content-Type': 'text/html' }
        });
    });
}

export async function mockChaptersApiUsingHtml(page: Page, filepath: string) {
    const htmlData = fs.readFileSync(filepath, 'utf-8');

    // Intercept the API request and respond with the HTML data
    await page.route(/\/api\/chapters/, async route => {
        await route.fulfill({
            status: 200,
            contentType: 'text/html',
            body: htmlData,
        });
    });
}

export async function mockCreateChapterApi(page: Page, chptrCode: string, chptrName: string) {
    await page.route(/\/create-chapter/, async route => {
        await route.fulfill({
            status: 200,
            body: `<tr><td>${chptrCode}</td><td>${chptrName}</td></tr>`,
            headers: { 'Content-Type': 'text/html' }
        });
    });    
}

export async function mockDeleteChapterApi(page: Page) {
    await page.route(/\/delete-chapter/, async route => {
        await route.fulfill({
            status: 200
        });
    });    
}