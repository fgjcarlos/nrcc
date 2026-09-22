module.exports = {
  flowFile: 'flows.json',
  uiPort: 1880,
  nodeDefaults: {
    "inject": { "repeat": false, "crontab": "" },
    "function": { "outputs": 1, "noerr": 0 },
  },
  httpMiddleware: function(req, res, next) { next(); },
}
