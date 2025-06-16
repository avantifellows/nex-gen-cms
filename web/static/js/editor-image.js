function insertImage(event) {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = function (e) {
        const img = document.createElement('img');
        img.src = e.target.result;
        img.style.maxWidth = '100%';
        img.style.display = 'block';
        img.style.margin = '0.5em 0';

        const range = window.getSelection().getRangeAt(0);
        range.insertNode(img);
        range.setStartAfter(img);
        range.setEndAfter(img);

        const selection = window.getSelection();
        selection.removeAllRanges();
        selection.addRange(range);

        renderMath(); // Optional if preview shows images too
    };
    reader.readAsDataURL(file);

    // Reset input so same file can be selected again
    event.target.value = '';
}