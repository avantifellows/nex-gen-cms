// Insert an editable inline math field at the cursor
function insertMath(editor) {
    const mathfield = document.createElement('math-field');

    mathfield.setAttribute('style', 'display:inline-block; min-width:9em; border: 1px solid #d1d5db;');
    mathfield.setAttribute('virtual-keyboard-target', '#math-keyboard');

    // Insert at caret position
    const range = window.getSelection().getRangeAt(0);
    range.insertNode(mathfield);
    mathfield.focus();

    mathfield.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            const latex = mathfield.getValue('latex')?.trim() || '';
            const span = document.createElement('span');
            span.textContent = `\\(${latex}\\)`;

            // Replace the mathfield with the LaTeX span directly in #editor
            mathfield.replaceWith(span);

            // Place caret after the span
            const range = document.createRange();
            const selection = window.getSelection();
            range.setStartAfter(span);
            range.setEndAfter(span);
            selection.removeAllRanges();
            selection.addRange(range);

            // Update preview
            renderMath(editor);
        }
    });
}

function renderMath(editor) {
    const container = editor.closest('.container');
    const output = container.querySelector('.output');
    output.innerHTML = editor.innerHTML;
    MathJax.typesetPromise([output]);
}
