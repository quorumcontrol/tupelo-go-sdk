import './extendedglobal';

import { run, ready } from './app';

test('basic', async () => {
  run("../main.wasm");
  await ready();
  expect(global.populateTupelo).toBeInstanceOf(Function);
});
