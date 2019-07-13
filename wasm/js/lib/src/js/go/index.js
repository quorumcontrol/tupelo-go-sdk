"use strict";
// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
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
var _this = this;
(function () {
    if (typeof global !== "undefined") {
        // global already exists
    }
    else if (typeof window !== "undefined") {
        window.global = window;
    }
    else if (typeof self !== "undefined") {
        self.global = self;
    }
    else {
        throw new Error("cannot export Go (neither global, window nor self is defined)");
    }
    var encoder, decoder;
    // Map web browser API and Node.js API to a single common API (preferring web standards over Node.js API).
    var isNodeJS = true; //global.process && global.process.title === "node";
    if (isNodeJS) {
        global.require = require;
        global.fs = require("fs");
        var nodeCrypto_1 = require("crypto");
        global.crypto = {
            getRandomValues: function (b) {
                nodeCrypto_1.randomFillSync(b);
            },
        };
        global.performance = {
            now: function () {
                var _a = process.hrtime(), sec = _a[0], nsec = _a[1];
                return sec * 1000 + nsec / 1000000;
            },
        };
        var util = require("util");
        encoder = new util.TextEncoder;
        decoder = new util.TextDecoder;
    }
    else {
        var outputBuf_1 = "";
        global.fs = {
            constants: { O_WRONLY: -1, O_RDWR: -1, O_CREAT: -1, O_TRUNC: -1, O_APPEND: -1, O_EXCL: -1 },
            writeSync: function (fd, buf) {
                outputBuf_1 += decoder.decode(buf);
                var nl = outputBuf_1.lastIndexOf("\n");
                if (nl != -1) {
                    console.log(outputBuf_1.substr(0, nl));
                    outputBuf_1 = outputBuf_1.substr(nl + 1);
                }
                return buf.length;
            },
            write: function (fd, buf, offset, length, position, callback) {
                if (offset !== 0 || length !== buf.length || position !== null) {
                    throw new Error("not implemented");
                }
                var n = this.writeSync(fd, buf);
                callback(null, n);
            },
            open: function (path, flags, mode, callback) {
                var err = new Error("not implemented");
                err.code = "ENOSYS";
                callback(err);
            },
            read: function (fd, buffer, offset, length, position, callback) {
                var err = new Error("not implemented");
                err.code = "ENOSYS";
                callback(err);
            },
            fsync: function (fd, callback) {
                callback(null);
            },
        };
        encoder = new TextEncoder("utf-8");
        decoder = new TextDecoder("utf-8");
    }
    global.Go = /** @class */ (function () {
        function class_1() {
            var _this = this;
            this.argv = ["js"];
            this.env = {};
            this.exit = function (code) {
                if (code !== 0) {
                    console.warn("exit code:", code);
                }
            };
            this._exitPromise = new Promise(function (resolve) {
                _this._resolveExitPromise = resolve;
            });
            this._pendingEvent = null;
            this._scheduledTimeouts = new Map();
            this._nextCallbackTimeoutID = 1;
            var mem = function () {
                // The buffer may change when requesting more memory.
                return new DataView(_this._inst.exports.mem.buffer);
            };
            var setInt64 = function (addr, v) {
                mem().setUint32(addr + 0, v, true);
                mem().setUint32(addr + 4, Math.floor(v / 4294967296), true);
            };
            var getInt64 = function (addr) {
                var low = mem().getUint32(addr + 0, true);
                var high = mem().getInt32(addr + 4, true);
                return low + high * 4294967296;
            };
            var loadValue = function (addr) {
                var f = mem().getFloat64(addr, true);
                if (f === 0) {
                    return undefined;
                }
                if (!isNaN(f)) {
                    return f;
                }
                var id = mem().getUint32(addr, true);
                return _this._values[id];
            };
            var storeValue = function (addr, v) {
                var nanHead = 0x7FF80000;
                if (typeof v === "number") {
                    if (isNaN(v)) {
                        mem().setUint32(addr + 4, nanHead, true);
                        mem().setUint32(addr, 0, true);
                        return;
                    }
                    if (v === 0) {
                        mem().setUint32(addr + 4, nanHead, true);
                        mem().setUint32(addr, 1, true);
                        return;
                    }
                    mem().setFloat64(addr, v, true);
                    return;
                }
                switch (v) {
                    case undefined:
                        mem().setFloat64(addr, 0, true);
                        return;
                    case null:
                        mem().setUint32(addr + 4, nanHead, true);
                        mem().setUint32(addr, 2, true);
                        return;
                    case true:
                        mem().setUint32(addr + 4, nanHead, true);
                        mem().setUint32(addr, 3, true);
                        return;
                    case false:
                        mem().setUint32(addr + 4, nanHead, true);
                        mem().setUint32(addr, 4, true);
                        return;
                }
                var ref = _this._refs.get(v);
                if (ref === undefined) {
                    ref = _this._values.length;
                    _this._values.push(v);
                    _this._refs.set(v, ref);
                }
                var typeFlag = 0;
                switch (typeof v) {
                    case "string":
                        typeFlag = 1;
                        break;
                    case "symbol":
                        typeFlag = 2;
                        break;
                    case "function":
                        typeFlag = 3;
                        break;
                }
                mem().setUint32(addr + 4, nanHead | typeFlag, true);
                mem().setUint32(addr, ref, true);
            };
            var loadSlice = function (addr) {
                var array = getInt64(addr + 0);
                var len = getInt64(addr + 8);
                return new Uint8Array(_this._inst.exports.mem.buffer, array, len);
            };
            var loadSliceOfValues = function (addr) {
                var array = getInt64(addr + 0);
                var len = getInt64(addr + 8);
                var a = new Array(len);
                for (var i = 0; i < len; i++) {
                    a[i] = loadValue(array + i * 8);
                }
                return a;
            };
            var loadString = function (addr) {
                var saddr = getInt64(addr + 0);
                var len = getInt64(addr + 8);
                return decoder.decode(new DataView(_this._inst.exports.mem.buffer, saddr, len));
            };
            var timeOrigin = Date.now() - performance.now();
            this.importObject = {
                go: {
                    // Go's SP does not change as long as no Go code is running. Some operations (e.g. calls, getters and setters)
                    // may synchronously trigger a Go event handler. This makes Go code get executed in the middle of the imported
                    // function. A goroutine can switch to a new stack if the current stack is too small (see morestack function).
                    // This changes the SP, thus we have to update the SP used by the imported function.
                    // func wasmExit(code int32)
                    "runtime.wasmExit": function (sp) {
                        var code = mem().getInt32(sp + 8, true);
                        _this.exited = true;
                        delete _this._inst;
                        delete _this._values;
                        delete _this._refs;
                        _this.exit(code);
                    },
                    // func wasmWrite(fd uintptr, p unsafe.Pointer, n int32)
                    "runtime.wasmWrite": function (sp) {
                        var fd = getInt64(sp + 8);
                        var p = getInt64(sp + 16);
                        var n = mem().getInt32(sp + 24, true);
                        fs.writeSync(fd, new Uint8Array(_this._inst.exports.mem.buffer, p, n));
                    },
                    // func nanotime() int64
                    "runtime.nanotime": function (sp) {
                        setInt64(sp + 8, (timeOrigin + performance.now()) * 1000000);
                    },
                    // func walltime() (sec int64, nsec int32)
                    "runtime.walltime": function (sp) {
                        var msec = (new Date).getTime();
                        setInt64(sp + 8, msec / 1000);
                        mem().setInt32(sp + 16, (msec % 1000) * 1000000, true);
                    },
                    // func scheduleTimeoutEvent(delay int64) int32
                    "runtime.scheduleTimeoutEvent": function (sp) {
                        var id = _this._nextCallbackTimeoutID;
                        _this._nextCallbackTimeoutID++;
                        _this._scheduledTimeouts.set(id, setTimeout(function () { _this._resume(); }, getInt64(sp + 8) + 1));
                        mem().setInt32(sp + 16, id, true);
                    },
                    // func clearTimeoutEvent(id int32)
                    "runtime.clearTimeoutEvent": function (sp) {
                        var id = mem().getInt32(sp + 8, true);
                        clearTimeout(_this._scheduledTimeouts.get(id));
                        _this._scheduledTimeouts.delete(id);
                    },
                    // func getRandomData(r []byte)
                    "runtime.getRandomData": function (sp) {
                        crypto.getRandomValues(loadSlice(sp + 8));
                    },
                    // func stringVal(value string) ref
                    "syscall/js.stringVal": function (sp) {
                        storeValue(sp + 24, loadString(sp + 8));
                    },
                    // func valueGet(v ref, p string) ref
                    "syscall/js.valueGet": function (sp) {
                        var result = Reflect.get(loadValue(sp + 8), loadString(sp + 16));
                        sp = _this._inst.exports.getsp(); // see comment above
                        storeValue(sp + 32, result);
                    },
                    // func valueSet(v ref, p string, x ref)
                    "syscall/js.valueSet": function (sp) {
                        Reflect.set(loadValue(sp + 8), loadString(sp + 16), loadValue(sp + 32));
                    },
                    // func valueIndex(v ref, i int) ref
                    "syscall/js.valueIndex": function (sp) {
                        storeValue(sp + 24, Reflect.get(loadValue(sp + 8), getInt64(sp + 16)));
                    },
                    // valueSetIndex(v ref, i int, x ref)
                    "syscall/js.valueSetIndex": function (sp) {
                        Reflect.set(loadValue(sp + 8), getInt64(sp + 16), loadValue(sp + 24));
                    },
                    // func valueCall(v ref, m string, args []ref) (ref, bool)
                    "syscall/js.valueCall": function (sp) {
                        try {
                            var v = loadValue(sp + 8);
                            var m = Reflect.get(v, loadString(sp + 16));
                            var args = loadSliceOfValues(sp + 32);
                            var result = Reflect.apply(m, v, args);
                            sp = _this._inst.exports.getsp(); // see comment above
                            storeValue(sp + 56, result);
                            mem().setUint8(sp + 64, 1);
                        }
                        catch (err) {
                            storeValue(sp + 56, err);
                            mem().setUint8(sp + 64, 0);
                        }
                    },
                    // func valueInvoke(v ref, args []ref) (ref, bool)
                    "syscall/js.valueInvoke": function (sp) {
                        try {
                            var v = loadValue(sp + 8);
                            var args = loadSliceOfValues(sp + 16);
                            var result = Reflect.apply(v, undefined, args);
                            sp = _this._inst.exports.getsp(); // see comment above
                            storeValue(sp + 40, result);
                            mem().setUint8(sp + 48, 1);
                        }
                        catch (err) {
                            storeValue(sp + 40, err);
                            mem().setUint8(sp + 48, 0);
                        }
                    },
                    // func valueNew(v ref, args []ref) (ref, bool)
                    "syscall/js.valueNew": function (sp) {
                        try {
                            var v = loadValue(sp + 8);
                            var args = loadSliceOfValues(sp + 16);
                            var result = Reflect.construct(v, args);
                            sp = _this._inst.exports.getsp(); // see comment above
                            storeValue(sp + 40, result);
                            mem().setUint8(sp + 48, 1);
                        }
                        catch (err) {
                            storeValue(sp + 40, err);
                            mem().setUint8(sp + 48, 0);
                        }
                    },
                    // func valueLength(v ref) int
                    "syscall/js.valueLength": function (sp) {
                        setInt64(sp + 16, parseInt(loadValue(sp + 8).length));
                    },
                    // valuePrepareString(v ref) (ref, int)
                    "syscall/js.valuePrepareString": function (sp) {
                        var str = encoder.encode(String(loadValue(sp + 8)));
                        storeValue(sp + 16, str);
                        setInt64(sp + 24, str.length);
                    },
                    // valueLoadString(v ref, b []byte)
                    "syscall/js.valueLoadString": function (sp) {
                        var str = loadValue(sp + 8);
                        loadSlice(sp + 16).set(str);
                    },
                    // func valueInstanceOf(v ref, t ref) bool
                    "syscall/js.valueInstanceOf": function (sp) {
                        mem().setUint8(sp + 24, loadValue(sp + 8) instanceof loadValue(sp + 16));
                    },
                    "debug": function (value) {
                        console.log(value);
                    },
                }
            };
        }
        class_1.prototype.run = function (instance) {
            return __awaiter(this, void 0, void 0, function () {
                var mem, offset, strPtr, argc, argvPtrs, keys, argv;
                var _this = this;
                return __generator(this, function (_a) {
                    switch (_a.label) {
                        case 0:
                            this._inst = instance;
                            this._values = [
                                NaN,
                                0,
                                null,
                                true,
                                false,
                                global,
                                this._inst.exports.mem,
                                this,
                            ];
                            this._refs = new Map();
                            this.exited = false;
                            mem = new DataView(this._inst.exports.mem.buffer);
                            offset = 4096;
                            strPtr = function (str) {
                                var ptr = offset;
                                new Uint8Array(mem.buffer, offset, str.length + 1).set(encoder.encode(str + "\0"));
                                offset += str.length + (8 - (str.length % 8));
                                return ptr;
                            };
                            argc = this.argv.length;
                            argvPtrs = [];
                            this.argv.forEach(function (arg) {
                                argvPtrs.push(strPtr(arg));
                            });
                            keys = Object.keys(this.env).sort();
                            argvPtrs.push(keys.length);
                            keys.forEach(function (key) {
                                argvPtrs.push(strPtr(key + "=" + _this.env[key]));
                            });
                            argv = offset;
                            argvPtrs.forEach(function (ptr) {
                                mem.setUint32(offset, ptr, true);
                                mem.setUint32(offset + 4, 0, true);
                                offset += 8;
                            });
                            this._inst.exports.run(argc, argv);
                            if (this.exited) {
                                this._resolveExitPromise();
                            }
                            return [4 /*yield*/, this._exitPromise];
                        case 1:
                            _a.sent();
                            return [2 /*return*/];
                    }
                });
            });
        };
        class_1.prototype._resume = function () {
            if (this.exited) {
                throw new Error("Go program has already exited");
            }
            this._inst.exports.resume();
            if (this.exited) {
                this._resolveExitPromise();
            }
        };
        class_1.prototype._makeFuncWrapper = function (id) {
            var go = this;
            return function () {
                var event = { id: id, this: this, args: arguments };
                go._pendingEvent = event;
                go._resume();
                return event.result;
            };
        };
        return class_1;
    }());
    global.Go.readyPromise = new Promise(function (resolve) {
        global.Go.readyResolver = resolve;
    });
})();
var runner = {
    run: function (path) { return __awaiter(_this, void 0, void 0, function () {
        var go, result;
        return __generator(this, function (_a) {
            switch (_a.label) {
                case 0:
                    go = new Go();
                    go.env = Object.assign({ TMPDIR: require("os").tmpdir() }, process.env);
                    return [4 /*yield*/, WebAssembly.instantiate(fs.readFileSync(path), go.importObject)];
                case 1:
                    result = _a.sent();
                    process.on("exit", function (code) {
                        Go.exit();
                        if (code === 0 && !go.exited) {
                            // deadlock, make Go print error and stack traces
                            go._pendingEvent = { id: 0 };
                            go._resume();
                        }
                    });
                    return [2 /*return*/, go.run(result.instance)];
            }
        });
    }); },
    ready: function (path) { return __awaiter(_this, void 0, void 0, function () {
        return __generator(this, function (_a) {
            return [2 /*return*/, global.Go.readyPromise];
        });
    }); },
    populate: function (obj) {
        return global.populateLibrary(obj);
    }
};
module.exports = runner;
// runner.run("/Users/tobowers/code/tupelo-go-sdk/wasm/main.wasm").then((result) => {
//     console.log(result);
// });
