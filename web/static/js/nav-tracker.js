var BACK_TO_ADD_TEST_EVT = "back-to-add-test";

var Add_TEST_DIV_SELECTOR = "#add-test-div";

var PUSHED_SCREEN_ADD_TEST = "add-test";

window.lastPushedScreen = null;

if (!window._htmxHistoryRestoreAttached) {
    window._htmxHistoryRestoreAttached = true;

    // listen for screen opened due to back/forward button press
    document.body.addEventListener('htmx:historyRestore', function (evt) {
        // Manually trigger something on a specific div if needed
        const addTestDiv = document.querySelector(Add_TEST_DIV_SELECTOR);
        if (addTestDiv) {
            addTestDiv.dispatchEvent(new CustomEvent(BACK_TO_ADD_TEST_EVT));
        }
    });

    // Monkey-patch pushState to track last screen pushed in stack
    const originalPushState = history.pushState;
    history.pushState = function (state, title, url) {
        // Extract screen identifier based on URL
        if (url.includes("/add-test") || url.includes("/edit-test")) {
            window.lastPushedScreen = PUSHED_SCREEN_ADD_TEST;
        } else {
            window.lastPushedScreen = null;
        }

        return originalPushState.apply(this, arguments);
    };
}