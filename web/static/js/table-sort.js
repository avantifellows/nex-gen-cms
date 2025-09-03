(function() {
    htmx.defineExtension('table-sort', {
        onEvent: function(name, evt) {
            if (name === "htmx:afterProcessNode") {
                let scope = evt.target.getAttribute("data-sort-scope");
                if (!scope) return;

                const SORT_COLUMN = scope + "SortColumn";
                const SORT_ORDER = scope + "SortOrder";

                restoreState(scope, SORT_COLUMN, SORT_ORDER);

            } else if (name === "htmx:configRequest") {
                // find the closest parent having data-sort-scope attribute
                let scopeElement = evt.target.closest("[data-sort-scope]");
                if (!scopeElement) return;

                let scope = scopeElement.getAttribute("data-sort-scope");
                if (!scope) return;

                const SORT_COLUMN = scope + "SortColumn";
                const SORT_ORDER = scope + "SortOrder";

                if (evt.detail.path.startsWith("/api/" + scope)) {
                    // Get sessionStorage values
                    let sortColumn = sessionStorage.getItem(SORT_COLUMN);
                    let sortOrder = sessionStorage.getItem(SORT_ORDER);
                    if (sortColumn) {
                        // Add them to the request parameters
                        evt.detail.parameters.sortColumn = sortColumn;
                        evt.detail.parameters.sortOrder = sortOrder;
                    }
                }
            }
        }
    });

    function restoreState(scope, colKey, orderKey) {
        updateSortIcons(
            sessionStorage.getItem(colKey),
            sessionStorage.getItem(orderKey),
            scope
        );
    }

    function updateSortState(column, scope) {
        const colKey = scope + "SortColumn";
        const orderKey = scope + "SortOrder";

        let previousColumn = sessionStorage.getItem(colKey);
        let previousOrder = sessionStorage.getItem(orderKey);

        let newOrder = "asc";
        if (previousColumn === column) {
            newOrder = previousOrder === "asc" ? "desc" : "asc";
        }

        sessionStorage.setItem(colKey, column);
        sessionStorage.setItem(orderKey, newOrder);
    }

    function updateSortIcons(column, order, scope) {
        if (!column || !order) return;

        const table = document.querySelector(`[data-sort-scope="${scope}"]`);
        if (!table) return;

        // Reset icons inside this scope
        table.querySelectorAll("th i.fas").forEach(icon => {
            icon.classList.remove("fa-sort-up", "fa-sort-down");
            icon.classList.add("fa-sort");
        });

        // Highlight active column
        table.querySelectorAll("th").forEach(th => {
            let link = th.querySelector("a");
            if (link && link.getAttribute("hx-on:click")?.includes(column)) {
                let icon = th.querySelector("i.fas");
                if (icon) {
                    icon.classList.remove("fa-sort");
                    icon.classList.add(order === "asc" ? "fa-sort-up" : "fa-sort-down");
                }
            }
        });
    }

    // Expose globally so HTMX can call it via hx-on:click
    window.updateTableSortState = updateSortState;
})();
