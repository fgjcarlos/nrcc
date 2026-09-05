/**
 * Node-RED 5 baseline settings.js fixture.
 *
 * Mirrors the canonical file Node-RED ships when an operator runs
 * `node-red` for the first time and accepts the default settings — minimum
 * surface area, no operator-owned extensions. Used by slice C of #757 to
 * prove that import → fingerprint → save round-trips byte-for-byte on a
 * pristine document: the no-op import/save path must leave every byte
 * untouched.
 */

module.exports = {
    uiPort: process.env.PORT || 1880,
    mqttReconnectTime: 15000,
    serialReconnectTime: 15000,
    httpAdminRoot: '/',
    httpNodeRoot: '/',
    https: {
        key: require("fs").readFileSync('/etc/ssl/node-red/server.key'),
        cert: require("fs").readFileSync('/etc/ssl/node-red/server.crt'),
    },
    requireHttps: true,
    httpNodeAuth: {user:"user",pass:"$2a$08$XXXXXXXXXXXXXXXXXXXXX"},
    httpStaticAuth: {user:"user",pass:"$2a$08$XXXXXXXXXXXXXXXXXXXXX"},
    adminAuth: {
        type: "credentials",
        users: [{
            username: "admin",
            password: "$2a$08$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
            permissions: "*"
        }]
    },
    logging: {
        console: {
            level: "info",
            metrics: false,
            audit: false
        }
    },
    editorTheme: {
        projects: {
            enabled: false
        }
    },
    functionGlobalContext: {
        os: require('os')
    }
}
