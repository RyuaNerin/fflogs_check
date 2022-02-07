let ENCOUNTER_NAME = {
    73: "어둠의 구름",
    74: "그림자의 왕",
    75: "페이트 브레이커",
    76: "에덴의 약속",
    77: "어둠의 무녀",
    1050: "절 알렉산더 토벌전",
    1048: "절 알테마 웨폰 파괴작전",
    1047: "절 바하무트 토벌전",
};

let ENCOUNTER_REQUEST_DATA = {
    'eden': [ 73, 74, 75, 76, 77, ],
    'tea':  [ 1050, ],
    'uwu':  [ 1048, ],
    'ucob': [ 1047, ],
};

let ALL_JOBS = [
    "Paladin", "Warrior", "DarkKnight", "Gunbreaker", 
    "WhiteMage", "Scholar", "Astrologian", /* "Sage", */ 
    "Monk", "Dragoon", "Ninja", "Samurai", /* "Reaper", */ 
    "Bard", "Machinist", "Dancer", 
    "BlackMage", "Summoner", "RedMage",
];

function buildHtml(data) {
    let html = "";
    
    html += `<div class="accordion" id="accordion">`;
    for (let index = 0; index < data.data.length; index++) {
        const element = array.data[index];

        html += `
        <div class="accordion-item">
            <h2 class="accordion-header" id="heading${index}">
                <button class="accordion-button" type="button" data-bs-toggle="collapse" data-bs-target="#collapse${index}" aria-expanded="true" aria-controls="collapse${index}">
                    ${ENCOUNTER_NAME[element.encounter.encounter]}
                </button>
            </h2>
            <div id="collapse${index}" class="accordion-collapse collapse show" aria-labelledby="headingOne" data-bs-parent="#accordion">
            <div class="accordion-body">`;

        html += `
            </div>
        </div>`;
    }
    html += `</div>`
}

document.addEventListener(
    'DOMContentLoaded',
    function () {
        let searchButton = document.getElementById("search");
        let content = document.getElementById("content");

        searchButton.addEventListener(
            'click',
            function(e) {
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
                                            content.innerText = `대기열 ${resp.data} 번 째`;
                                            break;
                                        case "start":
                                            content.innerText = `분석 시작`;
                                            break;
                                        case "progress":
                                            content.innerText = resp.data;
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