package components

func reloadSocketScript() []byte {
	return []byte(`
    if (m === undefined) var m = {};

    m.socket = (function () {
        var socket;
        function socketOpen(evt) {
            console.log("Reload socket open " + JSON.stringify(evt));
        }
        function socketClose(evt) {
            console.log("Reload socket closed " + JSON.stringify(evt));
        }
        function socketMessage(evt) {
            console.log("Reload socket message: " + evt.data);
            if (evt.data === "reload") {
                window.location.reload();
            }
        }
        function socketError(evt) {
            console.log("Reload socket Error " + JSON.stringify(evt));
        }
        function socketConnect() {
            var url;
            url = "ws://localhost:9876/ws";
            socket = new WebSocket(url);
            socket.onopen = socketOpen;
            socket.onclose = socketClose;
            socket.onmessage = socketMessage;
            socket.onerror = socketError;
            this.socketTestInterval = setInterval(function () {
                if (socket.readyState === 3) {
                    console.error('Reload socket down, reconecting...');
                    socketConnect();
                }
            }, 10000);
        }
        socketConnect();
    }());
`)
}
