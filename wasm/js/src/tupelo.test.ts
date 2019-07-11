import './extendedglobal';

import { Tupelo, TupeloWasm } from './tupelo';

test('basic', async () => {
  const res = await Tupelo.publish("test", new Uint8Array([1,2,3,4]))
  expect(res).toBeTruthy();
});

test('function population', async() => {
  const wasm = await TupeloWasm.get()
  expect(wasm).not.toBeNull()
})
