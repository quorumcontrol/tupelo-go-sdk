const WS = require('libp2p-websockets')
const Multiplex = require('libp2p-mplex')
require('libp2p-crypto-secp256k1') // no need to do anything with this, just require it.
const SECIO = require('libp2p-secio')
const Bootstrap = require('libp2p-bootstrap')
const KadDHT = require('libp2p-kad-dht')
const libp2p = require('libp2p')
const mergeOptions = require('merge-options')
const multiaddr = require('multiaddr')
const PeerInfo = require('peer-info')
const crypto = require('libp2p-crypto').keys
const PeerId = require('peer-id')

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
const bootstrapers = [
  "/ip4/192.168.2.112/tcp/53761/ws/ipfs/Qmdo9yooFfoxLCECUv9MD3YmWcvoTt9mAMLnu3RypTYUMx"
  // "/ip4/192.168.2.112/tcp/64302/ws/ipfs/QmZpDxFWd6fyVspmkeKYwhscXPwQrWHdt1C39EPEcPQpuv"
  // "/ip4/192.168.2.112/tcp/9000/http/p2p-webrtc-direct/ipfs/16Uiu2HAm3TGSEKEjagcCojSJeaT5rypaeJMKejijvYSnAjviWwV5"
]

class TupeloP2P extends libp2p {
  constructor (_options) {
    const defaults = {
      switch: {
        blacklistTTL: 2 * 60 * 1e3, // 2 minute base
        blackListAttempts: 5, // back off 5 times
        maxParallelDials: 100,
        maxColdCalls: 25,
        dialTimeout: 20e3
      },
      modules: {
        transport: [
          WS,
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
    }

    super(mergeOptions(defaults, _options))
  }
}

module.exports.TupeloP2P = TupeloP2P

module.exports.CreateNode = async function() {
  var resolve, reject;
  const p = new Promise((res,rej) => {
      resolve = res;
      reject = rej;
  });
  crypto.generateKeyPair('secp256k1', (err, key) => {
    if (err) {
      console.error("error generating key pair ", err);
      reject(err);
    }

    key.public.hash((err, digest) => {
      if (err) {
        console.error("error hashing ", err)
        reject(err);
      }
      const peerID = new PeerId(digest, key, key.public);
      const peerInfo = new PeerInfo(peerID);
      const node = new TupeloP2P({
        peerInfo
      });
      const peerIdStr = peerID.toB58String();
      console.log("peerIdStr ", peerIdStr);
      node.idStr = peerIdStr;
      resolve(node);
    })
  })
  return p;
}
