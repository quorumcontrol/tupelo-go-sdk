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
    testpubsub(publisher:IPubSub):Promise<String>{
        return new Promise<String>((res,rej)=> {}); // replaced by wasm
    }
    generateKey():Promise<Uint8Array> {
        return new Promise<Uint8Array>((res,rej)=> {}) // replaced by wasm
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

   export async function generateKey():Promise<Uint8Array> {
       const tw = await TupeloWasm.get()
       return tw.generateKey()
   }
}

export default Tupelo