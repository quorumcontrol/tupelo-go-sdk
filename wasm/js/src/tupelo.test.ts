import './extendedglobal';

import { TupeloWasm } from './tupelo';

test('function population', async() => {
  const wasm = await TupeloWasm.get()
  expect(wasm).not.toBeNull()
})
