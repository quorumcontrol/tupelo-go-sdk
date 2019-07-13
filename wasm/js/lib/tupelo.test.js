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
System.register("node", ["./js/p2p"], function (exports_1, context_1) {
    "use strict";
    var libp2p, p2p;
    var __moduleName = context_1 && context_1.id;
    return {
        setters: [
            function (libp2p_1) {
                libp2p = libp2p_1;
            }
        ],
        execute: function () {
            (function (p2p) {
                function createNode() {
                    return __awaiter(this, void 0, void 0, function () {
                        return __generator(this, function (_a) {
                            return [2 /*return*/, libp2p.CreateNode()];
                        });
                    });
                }
                p2p.createNode = createNode;
            })(p2p || (p2p = {}));
            exports_1("p2p", p2p);
        }
    };
});
System.register("tupelo", ["./js/go"], function (exports_2, context_2) {
    "use strict";
    var go, FakePublisher, UnderlyingWasm, TupeloWasm, Tupelo;
    var __moduleName = context_2 && context_2.id;
    return {
        setters: [
            function (go_1) {
                go = go_1;
            }
        ],
        execute: function () {
            FakePublisher = /** @class */ (function () {
                function FakePublisher() {
                }
                FakePublisher.prototype.publish = function (topic, data, cb) {
                    console.log("publishing ", data, " on ", topic, "cb ", cb);
                    cb(null);
                };
                return FakePublisher;
            }());
            UnderlyingWasm = /** @class */ (function () {
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
            })(TupeloWasm || (TupeloWasm = {}));
            exports_2("TupeloWasm", TupeloWasm);
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
            })(Tupelo || (Tupelo = {}));
            exports_2("Tupelo", Tupelo);
            exports_2("default", Tupelo);
        }
    };
});
System.register("tupelo.test", ["./extendedglobal", "node", "tupelo", "tupelo-messages"], function (exports_3, context_3) {
    "use strict";
    var _this, node_1, tupelo_1, tupelo_messages_1;
    _this = this;
    var __moduleName = context_3 && context_3.id;
    return {
        setters: [
            function (_1) {
            },
            function (node_1_1) {
                node_1 = node_1_1;
            },
            function (tupelo_1_1) {
                tupelo_1 = tupelo_1_1;
            },
            function (tupelo_messages_1_1) {
                tupelo_messages_1 = tupelo_messages_1_1;
            }
        ],
        execute: function () {
            test('function population', function () { return __awaiter(_this, void 0, void 0, function () {
                var wasm;
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0: return [4 /*yield*/, tupelo_1.TupeloWasm.get()];
                        case 1:
                            wasm = _a.sent();
                            expect(wasm).not.toBeNull();
                            return [2 /*return*/];
                    }
                });
            }); });
            test('generateKey', function () { return __awaiter(_this, void 0, void 0, function () {
                var key;
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0: return [4 /*yield*/, tupelo_1.Tupelo.generateKey()];
                        case 1:
                            key = _a.sent();
                            expect(key).toHaveLength(32);
                            return [2 /*return*/];
                    }
                });
            }); });
            test.only('end-to-end', function (done) { return __awaiter(_this, void 0, void 0, function () {
                var node, key, trans, payload;
                var _this = this;
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0: return [4 /*yield*/, node_1.p2p.createNode()];
                        case 1:
                            node = _a.sent();
                            expect(node).toBeTruthy();
                            return [4 /*yield*/, tupelo_1.Tupelo.generateKey()];
                        case 2:
                            key = _a.sent();
                            trans = new tupelo_messages_1.Transaction();
                            payload = new tupelo_messages_1.SetDataPayload();
                            payload.setPath("/hi");
                            payload.setValue("hihi!");
                            trans.setType(tupelo_messages_1.Transaction.Type.SETDATA);
                            trans.setSetDataPayload(payload);
                            node.on('connection:start', function (peer) {
                                console.log("connecting to ", peer, " started");
                            });
                            node.on('error', function (err) {
                                console.log('error');
                            });
                            node.once('peer:connect', function () { return __awaiter(_this, void 0, void 0, function () {
                                return __generator(this, function (_a) {
                                    console.log("peer connected");
                                    tupelo_1.Tupelo.playTransactions(node.pubsub, key, [trans]).then(function (success) {
                                        expect(success).toBeTruthy();
                                        done();
                                    }, function (err) {
                                        expect(err).toBeNull();
                                        done();
                                    });
                                    return [2 /*return*/];
                                });
                            }); });
                            node.start(function () {
                                console.log("node started");
                            });
                            return [2 /*return*/];
                    }
                });
            }); });
        }
    };
});
