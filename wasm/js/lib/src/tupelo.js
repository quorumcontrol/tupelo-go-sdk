"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : new P(function (resolve) { resolve(result.value); }).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (_) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (Object.hasOwnProperty.call(mod, k)) result[k] = mod[k];
    result["default"] = mod;
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
var go = __importStar(require("./js/go"));
var FakePublisher = /** @class */ (function () {
    function FakePublisher() {
    }
    FakePublisher.prototype.publish = function (topic, data, cb) {
        console.log("publishing ", data, " on ", topic, "cb ", cb);
        cb(null);
    };
    return FakePublisher;
}());
var UnderlyingWasm = /** @class */ (function () {
    function UnderlyingWasm() {
        this._populated = false;
    }
    UnderlyingWasm.prototype.testpubsub = function (publisher) {
        return new Promise(function (res, rej) { }); // replaced by wasm
    };
    UnderlyingWasm.prototype.generateKey = function () {
        return new Promise(function (res, rej) { }); // replaced by wasm
    };
    UnderlyingWasm.prototype.testclient = function (publisher, keys, transactions) {
        return new Promise(function (res, rej) { }); // replaced by wasm
    };
    return UnderlyingWasm;
}());
var TupeloWasm;
(function (TupeloWasm) {
    var _tupelowasm = new UnderlyingWasm();
    function get() {
        return __awaiter(this, void 0, void 0, function () {
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0:
                        if (_tupelowasm._populated) {
                            return [2 /*return*/, _tupelowasm];
                        }
                        go.run("../main.wasm");
                        return [4 /*yield*/, go.ready()];
                    case 1:
                        _a.sent();
                        go.populate(_tupelowasm);
                        _tupelowasm._populated = true;
                        return [2 /*return*/, _tupelowasm];
                }
            });
        });
    }
    TupeloWasm.get = get;
})(TupeloWasm = exports.TupeloWasm || (exports.TupeloWasm = {}));
var Tupelo;
(function (Tupelo) {
    function generateKey() {
        return __awaiter(this, void 0, void 0, function () {
            var tw;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, TupeloWasm.get()];
                    case 1:
                        tw = _a.sent();
                        return [2 /*return*/, tw.generateKey()];
                }
            });
        });
    }
    Tupelo.generateKey = generateKey;
    function playTransactions(publisher, key, transactions) {
        return __awaiter(this, void 0, void 0, function () {
            var tw, bits, _i, transactions_1, t;
            return __generator(this, function (_a) {
                switch (_a.label) {
                    case 0: return [4 /*yield*/, TupeloWasm.get()];
                    case 1:
                        tw = _a.sent();
                        bits = new Array();
                        for (_i = 0, transactions_1 = transactions; _i < transactions_1.length; _i++) {
                            t = transactions_1[_i];
                            bits = bits.concat(t.serializeBinary());
                        }
                        return [2 /*return*/, tw.testclient(publisher, key, bits)];
                }
            });
        });
    }
    Tupelo.playTransactions = playTransactions;
})(Tupelo = exports.Tupelo || (exports.Tupelo = {}));
exports.default = Tupelo;
