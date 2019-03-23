m.ui.components.login = (function(){
    m.ui.ready(function(){
        if (m.ui.jwt.authenticated()===true) {
            onLogin();
        }
    });
    function onLogin(){
        console.log("LOGIN");
    }
    return function(element, data){
        element.login = function() {
            m.ui.jwt.authenticate(element.getData("username"), element.getData("password")).then(function(){
                onLogin();
            }).catch(function(err){
                console.error("LOGIN:",err);
            });
        }
        return m.ui.component(element,data);
    };
}());