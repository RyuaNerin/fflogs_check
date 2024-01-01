const bubbleRadius = 4;

const colors = [
    "#FFCDD2",
    "#E1BEE7",
    "#C5CAE9",
    "#BBDEFB",

    "#B2DFDB",
    "#DCEDC8",
    "#FFF9C4",
    "#FFE0B2",

    "#F44336",
    "#9C27B0",
    "#3F51B5",
    "#03A9F4",

    "#4CAF50",
    "#CDDC39",
    "#FFC107",
    "#FF5722",
];

function refreshGraph(arg) {
    const encounter = arg.encounter;
    const job = arg.job;
    const data = arg.data;

    let chartDatasetMap = {};
    let charActions = [];

    for (const encounterID in encounter) {
        let charDataset = {
            labels: [],
            datasets: [],
        };

        for (const jobMyID in job) {
            for (const jobPartnerID in job) {
                const v = data[encounterID][jobMyID][jobPartnerID];

                if (!v) {
                    continue;
                }

                const label = `${job[jobMyID]}-${job[jobPartnerID]}`;
                let dataset = {
                    parsing: false,
                    backgroundColor: colors[Number(jobPartnerID) * 4 + Number(jobMyID)],
                    label: label,
                    data: [],
                };

                for (let i = 0; i < v.length; i++) {
                    dataset.data.push({
                        x: v[i].pn,
                        y: v[i].me,
                        r: bubbleRadius,
                        raw: v[i],
                    })                    
                }

                charDataset.labels.push(label);
                charDataset.datasets.push(dataset);
            }
        }

        chartDatasetMap[encounterID] = charDataset;
    }

    const ctx = document.getElementById('chart').getContext('2d');
    const chart = new Chart(
        ctx,
        {
            type: 'scatter',
            data: chartDatasetMap[0],
            options: {
                animation: false,
                responsive: true,
                maintainAspectRatio: true,
                onClick: (e) => {
                    const canvasPosition = Chart.helpers.getRelativePosition(e, chart);
        
                    // Substitute the appropriate scale IDs
                    const dataX = chart.scales.x.getValueForPixel(canvasPosition.x);
                    const dataY = chart.scales.y.getValueForPixel(canvasPosition.y);

                    console.log(dataX);
                    console.log(dataY);
                },
                interaction: {
                    axis: "xy",
                    mode: "point",
                },
                scales : {
                    y: {
                        title: {
                            display: true,
                            text: "HPS Rank",
                        },
                        type: 'linear',
                        min: 0,
                        max: 100,
                        tick: 10,
                    },
                    x: {
                        title: {
                            display: true,
                            text: "짝힐 HPS Rank",
                        },
                        type: 'linear',
                        min: 0,
                        max: 100,
                        tick: 10,
                    }
                },
                elements: {
                    point: {
                        radius: bubbleRadius,
                        hitRadius: bubbleRadius,
                        hoverRadius: bubbleRadius,
                    },
                },
                plugins: {
                    legend: {
                        position: 'top',
                    },
                    title: {
                        display: true,
                        text: `HPS 비교 그래프 (${arg.charname})`,
                        font: {
                            size: 18,
                        },
                    },
                    subtitle: {
                        display: true,
                        text: 'By RyuaNerin',
                        align: 'end',
                    },
                    tooltip: {
                        callbacks: {
                            label: function(item) {
                                const data = item.raw.raw;
                                return `${data.mename}: ${data.me} / ${data.pnname}: ${data.pn}`
                            }
                        }
                    },
                    zoom: {
                        limits: {
                            x: {min: 0, max: 100, minRange: 10},
                            y: {min: 0, max: 100, minRange: 10}
                        },
                        pan: {
                            enabled: true,
                        },
                        zoom: {
                            wheel: {
                                enabled: true,
                            },
                            pinch: {
                                enabled: true,
                            },
                            mode: 'xy',
                        },
                    },
                    annotation: {
                        annotations: {
                            pentagon: {
                                type: 'line',
                                borderColor: 'red',
                                borderWidth: 1,
                                xMin: 0,
                                xMax: 100,
                                yMin: 0,
                                yMax: 100,
                            }
                        }
                    }
                },
            },
        }
    );

    var buttons = document.getElementsByClassName("chart-encounter")
    for (const button of buttons) {
        button.addEventListener('click', function() {
            chart.data = chartDatasetMap[Number(button.getAttribute("data-encounter-id"))];
            chart.update();
        });
    }
}

addHandler(
    function() {
        let preset = '';
        switch (document.querySelector('input[name="encounter"]:checked').value) {
        case 'raids_64': preset = "64"; break;
        case 'raids_62': preset = "62"; break;
        case 'raids_60': preset = "60"; break;
        case 'trial_54'  : preset = "54_trial"; break;
        case 'trial_60'  : preset = "60_trial"; break;
        case 'ultimate_6': preset = "6_ulti"; break;
        case 'ultimate_5': preset = "5_ulti"; break;

        case 'eden_promise':
            preset =
                document.getElementById('includeEcho').checked
                ? "54_echo"
                : "54"
            break;
        }

        let m = /^(..)_(.+)$/.exec(document.getElementById('charServer').value);
        let region = m[1];
        let server = m[2];

        return {
            'service'     : 'hps',
            'char_name'   : document.getElementById('charName').value,
            'char_server' : server,
            'char_region' : region,
            'preset'      : preset,
        };
    },
    function() {
        nodeScriptReplace(document.getElementById('content'));
    }
);

function nodeScriptReplace(node) {
    if ( nodeScriptIs(node) === true ) {
            node.parentNode.replaceChild( nodeScriptClone(node) , node );
    }
    else {
            var i = -1, children = node.childNodes;
            while ( ++i < children.length ) {
                  nodeScriptReplace( children[i] );
            }
    }

    return node;
}
function nodeScriptClone(node){
    var script  = document.createElement("script");
    script.text = node.innerHTML;

    var i = -1, attrs = node.attributes, attr;
    while ( ++i < attrs.length ) {                                    
          script.setAttribute( (attr = attrs[i]).name, attr.value );
    }
    return script;
}

function nodeScriptIs(node) {
    return node.tagName === 'SCRIPT';
}