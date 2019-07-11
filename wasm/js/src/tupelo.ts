declare const Go: any;

import * as go from "./js/go"

export class FakePublisher {
    public publish(topic:string, data:Uint8Array, cb:Function) {
        console.log("publishing ", data, " on ", topic, "cb ", cb)
        cb(null)
    }
}

class UnderlyingWasm {
    _populated: boolean;

    constructor() {
        this._populated = false;
    }
    publish(publisher:FakePublisher) {
        return new Error("this should be swapped out by wasm")
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

    export async function publish(topic:string, data:Uint8Array):Promise<any> {
        const tw = await TupeloWasm.get()
        return tw.publish(new FakePublisher())
    }
}

export default Tupelo