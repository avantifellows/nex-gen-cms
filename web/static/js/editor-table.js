// The "editor" template partial (and this script tag with it) is included once per rich-text
// field on a page, so this file executes multiple times in the same global scope - var (not
// const) so redeclaration doesn't throw, matching IMAGE_CARET_MARKER in editor-image.js.
var TABLE_RESIZE_MARGIN = 6; // px on either side of an interior column border that triggers resize
var TABLE_MIN_COL_PERCENT = 5; // a column can't be dragged narrower than this % of the table's width

function createEditorTable(rows, cols) {
    const table = document.createElement('table');
    table.className = 'w-full border border-black border-collapse my-2';
    table.style.tableLayout = 'fixed';

    const colgroup = document.createElement('colgroup');
    const pct = (100 / Math.max(cols, 1)).toFixed(4);
    for (let j = 0; j < cols; j++) {
        const col = document.createElement('col');
        col.style.width = pct + '%';
        colgroup.appendChild(col);
    }
    table.appendChild(colgroup);

    for (let i = 0; i < rows; i++) {
        const tr = document.createElement('tr');
        for (let j = 0; j < cols; j++) {
            const td = document.createElement('td');
            td.className = 'border border-black h-8';
            tr.appendChild(td);
        }
        table.appendChild(tr);
    }
    return table;
}
window.createEditorTable = createEditorTable;

/**
 * Tables saved before this feature existed (or pasted in from elsewhere) have no <colgroup> -
 * their columns are sized however the browser's table auto-layout happens to lay them out.
 * Give them one on first encounter, sized from their current rendered widths (normalized to
 * 100%) so the table doesn't visibly jump the moment a user hovers it.
 */
function ensureTableColgroup(table) {
    if (table.querySelector(':scope > colgroup')) return;

    const rows = Array.from(table.rows);
    const numCols = rows.reduce((max, tr) => Math.max(max, tr.cells.length), 0);
    if (numCols === 0) return;

    const tableWidth = table.getBoundingClientRect().width || 1;
    const firstRow = rows[0];

    const colgroup = document.createElement('colgroup');
    for (let i = 0; i < numCols; i++) {
        const cell = firstRow?.cells[i];
        const cellWidth = cell ? cell.getBoundingClientRect().width : tableWidth / numCols;
        const col = document.createElement('col');
        col.style.width = ((cellWidth / tableWidth) * 100) + '%';
        colgroup.appendChild(col);
    }

    // Renormalize to exactly 100% - measurement/rounding drift here would otherwise compound
    // across future resizes, since each drag only touches two adjacent columns' share of the total.
    const colEls = Array.from(colgroup.children);
    const total = colEls.reduce((sum, col) => sum + parseFloat(col.style.width), 0) || 100;
    colEls.forEach(col => {
        col.style.width = ((parseFloat(col.style.width) / total) * 100).toFixed(4) + '%';
    });

    table.insertBefore(colgroup, table.firstChild);
    table.style.tableLayout = 'fixed';
    table.classList.add('w-full');
}

/** Interior column borders only - the outer edges have no neighboring column to trade width with. */
function getColumnBoundaries(table) {
    ensureTableColgroup(table);
    const firstRow = table.rows[0];
    const cols = table.querySelectorAll(':scope > colgroup > col');
    const tableRect = table.getBoundingClientRect();
    if (!firstRow || cols.length < 2) return { boundaries: [], tableRect };

    const boundaries = [];
    for (let i = 0; i < cols.length - 1; i++) {
        const cell = firstRow.cells[i];
        if (!cell) continue;
        boundaries.push({ leftCol: i, rightCol: i + 1, x: cell.getBoundingClientRect().right });
    }
    return { boundaries, tableRect };
}

function findTableColumnBorder(editor, clientX, clientY) {
    for (const table of editor.querySelectorAll('table')) {
        const { boundaries, tableRect } = getColumnBoundaries(table);
        if (clientY < tableRect.top || clientY > tableRect.bottom) continue;

        for (const b of boundaries) {
            if (Math.abs(clientX - b.x) <= TABLE_RESIZE_MARGIN) {
                return { table, leftCol: b.leftCol, rightCol: b.rightCol, edgeX: b.x, tableRect };
            }
        }
    }
    return null;
}

function initTableColumnResize(editor, editorWrapper) {
    const resizeLine = document.createElement('div');
    resizeLine.className = 'table-col-resize-line';
    editorWrapper.appendChild(resizeLine);

    const pctLabelLeft = document.createElement('div');
    pctLabelLeft.className = 'table-col-resize-pct table-col-resize-pct-left';
    editorWrapper.appendChild(pctLabelLeft);

    const pctLabelRight = document.createElement('div');
    pctLabelRight.className = 'table-col-resize-pct table-col-resize-pct-right';
    editorWrapper.appendChild(pctLabelRight);

    let dragging = false;

    // The line lives in editorWrapper, a sibling of the scrollable .editor box, so it isn't
    // clipped by .editor's own overflow - without this, it would visibly run past the editor's
    // visible top/bottom edge (over the toolbar, or beyond its bottom) whenever a table taller
    // than the editor's viewport is partly scrolled out of view.
    function visibleLineRect(tableRect) {
        const editorRect = editor.getBoundingClientRect();
        const top = Math.max(tableRect.top, editorRect.top);
        const bottom = Math.min(tableRect.bottom, editorRect.bottom);
        return { top, height: bottom - top };
    }

    function showLine(x, tableRect) {
        const { top, height } = visibleLineRect(tableRect);
        if (height <= 0) {
            hideLine();
            return null;
        }

        const wRect = editorWrapper.getBoundingClientRect();
        resizeLine.style.left = (x - wRect.left) + 'px';
        resizeLine.style.top = (top - wRect.top) + 'px';
        resizeLine.style.height = height + 'px';
        resizeLine.classList.add('active');
        return { top, height };
    }

    function hideLine() {
        resizeLine.classList.remove('active');
    }

    function showPercentLabels(x, pctA, pctB, visibleTop, visibleHeight) {
        const wRect = editorWrapper.getBoundingClientRect();
        const lineX = x - wRect.left;
        const midY = visibleTop + visibleHeight / 2 - wRect.top;

        pctLabelLeft.textContent = Math.round(pctA) + '%';
        pctLabelLeft.style.left = lineX + 'px';
        pctLabelLeft.style.top = midY + 'px';
        pctLabelLeft.classList.add('active');

        pctLabelRight.textContent = Math.round(pctB) + '%';
        pctLabelRight.style.left = lineX + 'px';
        pctLabelRight.style.top = midY + 'px';
        pctLabelRight.classList.add('active');
    }

    function hidePercentLabels() {
        pctLabelLeft.classList.remove('active');
        pctLabelRight.classList.remove('active');
    }

    editor.addEventListener('mousemove', (e) => {
        if (dragging) return;
        const target = findTableColumnBorder(editor, e.clientX, e.clientY);
        if (target) {
            editor.style.cursor = 'col-resize';
            const visible = showLine(target.edgeX, target.tableRect);
            if (visible) {
                const cols = target.table.querySelectorAll(':scope > colgroup > col');
                const pctA = parseFloat(cols[target.leftCol]?.style.width) || 0;
                const pctB = parseFloat(cols[target.rightCol]?.style.width) || 0;
                showPercentLabels(target.edgeX, pctA, pctB, visible.top, visible.height);
            } else {
                hidePercentLabels();
            }
        } else {
            editor.style.cursor = '';
            hideLine();
            hidePercentLabels();
        }
    });

    editor.addEventListener('mouseleave', () => {
        if (dragging) return;
        editor.style.cursor = '';
        hideLine();
        hidePercentLabels();
    });

    editor.addEventListener('mousedown', (e) => {
        const target = findTableColumnBorder(editor, e.clientX, e.clientY);
        if (!target) return;

        const cols = target.table.querySelectorAll(':scope > colgroup > col');
        const colA = cols[target.leftCol];
        const colB = cols[target.rightCol];
        if (!colA || !colB) return;

        e.preventDefault();
        dragging = true;
        document.body.classList.add('table-col-resizing');

        const startX = e.clientX;
        const tableWidth = target.table.getBoundingClientRect().width || 1;
        const startPctA = parseFloat(colA.style.width) || 0;
        const startPctB = parseFloat(colB.style.width) || 0;
        const totalPct = startPctA + startPctB;
        const minPct = Math.min(TABLE_MIN_COL_PERCENT, totalPct / 2);

        const onMouseMove = (ev) => {
            const dx = ev.clientX - startX;
            const dPct = (dx / tableWidth) * 100;
            const newPctA = Math.min(Math.max(startPctA + dPct, minPct), totalPct - minPct);
            const newPctB = totalPct - newPctA;

            colA.style.width = newPctA.toFixed(4) + '%';
            colB.style.width = newPctB.toFixed(4) + '%';

            const tableRect = target.table.getBoundingClientRect();
            const lineX = target.edgeX + (newPctA - startPctA) / 100 * tableWidth;
            const visible = showLine(lineX, tableRect);
            if (visible) {
                showPercentLabels(lineX, newPctA, newPctB, visible.top, visible.height);
            } else {
                hidePercentLabels();
            }
        };

        const onMouseUp = () => {
            window.removeEventListener('mousemove', onMouseMove);
            window.removeEventListener('mouseup', onMouseUp);
            document.body.classList.remove('table-col-resizing');
            dragging = false;
            editor.style.cursor = '';
            hideLine();
            hidePercentLabels();
            if (typeof renderMath === 'function') renderMath(editor);
        };

        window.addEventListener('mousemove', onMouseMove);
        window.addEventListener('mouseup', onMouseUp);
    });
}
window.initTableColumnResize = initTableColumnResize;
