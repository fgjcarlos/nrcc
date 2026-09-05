/**
 * Node-RED 5 settings.js fixture with callback-style middleware and nested
 * braces inside operator-owned functions.
 *
 * Exercises the brace-counting scanner used by ApplyBlockEdit and
 * findTopLevelBlock: a `httpMiddleware` whose body contains
 *   - a nested function expression,
 *   - string literals with single, double and backtick quotes,
 *   - block comments /* ... *\/, and
 *   - line comments with //.
 *
 * Any switch back to a regex matcher would miscount the braces and either
 * truncate the replacement or leave a dangling `httpMiddleware: null,`
 * behind — the original regex bug surfaced as #723. Slice C of #757 pins
 * this behaviour to a fixture so a future refactor that re-introduces
 * regex-based block detection is caught at the test stage.
 */

module.exports = {
    uiPort: 1880,
    httpAdminRoot: '/',

    httpMiddleware: function(req, res, next) {
        /* operator-owned block comment: the brace counter must still
           walk past this and find the matching '}' of the outer fn */
        const start = Date.now();
        function reply(message) {
            res.setHeader('X-Powered-By', "node-red-fixture");
            res.write(message);
            res.end();
        }
        res.on('finish', () => {
            // inline comment with a brace: {not really}
            console.log(req.method, req.url, res.statusCode,
                `took ${Date.now() - start}ms`);
        });
        next();
    },

    httpNodeRoot: '/api/',

    // a block whose body has strings with mixed quotes — must survive
    httpNodeAuth: {
        user: "alice",
        pass: "$2a$08$XXXXXXXXXXXXXXXXXXXXX",
        // operator comment with a closing brace: } 
        hint: 'use "strong" passwords'
    },

    adminAuth: {
        type: "credentials",
        users: [{
            username: "admin",
            password: "$2a$08$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
            permissions: "*"
        }]
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
