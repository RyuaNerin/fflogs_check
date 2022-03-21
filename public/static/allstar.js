addHandler(
    function() {
        let m = /^(..)_(.+)$/.exec(document.getElementById('charServer').value);
        let region = m[1];
        let server = m[2];

        return {
            'service'     : 'allstar',
            'char_name'   : document.getElementById('charName').value,
            'char_server' : server,
            'char_region' : region,
            'preset'      : document.querySelector('input[name="encounter"]:checked').value,
        };
    },
    function() {
    }
);