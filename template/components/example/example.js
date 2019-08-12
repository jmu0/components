m.ui.components.example = (function(){
    m.ui.router.add(/\/example\/(.*)/, function (uri, data) {
        console.log("ROUTE from example/*", uri, data);
        var el = m.ui.components.import(document.createElement("example"));
        el.template="example";
        el.render();
        var main =  document.querySelector("main");
        main.innerHTML="";
        main.appendChild(el);
    });
    
    return function (element, data) {
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
    
        
        return element;
    };
}());

