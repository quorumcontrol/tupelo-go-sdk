import { p2p } from './node';
import { TupeloWasm } from './tupelo';

test('creates node', async (done)=> {
    var node = await p2p.createNode();
    expect(node).toBeTruthy();
    node.start(()=> {
        node.stop();
        done();
    });
})

test('pubsubroundtrip', async () => {
    var node = await p2p.createNode();
    expect(node).toBeTruthy();

    const tw = await TupeloWasm.get()

    node.start(async ()=> {
       var resp = await tw.testpubsub(node.pubsub);
       expect(resp).toBe("hi")
    });
})