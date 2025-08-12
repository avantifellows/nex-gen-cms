var BACK_TO_ADD_TEST_EVT = "back-to-add-test";
var BACK_TO_TOPIC_EVT = "back-to-topic";

var ADD_TEST_DIV_SELECTOR = "#add-test-div";
var TOPIC_DIV_SELECTOR = "#topic-div";

var PUSHED_SCREEN_ADD_TEST = "add-test";
var PUSHED_SCREEN_TOPIC = "topic";

window.lastPushedScreen = null;

if (!window._htmxHistoryRestoreAttached) {
    window._htmxHistoryRestoreAttached = true;

    // listen for screen opened due to back/forward button press
    document.body.addEventListener('htmx:historyRestore', function (evt) {
        // Manually trigger something on a specific div if needed
        const addTestDiv = document.querySelector(ADD_TEST_DIV_SELECTOR);
        if (addTestDiv) {
            addTestDiv.dispatchEvent(new CustomEvent(BACK_TO_ADD_TEST_EVT));
            return;
        }
        const topicDiv = document.querySelector(TOPIC_DIV_SELECTOR);
        if (topicDiv) {
            topicDiv.dispatchEvent(new CustomEvent(BACK_TO_TOPIC_EVT));
            return;
        }
    });

    // Monkey-patch pushState to track last screen pushed in stack
    const originalPushState = history.pushState;
    history.pushState = function (state, title, url) {
        // Extract screen identifier based on URL
        if (url.includes("/add-test") || url.includes("/edit-test")) {
            window.lastPushedScreen = PUSHED_SCREEN_ADD_TEST;
        } else if (url.includes("/topic?id=")) {
            window.lastPushedScreen = PUSHED_SCREEN_TOPIC;
        } else {
            window.lastPushedScreen = null;
        }

        return originalPushState.apply(this, arguments);
    };
}