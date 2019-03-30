m.ui.components.anotherexample = (function(){
    //Do stuff
    var tmpl = "example";

    m.ui.router.add(/anotherexample/, function(){
        var e = m.ui.components.anotherexample(document.createElement("another"), {"testkey":"another example", "namekey":"Jos"});
        e.render();
        document.querySelector("main").appendChild(e);
    });
    
    return function(element, data) {
        element=m.ui.component(element,data);
        element.template=tmpl;
        element.save = function(){
            console.log("Saving:", element.getData());
        }
        return element;
    }
}());