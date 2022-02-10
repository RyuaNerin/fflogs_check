let ENCOUNTER_REQUEST_DATA = {
    'eden': [ 73, 74, 75, 76, 77, ],
    'ultimate':  [ 1047, 1048, 1050 ],
};

let ALL_JOBS = [
    "Paladin", "Warrior", "DarkKnight", "Gunbreaker", 
    "WhiteMage", "Scholar", "Astrologian", /* "Sage", */ 
    "Monk", "Dragoon", "Ninja", "Samurai", /* "Reaper", */ 
    "Bard", "Machinist", "Dancer", 
    "BlackMage", "Summoner", "RedMage",
];

let SPINNER = `
<div class="spinner-border text-primary spinner-border-sm me-3" role="status">
    <span class="visually-hidden">Loading...</span>
</div>`;

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

                                let jobs = [];

                                ALL_JOBS.forEach(job => {
                                    if (document.getElementById(`job${job}`).checked) {
                                        jobs.push(job);
                                    }
                                });

                                let encounter = document.querySelector('input[name="encounter"]:checked').value;

                                let requestData = {
                                    'char_name'   : document.getElementById('charName').value,
                                    'char_server' : document.getElementById('charServer').value,
                                    'char_region' : "kr",
                                    'encounters'  : ENCOUNTER_REQUEST_DATA[encounter],
                                    'partitions'  : document.getElementById('includeEcho').checked ? [ 17 ] : null,
                                    'jobs'        : jobs,
                                };
                                
                                //let socket = new WebSocket();
                                let socket = new WebSocket(
                                    (
                                        location.host == "dev.ryuar.in:5500"
                                        ? "ws://127.0.0.1:5555"
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