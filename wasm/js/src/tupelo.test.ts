import './extendedglobal';

import { TupeloWasm, Tupelo } from './tupelo';

test('function population', async() => {
  const wasm = await TupeloWasm.get()
  expect(wasm).not.toBeNull()
})

test('generateKey', async () => {
  const key = await Tupelo.generateKey()
  expect(key).toHaveLength(32)
})
