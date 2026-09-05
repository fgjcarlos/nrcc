/**
 * Node-RED 5 settings.js fixture with operator-authored extensions.
 *
 * Exercises the slice-A source-preserving contract on unmanaged regions:
 *   - httpMiddleware (function expression) must survive every edit verbatim.
 *   - externalModules (array of require() calls) must survive verbatim.
 *   - exportGlobalContextKeys must survive verbatim.
 *   - A top-level operator comment must survive verbatim.
 *
 * Used by slice C of #757 to prove that SourcePatch and ApplyBlockEdit
 * leave every byte NRCC does not explicitly rewrite alone.
 */

module.exports = {
    // operator-authored middleware: rate-limits /api/* and records latency.
    httpMiddleware: function(req, res, next) {
        const start = Date.now();
        res.on('finish', () => {
            console.log(req.method, req.url, res.statusCode, Date.now() - start, 'ms');
        });
        next();
    },

    externalModules: {
        sqlite: require('better-sqlite3'),
        lodash: require('lodash'),
    },

    exportGlobalContextKeys: ['os', 'fs', 'path'],

    uiPort: process.env.PORT || 1880,
    httpAdminRoot: '/',
    httpNodeRoot: '/api/',

    nodesDirArray: [
        '/opt/nrcc/nodes-core',
        '/opt/nrcc/nodes-contrib',
    ],

    adminAuth: {
        type: "credentials",
        users: [{
            username: "admin",
            password: "$2a$08$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
            permissions: "*"
        }]
    },

    functionGlobalContext: {
        lodash: require('lodash'),
        moment: require('moment'),
        // operator comment inside an unmanaged block — must survive
        secrets: require('./secrets.json'),
    },

    editorTheme: {
        projects: {
            enabled: false
        }
    },

    logging: {
        console: {
            level: "info",
            metrics: false,
            audit: false
        }
    }
}
