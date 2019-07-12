declare const Go: any;

import * as go from "./js/go"

class FakePublisher {
    public publish(topic:string, data:Uint8Array, cb:Function) {
        console.log("publishing ", data, " on ", topic, "cb ", cb)
        cb(null)
    }
}

export interface IPubSub {
    publish(topic:string, data:Uint8Array, cb:Function):null
    subscribe(topic:string, onMsg:Function, cb:Function):null
}

class UnderlyingWasm {
    _populated: boolean;

    constructor() {
        this._populated = false;
    }
    testpubsub(publisher:IPubSub) {
        return new Promise((res,rej)=> {});
    }
}

export namespace TupeloWasm {
    const _tupelowasm = new UnderlyingWasm();
    
    export async function get() {
        if (_tupelowasm._populated) {
            return _tupelowasm;
        }
    
        go.run("../main.wasm");
        await go.ready();
        go.populate(_tupelowasm);
        _tupelowasm._populated = true;
        return _tupelowasm;
    }
}

export namespace Tupelo {

   
}

export default Tupelo