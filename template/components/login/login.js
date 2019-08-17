m.ui.components.login = (function () {
    
    m.ui.router.add(/\/login/, function (uri, data) {
        var el = m.ui.components.import(document.createElement("login"));
        el.template = "login";
        el.render();
        var main = document.querySelector("main");
        main.innerHTML = "";
        main.appendChild(el);
    });

    m.ui.ready(function () {
        if (m.ui.jwt.authenticated() === true) {
            onLogin();
        }
    });
    function onLogin() {
        console.log("LOGIN");
    }
    return function (element, data) {
        element.login = function () {
            m.ui.jwt.authenticate(element.getData("username"), element.getData("password")).then(function () {
                onLogin();
            }).catch(function (err) {
                console.error("LOGIN:", err);
            });
        }
        return m.ui.component(element, data);
    };
}());