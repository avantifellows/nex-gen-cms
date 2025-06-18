(() => {
    // Attach event listeners
    document.getElementById('fontSelector').addEventListener('change', (e) => {
        document.execCommand('styleWithCSS', false, true);
        document.execCommand('fontName', false, e.target.value);
    });

    document.getElementById('boldBtn').addEventListener('click', () => {
        document.execCommand('bold');
    });

    document.getElementById('italicBtn').addEventListener('click', () => {
        document.execCommand('italic');
    });

    document.getElementById('underlineBtn').addEventListener('click', () => {
        document.execCommand('underline');
    });

    document.getElementById('mathBtn').addEventListener('click', insertMath);

    document.getElementById('imageBtn').addEventListener('click', () => {
        document.getElementById('imageUpload').click();
    });

    document.getElementById('imageUpload').addEventListener('change', insertImage);

    document.getElementById('editor').addEventListener('input', () => {
        clearTimeout(window.__renderTimeout);
        window.__renderTimeout = setTimeout(renderMath, 300);
    });

    document.getElementById("fontSizeSelector").addEventListener("change", function () {
        const size = this.value;
        document.execCommand("fontSize", false, "7"); // use size 7 as a placeholder
        const editor = document.getElementById("editor");
        const fontElements = editor.getElementsByTagName("font");
        for (let i = 0; i < fontElements.length; i++) {
            if (fontElements[i].size == "7") {
                fontElements[i].removeAttribute("size");
                fontElements[i].style.fontSize = size;
            }
        }
    });

    let savedRange = null;

    function saveSelection() {
        const sel = window.getSelection();
        if (sel.rangeCount > 0) {
            savedRange = sel.getRangeAt(0);
        }
    }

    function restoreSelection() {
        if (savedRange) {
            const sel = window.getSelection();
            sel.removeAllRanges();
            sel.addRange(savedRange);
        }
    }

    document.getElementById('foreColor').addEventListener('input', function () {
        restoreSelection();
        document.execCommand('foreColor', false, this.value);
    });

    document.getElementById('backColor').addEventListener('input', function () {
        restoreSelection();
        document.execCommand('hiliteColor', false, this.value);
    });

    document.getElementById("ulBtn").addEventListener("click", function () {
        document.execCommand("insertUnorderedList", false, null);
    });

    document.getElementById("olBtn").addEventListener("click", function () {
        document.execCommand("insertOrderedList", false, null);
    });

    const dropdownBtn = document.getElementById('paragraphDropdownBtn');
    const dropdownMenu = document.getElementById('paragraphDropdownMenu');

    dropdownBtn.addEventListener('click', (e) => {
        e.stopPropagation(); // Prevent bubbling up
        dropdownMenu.classList.toggle('hidden');
    });

    // Close dropdown when clicking outside
    document.addEventListener('click', (e) => {
        if (!document.getElementById('paragraphDropdownContainer').contains(e.target)) {
            dropdownMenu.classList.add('hidden');
        }
        if (!document.getElementById('insertTableWrapper').contains(e.target)) {
            gridPopup.classList.add('hidden');
        }
    });

    // Handle command execution
    dropdownMenu.querySelectorAll('button[data-cmd]').forEach(btn => {
        btn.addEventListener('click', () => {
            document.execCommand(btn.dataset.cmd);
            dropdownMenu.classList.add('hidden');
        });
    });

    document.getElementById('lineHeightSelector').addEventListener('change', function () {
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

    const gridPopup = document.getElementById("tableGridPopup");
    const gridContainer = document.getElementById("tableGrid");
    const label = document.getElementById("tableGridLabel");
    const insertTableBtn = document.getElementById("insertTableBtn");

    // Create 10x10 grid cells
    for (let i = 1; i <= 100; i++) {
        const cell = document.createElement("div");
        cell.className = "w-4 h-4 bg-gray-200 hover:bg-blue-400";
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
                cell.style.backgroundColor = (r <= selectedRows && c <= selectedCols) ? '#60a5fa' : '#e5e7eb';
            });
        }
    });

    // Insert table on click
    gridContainer.addEventListener("click", () => {
        restoreSelection(); // restore cursor back to editor

        const editor = document.getElementById("editor");

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

    // Close grid if clicking outside
    document.addEventListener("click", (e) => {
    });

    const linkBtn = document.getElementById("linkBtn");

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

    document.getElementById("editor").addEventListener("click", function (e) {
        const target = e.target;
        if (target.tagName === "A") {
            e.preventDefault(); // Prevent default edit behavior
            window.open(target.href, "_blank"); // Open link in new tab
        }
    });

    document.getElementById('hrBtn').addEventListener('click', function () {
        document.execCommand('insertHorizontalRule', false, null);
    });

    const fullscreenBtn = document.getElementById("fullscreenBtn");
    const editorWrapper = document.getElementById("editor-wrapper");
    let isFullscreen = false;

    fullscreenBtn.addEventListener("click", () => {
        isFullscreen = !isFullscreen;
        editorWrapper.classList.toggle("editor-fullscreen", isFullscreen);

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

    // let isCodeView = false;
    const codeViewBtn = document.getElementById("codeViewBtn");
    const editor = document.getElementById("editor");

    codeViewBtn.addEventListener("click", () => {
        toggleCodeView();
    });

    const codeView = document.getElementById("codeView");

    function toggleCodeView() {
        if (codeView.classList.contains("hidden")) {
            codeView.textContent = formatHTML(editor.innerHTML); // Show formatted HTML
            codeView.classList.remove("hidden");
            editor.classList.add("hidden");
        } else {
            editor.innerHTML = codeView.textContent; // Restore from code
            codeView.classList.add("hidden");
            editor.classList.remove("hidden");
        }
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

    const previewBtn = document.getElementById("previewToggleBtn");
    const output = document.getElementById("output");
    let isPreviewVisible = true;

    previewBtn.addEventListener("click", () => {
        isPreviewVisible = !isPreviewVisible;
        // don't make preview visible when it is full screen
        setPreviewVisibility(isPreviewVisible && !isFullscreen);
    });

    function setPreviewVisibility(visible) {
        if (visible) {
            output.classList.remove("hidden");
            editorWrapper.classList.remove("w-full");
            editorWrapper.classList.add("w-1/2");
        } else {
            output.classList.add("hidden");
            editorWrapper.classList.remove("w-1/2");
            editorWrapper.classList.add("w-full");
        }
    }
})();