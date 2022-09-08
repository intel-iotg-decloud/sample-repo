"use strict";

var _module = require("../module");

var _q = require("q");

var _q2 = _interopRequireDefault(_q);

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

describe('GenericDatasource', function () {
    var ctx = {};

    beforeEach(function () {
        ctx.$q = _q2.default;
        ctx.backendSrv = {};
        ctx.templateSrv = {};
        ctx.ds = new _module.Datasource({}, ctx.$q, ctx.backendSrv, ctx.templateSrv);
    });

    it('should return an empty array when no targets are set', function (done) {
        ctx.ds.query({ targets: [] }).then(function (result) {
            expect(result.data).to.have.length(0);
            done();
        });
    });
});
//# sourceMappingURL=datasource_spec.js.map
