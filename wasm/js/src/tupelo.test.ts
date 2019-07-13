import './extendedglobal';
import { p2p } from './node';

import { TupeloWasm, Tupelo } from './tupelo';
import { Transaction,SetDataPayload } from '/Users/tobowers/code/messages/build/js/transactions/transactions_pb'
const dagCBOR = require('ipld-dag-cbor');



test('function population', async() => {
  const wasm = await TupeloWasm.get()
  expect(wasm).not.toBeNull()
})

test('generateKey', async () => {
  const key = await Tupelo.generateKey()
  expect(key).toHaveLength(32)
})

test.only('end-to-end', async (done) => {
  jest.setTimeout(10000);

  var node = await p2p.createNode();
  expect(node).toBeTruthy();

  // const tw = await TupeloWasm.get()
  const key = await Tupelo.generateKey()
  const trans = new Transaction()

  const payload = new SetDataPayload()
  payload.setPath("/hi")

  const serialized = dagCBOR.util.serialize("hihi")

  payload.setValue(new Uint8Array(serialized))
  trans.setType(Transaction.Type.SETDATA)
  trans.setSetDataPayload(payload)

  node.on('connection:start', (peer:any) => {
    console.log("connecting to ", peer.id._idB58String, " started")
  })

  node.on('error', (err:any) => {
    console.log('error')
  })

  node.on('peer:connect', async () => {
    console.log("peer connect")
  })

  node.once('peer:connect', async ()=> {
    Tupelo.playTransactions(node.pubsub, key, [trans]).then(
      (success)=> {
        expect(success).toBeTruthy()
        done()
      },
      (err)=> {
        expect(err).toBeNull()
        done()
    })
  })

  node.start(()=>{
    console.log("node started");
  });
})