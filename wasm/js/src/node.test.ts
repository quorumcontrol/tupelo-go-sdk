import { p2p } from './node';

test('creates node', async (done)=> {
    var node = await p2p.createNode();
    expect(node).toBeTruthy();
    node.start(()=> {
        node.pubsub.subscribe("test", (msg:any) => {
            console.log(msg)
        }, ()=> {
            console.log("subscribe callback?");
            node.pubsub.publish("test", Buffer.from("hi"), (cb:any) => {console.log("callback: ", cb)});
    
            done();
        });
    })
    
    // 
})