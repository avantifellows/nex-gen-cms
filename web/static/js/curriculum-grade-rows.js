// Helpers shared by every screen that renders the "curriculum_grade_selects" row template
// (Add Test modal, and the Test Setup modal on the edit test screen).

// The selects load their options via htmx, so the saved value is carried in data-selected and
// applied once the options have arrived.
function selectPreloadedOption(selectEl) {
    const selectedValue = selectEl.dataset.selected;
    if (!selectedValue) return;
    const option = selectEl.querySelector(`option[value='${selectedValue}']`);
    if (option) {
        option.selected = true;
    }
}
