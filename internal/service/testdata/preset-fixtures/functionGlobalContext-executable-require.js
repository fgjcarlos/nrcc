module.exports = {
  flowFile: 'flows.json',
  functionGlobalContext: {
    lodash: require("lodash"),
    myHelpers: require("./lib/helpers"),
  },
  externalModules: {
    foo: require("foo"),
  },
}
