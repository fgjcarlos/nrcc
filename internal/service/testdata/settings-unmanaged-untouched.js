/**
 * Node-RED 5 settings.js fixture whose every key is unmanaged or has heavy
 * operator-authored formatting around it.
 *
 * Goals for slice C of #757:
 *   1. No managed edit must drop, reorder or reformat operator comments.
 *   2. Blank lines between unmanaged keys must survive verbatim.
 *   3. Tab/space-mixed indentation must survive verbatim.
 *   4. Multiple top-level comment blocks must survive verbatim.
 *   5. Operator-owned keys not in the managed catalog (httpMiddleware,
 *      externalModules, nodesDirArray, etc.) must survive verbatim.
 *
 * The fixture deliberately mixes 2-space and 4-space indentation across
 * blocks, mirrors a real-world operator "save the file by hand" file, and
 * anchors the byte-for-byte preservation contract that slice A introduced.
 */

// ============================================================================
// Header — operator-authored. Slice C round-trip must keep this intact.
// ============================================================================

// --- Top-level section: HTTP ------------------------------------------------
module.exports = {
    uiPort: 1880,

    // operator comment immediately under uiPort — must survive verbatim
    httpAdminRoot: '/',
    httpNodeRoot: '/api/',

    /* operator-owned block comment with a closing brace inside: } */
    httpMiddleware: function(req, res, next) {
        next();
    },

    // operator-owned single-line comment with a colon: "x: y"
    exportGlobalContextKeys: ['os', 'fs'],

    externalModules: {
        // nested comment with a comma: a, b
        sqlite: require('better-sqlite3'),
    },

    nodesDirArray: [
        '/opt/nrcc/nodes-core',
    ],

    // --- Operator section: AUTH ---------------------------------------------
    adminAuth: {
        type: "credentials",
        users: [{
            username: "admin",
            password: "$2a$08$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
            permissions: "*"
        }],
    },

    // --- Operator section: LOGGING ------------------------------------------
    logging: {
        console: {
            level: "info",
            metrics: false,
            audit: false,
        },
    },

    editorTheme: {
        projects: {
            enabled: false
        }
    },

    functionGlobalContext: {
        // operator-authored alias
        _: require('lodash'),
    },
}

// ============================================================================
// Footer — operator-authored. Slice C round-trip must keep this intact.
// ============================================================================
