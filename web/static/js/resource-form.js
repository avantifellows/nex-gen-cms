/**
 * Client-side type → subtype options for resource add/edit (no server-side validation).
 * An option is either a plain string (label === stored value) or {value, label}
 * when the stored value differs from what the dropdown shows.
 */
window.ResourceForm = {
    subtypesByType: {
        document: ["Module", "Previous Year Questions", { value: "DPP", label: "Daily Practice Problem (DPP)" }],
        quiz: ["Assessment"],
        quiz_template: ["Quiz Template"],
        video: ["Video Lectures"],
    },

    /**
     * @param {HTMLSelectElement} subtypeEl
     * @param {string} typeValue raw type value (e.g. "document")
     * @param {string} [selectedSubtype] pre-selected stored value, or "" to only show placeholder
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

        var options = subs.map(function (s) {
            return typeof s === "string" ? { value: s, label: s } : s;
        });
        var matched = selectedSubtype && options.some(function (o) { return o.value === selectedSubtype; });
        placeholder.selected = !matched;
        subtypeEl.appendChild(placeholder);

        options.forEach(function (o) {
            var opt = document.createElement("option");
            opt.value = o.value;
            opt.textContent = o.label;
            if (matched && selectedSubtype === o.value) {
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
