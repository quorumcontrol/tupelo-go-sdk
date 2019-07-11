import { p2p } from './node'

test('creates node', async ()=> {
    var node = await p2p.createNode();
    expect(node).toBeTruthy();
})