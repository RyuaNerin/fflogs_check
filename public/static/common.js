let SPINNER = `
<div class="spinner-border text-primary spinner-border-sm me-3" role="status">
    <span class="visually-hidden">Loading...</span>
</div>`;

function genShowSimpleChange(sourceClass, targetClass) {
    return function() {
        let checked = document.getElementById(sourceClass).checked;
        let elems = document.getElementsByClassName(targetClass);
        
        for (let i = 0; i < elems.length; i++) {
            if (checked) {
                elems[i].classList.remove('d-none');
            } else {
                elems[i].classList.add('d-none');
            }
        }
    }
}

function addHandler(parseRequestData, loadCompleted) {
    document.addEventListener(
        'DOMContentLoaded',
        function () {
            let searchButton = document.getElementById("search");
            let content = document.getElementById("content");

            searchButton.addEventListener(
                'click',
                function(e) {
                    content.innerHTML = "";

                    grecaptcha.ready(
                        function() {
                            grecaptcha.execute(
                                '6LezZkAeAAAAAIP2Yy7drAa1NBCZsMiLFdEJ706F',
                                {action: 'submit'}
                            ).then(
                                function(token) {
                                    searchButton.disabled = true;

                                    let requestData = parseRequestData();
                                    
                                    //let socket = new WebSocket();
                                    let socket = new WebSocket(
                                        (
                                            location.host.startsWith("dev.ryuar.in")
                                            ? "ws://127.0.0.1:57381"
                                            : (location.protocol == "https:" ? "wss://" : "ws://") + location.host
                                        ) + "/api/analysis"
                                    );
                                    
                                    socket.onopen = function(e) {
                                        socket.send(token);
                                    };
                                    
                                    let ok = false;
                                    socket.onclose = function(event) {
                                        searchButton.disabled = false;
                                        
                                        console.log(event);
                                        if (!ok && !event.wasClean) {
                                            content.innerHTML = "오류 발생";
                                        }

                                        searchButton.disabled = false;
                                    };
                                    
                                    socket.onerror = function(error) {
                                        console.log(error);
                                        searchButton.disabled = false;
                                    };
                                    
                                    socket.onmessage = function(event) {
                                        let resp = JSON.parse(event.data);
                                        
                                        switch (resp.event) {
                                            case "ready":
                                                socket.send(JSON.stringify(requestData));
                                                break;
                                            case "waiting":
                                                content.innerHTML = `${SPINNER} 대기열 ${resp.data} 번 째`;
                                                break;
                                            case "start":
                                                content.innerHTML = `${SPINNER} 분석 시작`;
                                                break;
                                            case "progress":
                                                content.innerHTML = `${SPINNER} ${resp.data}`;
                                                break;
                                            case "error":
                                                content.innerText = "오류 발생";
                                                break;
                                            case "complete":
                                                content.innerHTML = resp.data;
                                                loadCompleted();

                                                ok = true;
                                                break;
                                        }
                                    };
                                }
                            );
                        }
                    );
                }
            );
        }
    );
}