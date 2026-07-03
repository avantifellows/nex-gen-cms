function getEditorHtml(editor) {
    const clone = editor.cloneNode(true);
    clone.querySelectorAll('img.img-selected').forEach((img) => {
        img.classList.remove('img-selected');
    });
    if (typeof serializeMathFields === 'function') {
        serializeMathFields(clone);
    }
    if (typeof normalizeRenderedMath === 'function') {
        normalizeRenderedMath(clone);
    }
    let html = clone.innerHTML;
    if (typeof stripImageCaretMarkers === 'function') {
        html = stripImageCaretMarkers(html);
    }
    return html.trim();
}
window.getEditorHtml = getEditorHtml;

window.initializeRichTextEditors = function (root = document) {
    root.querySelectorAll('.container').forEach(container => {
    if (container.dataset.editorInitialized === 'true') return;

    try {
    const editorWrapper = container.querySelector(".editor-wrapper");
    const output = container.querySelector('.output');

    const editor = editorWrapper.querySelector(".editor");

    const initPreview = () => {
        // for edit problem screen update preview on opening page itself, because content must already be there in editor
        const latex = editor.innerHTML.trim();
        if (latex && typeof renderMath === 'function') {
            try {
                renderMath(editor);
            } catch (err) {
                console.error('Editor preview render failed', err);
            }
        }
    };

    const initEditorContent = () => {
        if (typeof normalizeRenderedMath === 'function') {
            normalizeRenderedMath(editor);
        }
        initPreview();
    };

    initEditorContent();

    // listener to display preview
    editor.addEventListener('input', () => {
        clearTimeout(editor.__renderTimeout);
        editor.__renderTimeout = setTimeout(() => {
            renderMath(editor);
        }, 300);
    });

    editor.addEventListener("click", function (e) {
        const target = e.target;
        if (target.tagName === "A") {
            e.preventDefault(); // Prevent default edit behavior
            window.open(target.href, "_blank"); // Open link in new tab
        }
    });

    const toolbar = editorWrapper.querySelector(".toolbar");
    toolbar.querySelector('.fontSelector').addEventListener('change', (e) => {
        document.execCommand('styleWithCSS', false, true);
        document.execCommand('fontName', false, e.target.value);
    });

    toolbar.querySelector('.boldBtn').addEventListener('click', () => {
        document.execCommand('bold');
    });

    toolbar.querySelector('.italicBtn').addEventListener('click', () => {
        document.execCommand('italic');
    });

    toolbar.querySelector('.underlineBtn').addEventListener('click', () => {
        document.execCommand('underline');
    });

    toolbar.querySelector(".fontSizeSelector").addEventListener("change", function () {
        const size = this.value;
        document.execCommand("fontSize", false, "7"); // use size 7 as a placeholder
        const fontElements = editor.getElementsByTagName("font");
        for (let i = 0; i < fontElements.length; i++) {
            if (fontElements[i].size == "7") {
                fontElements[i].removeAttribute("size");
                fontElements[i].style.fontSize = size;
            }
        }
    });

    toolbar.querySelector('.foreColorLabel').addEventListener('mousedown', saveSelection);
    toolbar.querySelector('.backColorLabel').addEventListener('mousedown', saveSelection);

    let savedRange = null;

    function saveSelection() {
        const sel = window.getSelection();
        if (sel.rangeCount > 0 && editor.contains(sel.anchorNode)) {
            savedRange = sel.getRangeAt(0);
        }
    }

    function ensureSelectionInEditor() {
        const sel = window.getSelection();
        if (!sel) return;

        // If we already have a saved range inside this editor, use it.
        if (
            savedRange &&
            editor.contains(savedRange.commonAncestorContainer)
        ) {
            sel.removeAllRanges();
            sel.addRange(savedRange);
            editor.focus();
            return;
        }

        // Otherwise, force a caret at the end of this editor so toolbar actions
        // always apply to the correct editor instance.
        const range = document.createRange();
        range.selectNodeContents(editor);
        range.collapse(false);
        savedRange = range;
        sel.removeAllRanges();
        sel.addRange(range);
        editor.focus();
    }

    function restoreSelection() {
        ensureSelectionInEditor();
    }

    function execCommandInEditor(command, value = null) {
        restoreSelection();
        return document.execCommand(command, false, value);
    }

    // Keep focus/selection in the editor when using toolbar buttons
    toolbar.addEventListener('mousedown', (e) => {
        if (e.target.closest('input[type="color"], input[type="file"], select')) return;
        if (e.target.closest('button, label.foreColorLabel, label.backColorLabel')) {
            saveSelection();
            e.preventDefault();
        }
    });

    toolbar.querySelector('.foreColor').addEventListener('input', function () {
        restoreSelection();
        document.execCommand('foreColor', false, this.value);
    });

    toolbar.querySelector('.backColor').addEventListener('input', function () {
        restoreSelection();
        document.execCommand('hiliteColor', false, this.value);
    });

    // Keep selection saved as user moves caret in editor (toolbar clicks steal focus)
    editor.addEventListener('keyup', saveSelection);
    editor.addEventListener('mouseup', saveSelection);
    editor.addEventListener('touchend', saveSelection);

    toolbar.querySelector(".ulBtn").addEventListener("click", function () {
        execCommandInEditor("insertUnorderedList");
    });

    let activeOrderedListType = '1';

    function closestElement(node, tagName) {
        if (!node) return null;
        const upper = tagName.toUpperCase();
        let cur = node.nodeType === Node.ELEMENT_NODE ? node : node.parentElement;
        while (cur) {
            if (cur.tagName === upper) return cur;
            cur = cur.parentElement;
        }
        return null;
    }

    function getActiveOrderedList() {
        const sel = window.getSelection();
        if (!sel || sel.rangeCount === 0) return null;
        const anchor = sel.anchorNode;
        if (!anchor || !editor.contains(anchor)) return null;
        return closestElement(anchor, 'ol');
    }

    const ORDERED_LIST_STYLES = {
        '1': null,
        'a': 'lower-alpha',
        'A': 'upper-alpha',
        'i': 'lower-roman',
        'I': 'upper-roman',
    };

    function setTypeOnOl(ol, type) {
        if (!ol) return;
        const listStyleType = ORDERED_LIST_STYLES[type];
        if (!listStyleType) {
            ol.removeAttribute('type');
            ol.removeAttribute('data-ol-style');
            ol.style.removeProperty('list-style-type');
            return;
        }
        // Browsers normalize <ol type="a"> to type="A" in the DOM, so use data-ol-style + inline style.
        ol.setAttribute('data-ol-style', type);
        ol.style.listStyleType = listStyleType;
        ol.setAttribute('type', type);
    }

    function applyOrderedListType(type, { ensureList = false } = {}) {
        if (!type) return;
        activeOrderedListType = type;

        let ol = getActiveOrderedList();
        if (!ol && ensureList) {
            execCommandInEditor("insertOrderedList");
            ol = getActiveOrderedList();
        }
        if (!ol) return;

        setTypeOnOl(ol, type);
    }

    const olBtn = toolbar.querySelector(".olBtn");
    olBtn.addEventListener("click", function () {
        restoreSelection();
        const wasInOl = !!getActiveOrderedList();
        document.execCommand("insertOrderedList", false, null);

        // If toggled off ("unlist"), don't recreate the list by applying subtype.
        const nowInOl = !!getActiveOrderedList();
        if (!wasInOl && nowInOl && activeOrderedListType && activeOrderedListType !== '1') {
            applyOrderedListType(activeOrderedListType, { ensureList: false });
        } else if (wasInOl && nowInOl && activeOrderedListType && activeOrderedListType !== '1') {
            applyOrderedListType(activeOrderedListType, { ensureList: false });
        }
    });

    const olTypeDropdownBtn = toolbar.querySelector('.olTypeDropdownBtn');
    const olTypeDropdownMenu = toolbar.querySelector('.olTypeDropdownMenu');

    if (olTypeDropdownBtn && olTypeDropdownMenu) {
        // Save selection before focus moves to toolbar
        olTypeDropdownBtn.addEventListener('mousedown', saveSelection);

        olTypeDropdownBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            olTypeDropdownMenu.classList.toggle('hidden');
        });

        olTypeDropdownMenu.querySelectorAll('button[data-ol-type]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                restoreSelection();
                applyOrderedListType(btn.dataset.olType, { ensureList: true });
                olTypeDropdownMenu.classList.add('hidden');
            });
        });
    }

    const dropdownBtn = toolbar.querySelector('.paragraphDropdownBtn');
    const dropdownMenu = toolbar.querySelector('.paragraphDropdownMenu');
    const mathTemplateBtn = toolbar.querySelector('.mathTemplateBtn');
    const mathTemplateDropdown = toolbar.querySelector('.mathTemplateDropdown');

    dropdownBtn.addEventListener('click', (e) => {
        e.stopPropagation(); // Prevent bubbling up
        dropdownMenu.classList.toggle('hidden');
    });

    dropdownMenu.querySelectorAll('button[data-cmd]').forEach(btn => {
        btn.addEventListener('click', () => {
            execCommandInEditor(btn.dataset.cmd);
            dropdownMenu.classList.add('hidden');
        });
    });

    function closeToolbarMenus() {
        dropdownMenu.classList.add('hidden');
        if (olTypeDropdownMenu) olTypeDropdownMenu.classList.add('hidden');
        if (mathTemplateDropdown) mathTemplateDropdown.classList.add('hidden');
    }

    // Close dropdowns when clicking outside this editor within the page form/content area
    const menuCloseScope = container.closest('form') || container.closest('#content') || container;
    menuCloseScope.addEventListener('click', (e) => {
        if (!container.contains(e.target)) {
            closeToolbarMenus();
            return;
        }
        if (!toolbar.contains(e.target)) {
            closeToolbarMenus();
            return;
        }
        if (olTypeDropdownMenu && !olTypeDropdownMenu.contains(e.target) && !olTypeDropdownBtn?.contains(e.target)) {
            olTypeDropdownMenu.classList.add('hidden');
        }
        if (mathTemplateDropdown && !mathTemplateDropdown.contains(e.target) && !mathTemplateBtn?.contains(e.target)) {
            mathTemplateDropdown.classList.add('hidden');
        }
        if (!dropdownMenu.contains(e.target) && !dropdownBtn.contains(e.target)) {
            dropdownMenu.classList.add('hidden');
        }
    });

    toolbar.querySelector('.lineHeightSelector').addEventListener('change', function () {
        const value = this.value;
        const selection = window.getSelection();
        if (!selection.rangeCount) return;

        const range = selection.getRangeAt(0);
        const ancestor = range.commonAncestorContainer;

        const walker = document.createTreeWalker(
            ancestor,
            NodeFilter.SHOW_ELEMENT,
            {
                acceptNode: (node) => {
                    if (!range.intersectsNode(node)) return NodeFilter.FILTER_SKIP;
                    const display = window.getComputedStyle(node).display;
                    return display === 'block' ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_SKIP;
                },
            }
        );

        const toStyle = new Set();
        while (walker.nextNode()) {
            toStyle.add(walker.currentNode);
        }

        // Fallback for single-line selection: add block parent manually if none found
        if (toStyle.size === 0) {
            let blockParent = range.startContainer;
            while (blockParent && blockParent.nodeType === Node.TEXT_NODE) {
                blockParent = blockParent.parentNode;
            }
            if (blockParent && window.getComputedStyle(blockParent).display === 'block') {
                toStyle.add(blockParent);
            }
        }

        toStyle.forEach((el) => {
            el.style.lineHeight = value;
        });
    });

    const gridPopup = toolbar.querySelector(".tableGridPopup");
    const gridContainer = gridPopup.querySelector(".tableGrid");
    const label = gridPopup.querySelector(".tableGridLabel");
    const insertTableBtn = toolbar.querySelector(".insertTableBtn");

    // Create 10x10 grid cells
    for (let i = 1; i <= 100; i++) {
        const cell = document.createElement("div");
        cell.className = "w-4 h-4 bg-bg-card-alt hover:bg-accent";
        cell.dataset.row = Math.ceil(i / 10);
        cell.dataset.col = i % 10 === 0 ? 10 : i % 10;

        gridContainer.appendChild(cell);
    }

    let selectedRows = 0;
    let selectedCols = 0;

    // Hover feedback
    gridContainer.addEventListener("mouseover", (e) => {
        if (e.target.dataset.row) {
            selectedRows = parseInt(e.target.dataset.row);
            selectedCols = parseInt(e.target.dataset.col);

            label.textContent = `${selectedRows} × ${selectedCols}`;

            // Highlight cells
            [...gridContainer.children].forEach(cell => {
                const r = parseInt(cell.dataset.row);
                const c = parseInt(cell.dataset.col);
                cell.style.backgroundColor = (r <= selectedRows && c <= selectedCols) ? '#ad2f2f' : '#f3ece5';
            });
        }
    });

    // Insert table on click
    gridContainer.addEventListener("click", () => {
        restoreSelection(); // restore cursor back to editor

        const table = document.createElement("table");
        table.className = "w-full border border-black border-collapse my-2";

        for (let i = 0; i < selectedRows; i++) {
            const tr = document.createElement("tr");
            for (let j = 0; j < selectedCols; j++) {
                const td = document.createElement("td");
                td.textContent = " ";
                td.className = "border border-black w-16 h-8 text-center";
                tr.appendChild(td);
            }
            table.appendChild(tr);
        }

        const selection = window.getSelection();
        if (!selection.rangeCount) return;
        const range = selection.getRangeAt(0);
        range.deleteContents();
        range.insertNode(table);

        gridPopup.classList.add("hidden");
    });

    // Toggle popup on button click
    insertTableBtn.addEventListener("click", () => {
        saveSelection(); // save cursor before opening popup
        gridPopup.classList.toggle("hidden");
    });

    const linkBtn = toolbar.querySelector(".linkBtn");

    linkBtn.addEventListener("click", () => {
        const url = prompt("Enter the URL:");
        if (!url) return;

        const selection = window.getSelection();
        if (!selection.rangeCount) return;

        const range = selection.getRangeAt(0);

        const link = document.createElement("a");
        link.href = url;
        link.target = "_blank";
        link.rel = "noopener noreferrer";
        link.style.color = "#1a0dab"; // style like a link
        link.style.textDecoration = "underline"; // underline

        // If text is selected, wrap it
        if (!range.collapsed) {
            link.textContent = range.toString();
            range.deleteContents();
            range.insertNode(link);
        } else {
            // No text selected, insert placeholder
            link.textContent = url;
            range.insertNode(link);
        }
    });

    toolbar.querySelector('.hrBtn').addEventListener('click', function () {
        document.execCommand('insertHorizontalRule', false, null);
    });

    const imageUpload = toolbar.querySelector('.imageUpload');
    toolbar.querySelector('.imageBtn').addEventListener('click', () => {
        imageUpload.click();
    });

    imageUpload.addEventListener('change', (event) => {
        insertImage(event, editor);
    });

    if (typeof initImageEditing === 'function') {
        initImageEditing(editor, editorWrapper);
    }

    const fullscreenBtn = toolbar.querySelector(".fullscreenBtn");
    let isFullscreen = false;
    let isPreviewVisible = true;

    fullscreenBtn.addEventListener("click", () => {
        isFullscreen = !isFullscreen;
        editorWrapper.classList.toggle("editor-fullscreen", isFullscreen);
        editor.classList.toggle('h-52', !isFullscreen);

        if (isFullscreen) {
            // hide preview (without this vertical scrollbar is not working properly under editor, 
            // as it shows scrollbar in outer container including toolbar)
            setPreviewVisibility(false);
        } else {
            // reset preview visibility to its last state
            setPreviewVisibility(isPreviewVisible);
        }

        // Swap icon between expand and compress
        fullscreenBtn.innerHTML = isFullscreen
            ? '<i class="fas fa-compress-arrows-alt"></i>'
            : '<i class="fas fa-expand-arrows-alt"></i>';
    });

    const previewBtn = toolbar.querySelector(".previewToggleBtn");

    previewBtn.addEventListener("click", () => {
        isPreviewVisible = !isPreviewVisible;
        // don't make preview visible when it is full screen
        setPreviewVisibility(isPreviewVisible && !isFullscreen);
    });

    function setPreviewVisibility(visible) {
        output.classList.toggle("hidden", !visible);
        editorWrapper.classList.toggle("w-full", !visible);
        editorWrapper.classList.toggle("w-1/2", visible);
        if (visible) requestAnimationFrame(syncPreviewSize);
    }

    const codeViewBtn = toolbar.querySelector(".codeViewBtn");

    codeViewBtn.addEventListener("click", () => {
        toggleCodeView();
    });

    const codeView = editorWrapper.querySelector(".codeView");

    function activeEditorSurface() {
        return codeView.classList.contains("hidden") ? editor : codeView;
    }

    function applyResizableBounds() {
        const previewVisible = isPreviewVisible && !output.classList.contains("hidden");
        const gap = parseFloat(getComputedStyle(container).columnGap || getComputedStyle(container).gap || '0') || 0;
        const maxWidth = Math.floor((container.clientWidth - (previewVisible ? gap : 0)) / (previewVisible ? 2 : 1));

        [editor, codeView, editorWrapper, output].forEach((el) => {
            el.style.maxWidth = maxWidth + 'px';
        });
    }

    function syncPreviewSize() {
        applyResizableBounds();
        if (!isPreviewVisible || output.classList.contains("hidden")) return;

        const surface = activeEditorSurface();
        const rect = surface.getBoundingClientRect();
        if (!rect.width || !rect.height) return;

        output.style.height = rect.height + 'px';
        if (surface.style.width) {
            editorWrapper.style.width = rect.width + 'px';
            output.style.width = rect.width + 'px';
            output.style.flex = '0 0 ' + rect.width + 'px';
        }
    }

    const resizeObserver = new ResizeObserver(syncPreviewSize);
    resizeObserver.observe(editor);
    resizeObserver.observe(codeView);
    window.addEventListener('resize', syncPreviewSize);
    syncPreviewSize();

    function toggleCodeView() {
        if (codeView.classList.contains("hidden")) {
            codeView.textContent = formatHTML(getEditorHtml(editor)); // Show formatted HTML
            codeView.classList.remove("hidden");
            editor.classList.add("hidden");
        } else {
            editor.innerHTML = codeView.textContent; // Restore from code
            codeView.classList.add("hidden");
            editor.classList.remove("hidden");
        }
        syncPreviewSize();
    }

    function formatHTML(html) {
        const tab = '  ';
        let result = '';

        // Use browser to parse and normalize input HTML
        const div = document.createElement('div');
        div.innerHTML = html.trim();

        function format(node, level = 0) {
            const indent = tab.repeat(level);

            if (node.nodeType === 3) { // text node
                const trimmed = node.textContent.trim();
                if (trimmed) result += indent + trimmed + '\n';
                return;
            }

            if (node.nodeType !== 1) return; // skip comments/others

            const attrs = [...node.attributes].map(attr => {
                const escapedValue = attr.value.replace(/"/g, '&quot;');
                return ` ${attr.name}="${escapedValue}"`;
            }).join('');
            const tagOpen = `<${node.nodeName.toLowerCase()}${attrs}>`;
            const tagClose = `</${node.nodeName.toLowerCase()}>`;

            result += indent + tagOpen + '\n';

            for (let child of node.childNodes) {
                format(child, level + 1);
            }

            result += indent + tagClose + '\n';
        }

        for (let child of div.childNodes) {
            format(child, 0);
        }

        return result.trim();
    }

    toolbar.querySelector('.mathBtn').addEventListener('click', () => {
        insertMath(editor);
    });

    if (mathTemplateBtn && mathTemplateDropdown) {
        mathTemplateBtn.addEventListener('click', (e) => {
            e.stopPropagation();
            mathTemplateDropdown.classList.toggle('hidden');
        });

        mathTemplateDropdown.querySelectorAll('button[data-math-template]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                e.preventDefault();
                e.stopPropagation();
                restoreSelection();
                insertMathTemplate(editor, btn.dataset.mathTemplate);
                mathTemplateDropdown.classList.add('hidden');
            });
        });
    }

    editor.addEventListener('keydown', (e) => {
        if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === 'h') {
            e.preventDefault();
            const selection = window.getSelection();
            if (!selection.rangeCount) return;
            if (!editor.contains(selection.anchorNode)) return;
            insertInlineMathDelimiters(editor);
        }
    });

    container.dataset.editorInitialized = 'true';
    } catch (err) {
        console.error('Failed to initialize rich text editor', err);
    }
    });
};
