import * as libp2p from './js/p2p';

interface INodeCreator {
    pubsub:any;
    start(cb:Function):null;
}

export namespace p2p {
    export async function createNode():Promise<INodeCreator> {
        return libp2p.CreateNode()
    }
}