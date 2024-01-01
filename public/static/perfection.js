let showScoreChanged = genShowSimpleChange('showScore', 'score');
let showKillsChanged = genShowSimpleChange('showKills', 'kills');
let showDpsChanged   = genShowSimpleChange('showDps'  , 'dps'  );
let showHpsChanged   = genShowSimpleChange('showHps'  , 'hps'  );

addHandler(
    function() {
        let preset = '';
        switch (document.querySelector('input[name="encounter"]:checked').value) {
        case 'raids_64'  : preset = "64"; break;
        case 'raids_62'  : preset = "62"; break;
        case 'raids_60'  : preset = "60"; break;
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
            'service'     : 'perfection',
            'char_name'   : document.getElementById('charName').value,
            'char_server' : server,
            'char_region' : region,
            'preset'      : preset,
        };
    },
    function() {
        try {
            showScoreChanged();
            showKillsChanged();
            showDpsChanged();
            showHpsChanged();

            document.getElementById('showScore').addEventListener('change', showScoreChanged);
            document.getElementById('showKills').addEventListener('change', showKillsChanged);    
            document.getElementById('showDps'  ).addEventListener('change', showDpsChanged  );
            document.getElementById('showHps'  ).addEventListener('change', showHpsChanged  );
        } catch {
        }
    }
);
