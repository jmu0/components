m.ui.components.example = function (element, data) {
    element = m.ui.component(element, data);
    element.template = "example";
    element.save = function () {
        console.log("SAVING:", element.getData());
    };

    element.exampleAction = function () {
        console.log("example action");
    };
    element.anotherAction = function () {
        console.log("another action");
    };

    m.ui.router.add(/\/example\/(.*)/, function (uri, data) {
        console.log("ROUTE from example", uri, data);
    });

    return element;
};