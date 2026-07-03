function whenMathEditorReady(callback) {
    if (window.customElements?.get('math-field')) {
        callback();
        return;
    }
    if (window.customElements?.whenDefined) {
        customElements.whenDefined('math-field').then(callback).catch(() => callback());
        return;
    }
    callback();
}
window.whenMathEditorReady = whenMathEditorReady;
window.whenMathNormalizerReady = whenMathEditorReady;

function mathmlTextContent(node) {
    return (node.textContent || '')
        .replace(/\u00A0/g, ' ')
        .replace(/&nbsp;/gi, ' ');
}

function isMathmlElement(node, name) {
    return node?.nodeType === Node.ELEMENT_NODE && node.localName?.toLowerCase() === name;
}

function mathmlChildren(node) {
    return [...node.childNodes].filter((child) => child.nodeType === Node.ELEMENT_NODE);
}

function serializeMathmlRow(nodes) {
    const parts = [];
    let run = '';

    const flushRun = () => {
        if (!run) return;
        parts.push(run);
        run = '';
    };

    for (const child of nodes) {
        if (isMathmlElement(child, 'mi') || isMathmlElement(child, 'mn')) {
            run += mathmlTextContent(child);
            continue;
        }
        flushRun();
        parts.push(serializeMathmlNode(child));
    }
    flushRun();
    return parts.join('');
}

function serializeMathmlNode(node) {
    if (!node || node.nodeType !== Node.ELEMENT_NODE) return '';

    const tag = node.localName.toLowerCase();
    const children = mathmlChildren(node);

    switch (tag) {
        case 'math':
            return serializeMathmlRow(children);
        case 'mrow':
            return serializeMathmlRow(children);
        case 'mn':
        case 'mi':
            return mathmlTextContent(node);
        case 'mtext': {
            const text = mathmlTextContent(node);
            if (!text) return '';
            if (/^\s+$/.test(text)) return text;
            return `\\text{${text}}`;
        }
        case 'mo':
            return mathmlTextContent(node);
        case 'msub': {
            const base = serializeMathmlNode(children[0]);
            const sub = serializeMathmlNode(children[1]);
            return `${base}_{${sub}}`;
        }
        case 'msup': {
            const base = serializeMathmlNode(children[0]);
            const sup = serializeMathmlNode(children[1]);
            return `${base}^{${sup}}`;
        }
        case 'mfrac': {
            const num = serializeMathmlNode(children[0]);
            const den = serializeMathmlNode(children[1]);
            return `\\frac{${num}}{${den}}`;
        }
        case 'mstyle': {
            // MathJax uses a leading mspace to indent continuation lines under a bullet.
            const mspace = children.find((child) => isMathmlElement(child, 'mspace'));
            if (mspace && children.length === 1) {
                return serializeMathmlNode(mspace);
            }
            return serializeMathmlRow(children);
        }
        case 'mspace': {
            const width = node.getAttribute('width') || '1em';
            return `\\hspace{${width}}`;
        }
        default:
            return serializeMathmlRow(children);
    }
}

function mathJaxAssistiveMathToLatex(mathEl) {
    if (!mathEl) return '';
    return serializeMathmlNode(mathEl).replace(/\s{2,}/g, ' ').trim();
}

function createMathField() {
    const mathfield = document.createElement('math-field');
    mathfield.setAttribute(
        'style',
        'display:inline-block; min-width:9em; border: 1px solid rgba(38, 20, 16, 0.15);'
    );
    mathfield.setAttribute('virtual-keyboard-target', '#math-keyboard');
    return mathfield;
}

function latexSpanFromText(latex) {
    const span = document.createElement('span');
    span.textContent = `\\(${latex}\\)`;
    return span;
}

const mathTemplates = {
    piecewise: 'f(x)=\\begin{cases}#? & #? \\\\ #? & #?\\end{cases}',
};

function getLatexFromMathJaxContainer(container) {
    const dataTex = container.getAttribute('data-tex');
    if (dataTex) return dataTex;

    const assistiveMath = container.querySelector('mjx-assistive-mml math');
    if (assistiveMath) {
        const latex = mathJaxAssistiveMathToLatex(assistiveMath);
        if (latex) return latex;
    }

    if (window.MathJax?.startup?.document?.math) {
        for (const math of MathJax.startup.document.math) {
            if (container === math.typesetRoot && math.math) {
                return math.math;
            }
        }
    }

    return null;
}

function wrapLatexDelimiters(latex, container, assistive) {
    const isDisplay = assistive?.getAttribute('display') === 'block' ||
        container.getAttribute('display') === 'true';
    return isDisplay ? `\\[${latex}\\]` : `\\(${latex}\\)`;
}

function serializeMathFields(root) {
    // Toolbar insertMath leaves temporary math-field nodes until Enter is pressed.
    root.querySelectorAll('math-field').forEach((mathfield) => {
        const latex = mathfield.getValue('latex')?.trim() || '';
        mathfield.replaceWith(latex ? latexSpanFromText(latex) : document.createTextNode(''));
    });
}
window.serializeMathFields = serializeMathFields;

// Replace MathJax-rendered nodes with editable \( \) LaTeX text.
function normalizeRenderedMath(root) {
    if (!root) return false;

    const containers = [...root.querySelectorAll('mjx-container')];
    if (!containers.length) return false;

    let converted = 0;
    containers.forEach((container) => {
        const latex = getLatexFromMathJaxContainer(container);
        if (!latex) return;

        const assistive = container.querySelector('mjx-assistive-mml');
        const span = document.createElement('span');
        span.textContent = wrapLatexDelimiters(latex, container, assistive);
        container.replaceWith(span);
        converted += 1;
    });

    return converted > 0;
}
window.normalizeRenderedMath = normalizeRenderedMath;

function normalizeEditorHtmlString(html) {
    if (!html || !html.includes('mjx-container')) return html;

    const temp = document.createElement('div');
    temp.innerHTML = html;
    normalizeRenderedMath(temp);
    return temp.innerHTML;
}
window.normalizeEditorHtmlString = normalizeEditorHtmlString;

// Insert \( \) delimiters at the cursor so LaTeX can be typed directly.
function insertInlineMathDelimiters(editor) {
    const selection = window.getSelection();
    if (!selection.rangeCount) return;

    const range = selection.getRangeAt(0);
    const open = '\\(';
    const close = '\\)';

    if (!range.collapsed) {
        const textNode = document.createTextNode(open + range.toString() + close);
        range.deleteContents();
        range.insertNode(textNode);

        const after = document.createRange();
        after.setStartAfter(textNode);
        after.collapse(true);
        selection.removeAllRanges();
        selection.addRange(after);
    } else {
        const textNode = document.createTextNode(open + close);
        range.insertNode(textNode);

        const inside = document.createRange();
        inside.setStart(textNode, open.length);
        inside.collapse(true);
        selection.removeAllRanges();
        selection.addRange(inside);
    }

    renderMath(editor);
}

function commitMathFieldOnEnter(mathfield, editor) {
    mathfield.addEventListener('keydown', (e) => {
        if (e.key !== 'Enter') return;
        e.preventDefault();

        const latex = mathfield.getValue('latex')?.trim() || '';
        const replacement = latex ? latexSpanFromText(latex) : document.createTextNode('');

        mathfield.replaceWith(replacement);

        const selection = window.getSelection();
        const after = document.createRange();
        after.setStartAfter(replacement);
        after.collapse(true);
        selection.removeAllRanges();
        selection.addRange(after);

        renderMath(editor);
    });
}

function insertMathFieldAtCursor(editor, latex = '') {
    const selection = window.getSelection();
    if (!selection.rangeCount) return;

    const mathfield = createMathField();

    const range = selection.getRangeAt(0);
    range.insertNode(mathfield);
    mathfield.focus();

    if (latex) {
        mathfield.insert(latex, { selectionMode: 'placeholder' });
    }

    commitMathFieldOnEnter(mathfield, editor);
}

// Insert an editable inline math field at the cursor
function insertMath(editor) {
    insertMathFieldAtCursor(editor);
}

function insertMathTemplate(editor, templateName) {
    const latex = mathTemplates[templateName];
    if (!latex) return;
    insertMathFieldAtCursor(editor, latex);
}
window.insertMathTemplate = insertMathTemplate;

function renderMath(editor) {
    const container = editor.closest('.container');
    const output = container.querySelector('.output');
    const clone = editor.cloneNode(true);
    serializeMathFields(clone);
    output.innerHTML = clone.innerHTML;
    if (window.MathJax?.typesetPromise) {
        MathJax.typesetPromise([output]).catch(console.error);
    }
}
