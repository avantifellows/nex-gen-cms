/**
 * Client-side type → subtype options for resource add/edit (no server-side validation).
 */
window.ResourceForm = {
    subtypesByType: {
        document: ["Module", "Previous Year Questions"],
        quiz: ["Assessment"],
        video: ["Video Lectures"],
    },

    /**
     * @param {HTMLSelectElement} subtypeEl
     * @param {string} typeValue raw type value (e.g. "document")
     * @param {string} [selectedSubtype] pre-selected label, or "" to only show placeholder
     */
    fillSubtypeSelect: function (subtypeEl, typeValue, selectedSubtype) {
        var t = (typeValue || "").toLowerCase();
        var subs = this.subtypesByType[t] || [];
        selectedSubtype = selectedSubtype === undefined || selectedSubtype === null ? "" : String(selectedSubtype);

        subtypeEl.innerHTML = "";
        var placeholder = document.createElement("option");
        placeholder.value = "";
        placeholder.disabled = true;
        placeholder.hidden = true;
        placeholder.textContent = subs.length ? "Select Subtype" : "Select type first";

        var matched = selectedSubtype && subs.indexOf(selectedSubtype) !== -1;
        placeholder.selected = !matched;
        subtypeEl.appendChild(placeholder);

        subs.forEach(function (label) {
            var opt = document.createElement("option");
            opt.value = label;
            opt.textContent = label;
            if (matched && selectedSubtype === label) {
                opt.selected = true;
                placeholder.selected = false;
            }
            subtypeEl.appendChild(opt);
        });

        subtypeEl.disabled = subs.length === 0;
    },

    /**
     * @param {string} typeId element id of type select
     * @param {string} subtypeId element id of subtype select
     * @param {string} [selectedSubtype] omit or pass "" when type changes (clear subtype)
     */
    sync: function (typeId, subtypeId, selectedSubtype) {
        var typeEl = document.getElementById(typeId);
        var subEl = document.getElementById(subtypeId);
        if (!typeEl || !subEl) return;
        if (selectedSubtype === undefined) {
            selectedSubtype = "";
        }
        this.fillSubtypeSelect(subEl, typeEl.value, selectedSubtype);
    },
};
