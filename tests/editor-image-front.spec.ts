import { test, expect } from '@playwright/test';

test.use({ storageState: { cookies: [], origins: [] } });

test('image toolbar can keep a diagram inline with its label', async ({ page }) => {
  await page.request.post('http://localhost:8080/dev-login');
  await page.goto('http://localhost:8080/topic/add-problem?topic_id=3');

  const editor = page.locator('#questionDiv .editor');
  await editor.evaluate((el) => {
    el.innerHTML = `
      <table class="w-full border border-black border-collapse my-2">
        <tbody>
          <tr>
            <td class="border border-black w-16 h-8 text-center">
              <div style="text-align: left;">
                (A)
                <p class="editor-img-float editor-img-left">
                  <img alt="diagram" src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAoAAAAKCAIAAAACUFjqAAAAFElEQVR4AWP8z8Dwn4EIwESJ5lEDAN9OCJm5N4+jAAAAAElFTkSuQmCC" style="display: block; float: left; margin: 0px 0.75em 0.5em 0px; width: 40px; max-width: 40px; height: auto;">
                </p>
              </div>
            </td>
            <td class="border border-black w-16 h-8 text-center">i) 2R</td>
          </tr>
        </tbody>
      </table>`;
  });

  const image = page.locator('#questionDiv .editor img[alt="diagram"]');
  await image.click();
  await page.locator('#questionDiv').getByTitle('Inline With Text').click();

  await expect(image).toHaveCSS('display', 'inline-block');
  await expect(image).toHaveCSS('float', 'none');

  await expect(editor.locator('p.editor-img-float')).toHaveCount(0);

  await expect
    .poll(() => editor.evaluate((el: any) => window.getEditorHtml(el)))
    .toContain('(A)');
  await expect
    .poll(() => editor.evaluate((el: any) => window.getEditorHtml(el)))
    .toContain('display: inline-block');
  await expect
    .poll(() => editor.evaluate((el: any) => window.getEditorHtml(el)))
    .not.toContain('img-selected');
});

test('resizing the editor keeps the preview matched', async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 900 });
  await page.request.post('http://localhost:8080/dev-login');
  await page.goto('http://localhost:8080/topic/add-problem?topic_id=3');

  const editor = page.locator('#questionDiv .editor');
  const preview = page.locator('#questionDiv .output');
  const wrapper = page.locator('#questionDiv .editor-wrapper');

  await expect(editor).toHaveCSS('resize', 'both');

  await editor.evaluate((el: HTMLElement) => {
    el.style.width = '640px';
    el.style.height = '360px';
  });

  await expect.poll(async () => {
    const [editorBox, previewBox, wrapperBox] = await Promise.all([
      editor.boundingBox(),
      preview.boundingBox(),
      wrapper.boundingBox(),
    ]);
    return {
      editorWidth: Math.round(editorBox?.width ?? 0),
      editorHeight: Math.round(editorBox?.height ?? 0),
      previewWidth: Math.round(previewBox?.width ?? 0),
      previewHeight: Math.round(previewBox?.height ?? 0),
      wrapperWidth: Math.round(wrapperBox?.width ?? 0),
    };
  }).toEqual({
    editorWidth: 640,
    editorHeight: 360,
    previewWidth: 640,
    previewHeight: 360,
    wrapperWidth: 640,
  });
});

test('resizing the editor cannot push the preview outside the problem card', async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 900 });
  await page.request.post('http://localhost:8080/dev-login');
  await page.goto('http://localhost:8080/topic/add-problem?topic_id=3');

  const editor = page.locator('#questionDiv .editor');
  const preview = page.locator('#questionDiv .output');
  const form = page.locator('#content > form');

  await editor.evaluate((el: HTMLElement) => {
    el.style.width = '1200px';
    el.style.height = '360px';
  });
  await page.waitForTimeout(300);

  await expect.poll(async () => {
    const [editorBox, previewBox, formBox] = await Promise.all([
      editor.boundingBox(),
      preview.boundingBox(),
      form.boundingBox(),
    ]);
    const previewRight = (previewBox?.x ?? 0) + (previewBox?.width ?? 0);
    const formRight = (formBox?.x ?? 0) + (formBox?.width ?? 0);
    const overflow = Math.round(previewRight - formRight);
    const widthDelta = Math.abs(Math.round((previewBox?.width ?? 0) - (editorBox?.width ?? 0)));
    return overflow <= 0 && widthDelta <= 1;
  }).toBe(true);
});

test('problem editor page uses the available horizontal space', async ({ page }) => {
  await page.setViewportSize({ width: 1600, height: 900 });
  await page.request.post('http://localhost:8080/dev-login');
  await page.goto('http://localhost:8080/topic/add-problem?topic_id=3');

  const content = page.locator('#content');
  const form = page.locator('#content > form');

  await expect.poll(async () => {
    const [contentBox, formBox] = await Promise.all([
      content.boundingBox(),
      form.boundingBox(),
    ]);
    return Math.round((contentBox?.width ?? 0) - (formBox?.width ?? 0));
  }).toBeLessThanOrEqual(4);
});
