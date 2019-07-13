"use strict";
var __extends = (this && this.__extends) || (function () {
    var extendStatics = function (d, b) {
        extendStatics = Object.setPrototypeOf ||
            ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
            function (d, b) { for (var p in b) if (b.hasOwnProperty(p)) d[p] = b[p]; };
        return extendStatics(d, b);
    };
    return function (d, b) {
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
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
var WS = require('libp2p-websockets');
var Multiplex = require('pull-mplex');
// const Multiplex = require('libp2p-mplex')
require('libp2p-crypto-secp256k1'); // no need to do anything with this, just require it.
var SECIO = require('libp2p-secio');
var Bootstrap = require('libp2p-bootstrap');
var KadDHT = require('libp2p-kad-dht');
var libp2p = require('libp2p');
var mergeOptions = require('merge-options');
var multiaddr = require('multiaddr');
var PeerInfo = require('peer-info');
var crypto = require('libp2p-crypto').keys;
var PeerId = require('peer-id');
var TCP = require('libp2p-tcp');
// Find this list at: https://github.com/ipfs/js-ipfs/blob/master/src/core/runtime/config-nodejs.json
// const bootstrapers = [
//   '/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ',
//   '/ip4/104.236.176.52/tcp/4001/p2p/QmSoLnSGccFuZQJzRadHn95W2CrSFmZuTdDWP8HXaHca9z',
//   '/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM',
//   '/ip4/162.243.248.213/tcp/4001/p2p/QmSoLueR4xBeUbY9WZ9xGUUxunbKWcrNFTDAadQJmocnWm',
//   '/ip4/128.199.219.111/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu',
//   '/ip4/104.236.76.40/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64',
//   '/ip4/178.62.158.247/tcp/4001/p2p/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd',
//   '/ip4/178.62.61.185/tcp/4001/p2p/QmSoLMeWqB7YGVLJN3pNLQpmmEk35v6wYtsMGLzSr5QBU3',
//   '/ip4/104.236.151.122/tcp/4001/p2p/QmSoLju6m7xTh3DuokvT3886QRYqxAzb1kShaasnJgW36yx'
// ]
var bootstrapers = [
    "/ip4/192.168.2.112/tcp/34001/ipfs/16Uiu2HAm3TGSEKEjagcCojSJeaT5rypaeJMKejijvYSnAjviWwV5"
    // "/ip4/192.168.2.112/tcp/64302/ws/ipfs/QmZpDxFWd6fyVspmkeKYwhscXPwQrWHdt1C39EPEcPQpuv"
    // "/ip4/192.168.2.112/tcp/9000/http/p2p-webrtc-direct/ipfs/16Uiu2HAm3TGSEKEjagcCojSJeaT5rypaeJMKejijvYSnAjviWwV5"
];
var TupeloP2P = /** @class */ (function (_super) {
    __extends(TupeloP2P, _super);
    function TupeloP2P(_options) {
        var _this = this;
        var defaults = {
            switch: {
                blacklistTTL: 2 * 60 * 1e3,
                blackListAttempts: 5,
                maxParallelDials: 100,
                maxColdCalls: 25,
                dialTimeout: 20e3
            },
            modules: {
                transport: [
                    WS,
                    TCP,
                ],
                streamMuxer: [
                    Multiplex
                ],
                connEncryption: [
                    SECIO
                ],
                peerDiscovery: [
                    Bootstrap
                ],
                dht: KadDHT
            },
            config: {
                peerDiscovery: {
                    autoDial: true,
                    bootstrap: {
                        enabled: true,
                        list: bootstrapers
                    }
                },
                dht: {
                    enabled: false
                },
                EXPERIMENTAL: {
                    pubsub: true
                }
            }
        };
        _this = _super.call(this, mergeOptions(defaults, _options)) || this;
        return _this;
    }
    return TupeloP2P;
}(libp2p));
module.exports.TupeloP2P = TupeloP2P;
module.exports.CreateNode = function () {
    return __awaiter(this, void 0, void 0, function () {
        var resolve, reject, p;
        return __generator(this, function (_a) {
            p = new Promise(function (res, rej) {
                resolve = res;
                reject = rej;
            });
            crypto.generateKeyPair('secp256k1', function (err, key) {
                if (err) {
                    console.error("error generating key pair ", err);
                    reject(err);
                }
                key.public.hash(function (err, digest) {
                    if (err) {
                        console.error("error hashing ", err);
                        reject(err);
                    }
                    var peerID = new PeerId(digest, key, key.public);
                    var peerInfo = new PeerInfo(peerID);
                    peerInfo.multiaddrs.add('/ip4/0.0.0.0/tcp/0');
                    var node = new TupeloP2P({
                        peerInfo: peerInfo
                    });
                    var peerIdStr = peerID.toB58String();
                    console.log("peerIdStr ", peerIdStr);
                    node.idStr = peerIdStr;
                    process.on("exit", function () {
                        node.stop();
                    });
                    resolve(node);
                });
            });
            return [2 /*return*/, p];
        });
    });
};
