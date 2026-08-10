function compressImage(dataUrl, { maxDimension = 1200, quality = 0.90 } = {}) {
    return new Promise((resolve) => {
        const img = new Image();
        img.onload = function () {
            let { width, height } = img;
            if (width > maxDimension || height > maxDimension) {
                if (width >= height) {
                    height = Math.round(height * maxDimension / width);
                    width = maxDimension;
                } else {
                    width = Math.round(width * maxDimension / height);
                    height = maxDimension;
                }
            }
            const canvas = document.createElement('canvas');
            canvas.width = width;
            canvas.height = height;
            canvas.getContext('2d').drawImage(img, 0, 0, width, height);
            resolve(canvas.toDataURL('image/webp', quality));
        };
        img.src = dataUrl;
    });
}

function insertImage(event, editor) {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = function (e) {
        compressImage(e.target.result).then(compressedSrc => {
            const img = document.createElement('img');
            img.src = compressedSrc;

            const range = window.getSelection().getRangeAt(0);
            const block = buildFloatBlock(img, 'left', editor);
            applyImageSize(img, 100);

            range.insertNode(block);
            placeCaretAfterImage(img);

            renderMath(editor);
        });
    };
    reader.readAsDataURL(file);

    event.target.value = '';
}

function initImageEditing(editor, editorWrapper) {
    const imgToolbar = editorWrapper.querySelector('.img-edit-toolbar');
    const resizeOverlay = editorWrapper.querySelector('.img-resize-overlay');
    if (!imgToolbar || !resizeOverlay) return;

    let selectedImg = null;

    function positionUI(img) {
        const wRect = editorWrapper.getBoundingClientRect();
        const iRect = img.getBoundingClientRect();

        resizeOverlay.style.top    = (iRect.top - wRect.top) + 'px';
        resizeOverlay.style.left   = (iRect.left - wRect.left) + 'px';
        resizeOverlay.style.width  = iRect.width + 'px';
        resizeOverlay.style.height = iRect.height + 'px';

        const toolbarH = imgToolbar.offsetHeight || 32;
        const top = iRect.top - wRect.top;
        const toolbarTop = top - toolbarH - 4;
        imgToolbar.style.top  = (toolbarTop < 0 ? top + iRect.height + 4 : toolbarTop) + 'px';
        imgToolbar.style.left = (iRect.left - wRect.left) + 'px';
    }

    function selectImage(img) {
        selectedImg = img;
        img.classList.add('img-selected');
        resizeOverlay.classList.add('active');
        imgToolbar.classList.add('active');
        positionUI(img);
    }

    function deselectImage() {
        if (selectedImg) selectedImg.classList.remove('img-selected');
        selectedImg = null;
        resizeOverlay.classList.remove('active');
        imgToolbar.classList.remove('active');
    }

    editor.addEventListener('click', (e) => {
        if (e.target.tagName === 'IMG') {
            selectImage(e.target);
        } else {
            deselectImage();
        }
    });

    editor.addEventListener('mousedown', (e) => {
        if (e.target.tagName === 'IMG') return;

        for (const img of editor.querySelectorAll('img')) {
            const float = img.style.float;
            if (float !== 'left' && float !== 'right') continue;

            const rect = img.getBoundingClientRect();
            const besideVertically = e.clientY >= rect.top && e.clientY <= rect.bottom;
            const besideLeft  = float === 'right' && e.clientX < rect.left;
            const besideRight = float === 'left' && e.clientX > rect.right;

            if (besideVertically && (besideLeft || besideRight)) {
                placeCaretAfterImage(img);
                e.preventDefault();
                return;
            }
        }
    });

    editor.addEventListener('scroll', () => {
        if (selectedImg) positionUI(selectedImg);
    });

    imgToolbar.addEventListener('mousedown', (e) => e.preventDefault());
    imgToolbar.addEventListener('click', (e) => {
        if (!selectedImg) return;
        const btn = e.target.closest('button');
        if (!btn) return;

        if (btn.dataset.imgAlign) {
            applyImageAlign(selectedImg, btn.dataset.imgAlign, editor);
        } else if (btn.dataset.imgSize) {
            applyImageSize(selectedImg, parseInt(btn.dataset.imgSize));
        } else if (btn.classList.contains('img-delete-btn')) {
            const block = getImageBlock(selectedImg);
            if (block) block.remove();
            else selectedImg.remove();
            deselectImage();
            renderMath(editor);
            return;
        }

        renderMath(editor);
        requestAnimationFrame(() => {
            if (selectedImg) positionUI(selectedImg);
        });
    });

    resizeOverlay.addEventListener('mousedown', (e) => {
        const handle = e.target.closest('.img-resize-handle');
        if (!handle || !selectedImg) return;
        e.preventDefault();

        const dir = handle.dataset.dir;
        const startX = e.clientX;
        const startW = selectedImg.getBoundingClientRect().width;

        const onMouseMove = (ev) => {
            const dx = ev.clientX - startX;
            const newW = Math.max(30, dir.includes('e') ? startW + dx : startW - dx);
            selectedImg.style.width = newW + 'px';
            selectedImg.style.maxWidth = newW + 'px';
            selectedImg.style.height = 'auto';
            positionUI(selectedImg);
        };

        const onMouseUp = () => {
            window.removeEventListener('mousemove', onMouseMove);
            window.removeEventListener('mouseup', onMouseUp);
        };

        window.addEventListener('mousemove', onMouseMove);
        window.addEventListener('mouseup', onMouseUp);
    });

    const closeScope = editorWrapper.closest('form') || editorWrapper.closest('#content') || editorWrapper;
    closeScope.addEventListener('click', (e) => {
        if (!editorWrapper.contains(e.target)) {
            deselectImage();
        }
    });
}

function getImageBlock(img) {
    return img.closest('.editor-img-float, .editor-img-row, .editor-img-justify');
}

/** Unwrap legacy flex span so text flows naturally in the paragraph. */
function flattenTextSpan(block) {
    const span = block.querySelector('.editor-img-text');
    if (!span) return;

    while (span.firstChild) {
        block.insertBefore(span.firstChild, span);
    }
    span.remove();
    block.classList.remove('editor-img-row', 'editor-img-left', 'editor-img-right');
    block.style.display = '';
    block.style.flexDirection = '';
}

function buildFloatBlock(img, align, editor) {
    const block = document.createElement('p');
    block.className = 'editor-img-float';
    block.classList.add(align === 'right' ? 'editor-img-right' : 'editor-img-left');

    const parent = img.parentElement;
    if (parent && parent !== block) {
        parent.insertBefore(block, img);
        block.appendChild(img);
        if (parent !== editor && parent.childNodes.length === 0 && parent.tagName === 'P') {
            parent.remove();
        }
    } else {
        block.appendChild(img);
    }

    applyFloatStyles(img, align);
    ensureTextAfterImage(img);
    return block;
}

function ensureImageBlock(img, editor) {
    let block = getImageBlock(img);
    if (block) {
        flattenTextSpan(block);
        return block;
    }

    block = document.createElement('p');
    block.className = 'editor-img-float';
    block.style.margin = '0.5em 0';

    const parent = img.parentElement;
    if (parent === editor) {
        editor.insertBefore(block, img);
        block.appendChild(img);
    } else if (parent) {
        parent.insertBefore(block, img);
        block.appendChild(img);
        if (parent !== editor && parent.childNodes.length === 0 && parent.tagName === 'P') {
            parent.remove();
        }
    }

    return block;
}

/** Zero-width space lets the caret sit after a floated image; stripped before save. */
var IMAGE_CARET_MARKER = '\u200B';
var IMAGE_CARET_MARKER_ENTITY = `&#${IMAGE_CARET_MARKER.charCodeAt(0)};`;

function ensureTextAfterImage(img) {
    const next = img.nextSibling;
    if (next?.nodeType === Node.TEXT_NODE) {
        if (!next.textContent.includes(IMAGE_CARET_MARKER)) {
            next.textContent = IMAGE_CARET_MARKER + next.textContent;
        }
        return next;
    }

    const textNode = document.createTextNode(IMAGE_CARET_MARKER);
    img.after(textNode);
    return textNode;
}

function stripImageCaretMarkers(html) {
    return html.replaceAll(IMAGE_CARET_MARKER, '').replaceAll(IMAGE_CARET_MARKER_ENTITY, '');
}

function placeCaretAfterImage(img) {
    const textNode = ensureTextAfterImage(img);
    const offset = Math.max(textNode.textContent.length, 1);
    const range = document.createRange();
    const sel = window.getSelection();
    range.setStart(textNode, Math.min(offset, textNode.textContent.length));
    range.collapse(true);
    sel.removeAllRanges();
    sel.addRange(range);
}

function clearInlineImageStyles(img) {
    img.style.verticalAlign = '';
}

function applyFloatStyles(img, align) {
    clearInlineImageStyles(img);
    img.style.flexBasis = '';
    img.style.display = 'block';

    if (align === 'left') {
        img.style.float = 'left';
        img.style.margin = '0 0.75em 0.5em 0';
    } else if (align === 'right') {
        img.style.float = 'right';
        img.style.margin = '0 0 0.5em 0.75em';
    }
}

function applyImageAlign(img, align, editor) {
    if (align === 'inline') {
        applyInlineImage(img);
        placeCaretAfterImage(img);
        return;
    }

    if (align === 'justify') {
        clearInlineImageStyles(img);
        let block = getImageBlock(img);
        flattenTextSpan(block || img.parentElement);

        if (!block || !block.classList.contains('editor-img-justify')) {
            block = ensureImageBlock(img, editor);
            block.className = 'editor-img-justify';
            block.style.margin = '0.5em 0';
        }

        img.style.float = 'none';
        img.style.width = '100%';
        img.style.maxWidth = '100%';
        img.style.display = 'block';
        img.style.margin = '0.5em 0';
        return;
    }

    const block = buildFloatBlock(img, align, editor);
    block.className = 'editor-img-float';
    block.classList.toggle('editor-img-left', align === 'left');
    block.classList.toggle('editor-img-right', align === 'right');

    if (!img.style.width) {
        applyImageSize(img, 100);
    }

    applyFloatStyles(img, align);
    placeCaretAfterImage(img);
}

function applyInlineImage(img) {
    const block = getImageBlock(img);
    if (block) {
        flattenTextSpan(block);
        block.replaceWith(img);
    }

    img.style.float = 'none';
    img.style.display = 'inline-block';
    img.style.verticalAlign = 'middle';
    img.style.margin = '0 0.25em';
    img.style.height = 'auto';
    img.style.flexBasis = '';
}

function applyImageSize(img, percent) {
    img.style.width = percent + '%';
    img.style.maxWidth = percent + '%';
    img.style.height = 'auto';
    img.style.flexBasis = '';
}
