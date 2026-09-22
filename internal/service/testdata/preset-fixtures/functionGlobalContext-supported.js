module.exports = {
  flowFile: 'flows.json',
  uiPort: 1880,
  functionGlobalContext: {
    FOO: 'bar',
    PORT_NUMBER: 3000,
    ENABLE_FEATURE: true,
  },
  // operator-owned code below must stay byte-stable through any preset apply
  httpMiddleware: function(req, res, next) { next(); },
  nodesDir: '/data/nodes',
}
