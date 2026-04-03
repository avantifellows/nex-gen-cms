(function () {
    htmx.defineExtension('infinite-scroll', {
        onEvent: function (name, evt) {

            const allowedEvents = [
                "htmx:beforeRequest",
                "htmx:beforeSwap", 
                "htmx:afterSwap",
                "htmx:responseError"
            ];
    
            if (!allowedEvents.includes(name)) return;
    
            const requestPath = evt.detail?.elt?.getAttribute("hx-get");
            if (!isSearchEndpoint(requestPath)) return;

            const target = evt.detail?.target || evt.detail?.elt;

            // Configurable attributes (with defaults)
            const loaderSelector = target.getAttribute("data-loader");
            const sentinelSelector = target.getAttribute("data-sentinel");
            
            const loader = document.querySelector(loaderSelector);

            if (name === 'htmx:beforeRequest') {
                const requestUrl = evt.detail.pathInfo?.finalRequestPath;
                const queryString = requestUrl.split("?")[1] || "";
                target.setAttribute("data-active-params", queryString);

                // this event is fired only from hx-get calls and not from manual js htmx.ajax() calls,
                // hence only initially / on changing search term / on changing sort order, so reset offset and data
                target.setAttribute("data-offset", "0");
                target.innerHTML = "";

                if (loader) loader.classList.remove("hidden");

            } else if (name === 'htmx:beforeSwap') {
                const xhr = evt.detail.xhr;
                const responseUrl = xhr.responseURL;
                const responseParams = responseUrl.split("?")[1] || "";
                const responseSearch = getSearchValue(responseParams);
                const responseSubject = getSubjectValue(responseParams);

                const activeParams = target.getAttribute("data-active-params");
                const activeSearch = getSearchValue(activeParams);
                const activeSubject = getSubjectValue(activeParams);

                if (responseSearch !== activeSearch || responseSubject !== activeSubject) {
                    // search term has changed so ignore response
                    evt.detail.shouldSwap = false;   // cancel swap
                    return;
                }

            } else if (name === 'htmx:responseError') {
                if (loader) loader.classList.add("hidden");

                target.removeAttribute("data-loading");

                let status = evt.detail.xhr?.status || "Network";
                let message = evt.detail.xhr?.statusText || "Request failed";

                alert(`Error loading data: ${status} ${message}`);

            } else if (name === 'htmx:afterSwap') {
                if (loader) loader.classList.add("hidden");
                const limit = parseInt(target.getAttribute("data-limit") || "10", 10);
                const offset = parseInt(target.getAttribute("data-offset") || "0", 10);
                target.setAttribute("data-offset", offset + limit);
                target.removeAttribute("data-loading");

                const hasMore = evt.detail.xhr.getResponseHeader("hasMore");
                // Check server flag if present
                if (hasMore === "false") {
                    // Server indicates no more results, unobserving sentinel.
                    if (target.__observer && target.__sentinel) {
                        target.__observer.unobserve(target.__sentinel);
                    }
                    return;
                }

                // Create observer once to load more items on scroll
                if (!target.__sentinel) {
                    const sentinel = document.querySelector(sentinelSelector);
                    if (!sentinel) return;

                    target.__sentinel = sentinel;

                    const observer = new IntersectionObserver((entries) => {
                        entries.forEach(entry => {
                            if (entry.isIntersecting && target.getAttribute("data-loading") !== "true") {
                                target.setAttribute("data-loading", "true");
                                if (loader) loader.classList.remove("hidden");

                                let limit = target.getAttribute("data-limit") || "10";
                                let offset = target.getAttribute("data-offset") || "0";

                                let baseUrl = target.getAttribute("hx-get");
                                let params = new URLSearchParams(target.getAttribute("data-active-params") || "");
                                params.set("limit", limit);
                                params.set("offset", offset);

                                const [basePath] = baseUrl.split("?");
                                const finalUrl = `${basePath}?${params.toString()}`;

                                htmx.ajax("GET", finalUrl, {
                                    target: target,
                                    swap: "beforeend"
                                });
                            }
                        });

                    }, {
                        root: null,
                        rootMargin: "100px",
                        threshold: 0
                    });

                    observer.observe(sentinel);
                    target.__observer = observer;

                    // Cleanup observer if element removed
                    const cleanupObserver = new MutationObserver(() => {
                        if (!document.contains(target)) {
                            observer.disconnect();
                            cleanupObserver.disconnect();
                        }
                    });
                    cleanupObserver.observe(document.body, {
                        childList: true,
                        subtree: true
                    });

                } else {
                    target.__observer.unobserve(target.__sentinel);
                    target.__observer.observe(target.__sentinel);
                }
            }
        }
    });

    function isSearchEndpoint(path) {
        return path?.startsWith("/api/search");
    }

    function getSearchValue(query) {
        return getKeyValue(query, "search");
    }

    function getSubjectValue(query) {
        return getKeyValue(query, "subject");
    }

    function getKeyValue(query, keyPart) {
        const params = new URLSearchParams(query);
    
        // Find the param that contains search text
        for (const [key, value] of params.entries()) {
            if (key.includes(keyPart)) {
                return value || "";
            }
        }
        return "";
    }
})();