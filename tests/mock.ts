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
            body: `
                <body>
                    <div class="m-7" id="nav-tabContent">
                        <div id="content">
                            <p>${content}</p>
                        </div>
                    </div>
                </body>
            `,
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

export async function mockChaptersApiUsingHtml(page: Page, filepath: string, chapterObj: {
    id: string;
    name: string; code: string;
}) {
    let htmlData = fs.readFileSync(filepath, 'utf-8');
    htmlData = htmlData.replace("{{.ID}}", chapterObj.id);
    htmlData = htmlData.replace("{{.Code}}", chapterObj.code);
    htmlData = htmlData.replace("{{.Name}}", chapterObj.name);

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

export async function mockEditChapterUsingHtml(page: Page, filepath: string, baseFilepath: string,
    chapterObj: { id: string; name: string; code: string }) {
    let contentData = fs.readFileSync(filepath, 'utf-8');
    contentData = contentData.replace("{{.ChapterPtr.ID}}", chapterObj.id);
    contentData = contentData.replace("{{.ChapterPtr.Name}}", chapterObj.name);
    contentData = contentData.replace("{{.ChapterPtr.Code}}", chapterObj.code);

    let baseData = fs.readFileSync(baseFilepath, 'utf-8');
    const extractedBody = baseData.match(/<body>([\s\S]*?)<\/body>/);
    if (extractedBody) {
        baseData = extractedBody[1].trim();
    }
    baseData = baseData.replace(/{{ if .InitialLoad }}[\s\S]*?{{ end }}/, '');
    const mergedData = baseData.replace(/{{ block "content" . }}[\s\S]*?{{ end }}/, contentData);

    // Intercept the API request and respond with the HTML data
    await page.route(/\/edit-chapter/, async route => {
        await route.fulfill({
            status: 200,
            contentType: 'text/html',
            body: mergedData
        });
    });
}

export async function mockUpdateChapterUsingHtml(page: Page, filepath: string) {
    let htmlData = fs.readFileSync(filepath, 'utf-8');

    // Intercept the API request and respond with the HTML data
    await page.route(/\/update-chapter/, async route => {
        await route.fulfill({
            status: 200,
            contentType: 'text/html',
            body: htmlData,
        });
    });
}