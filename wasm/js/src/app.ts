declare const Go: any;

import * as tupelo from './js/go'

export async function run(path:string):Promise<any> {
    return tupelo.run(path);
}

export async function ready():Promise<any> {
    return tupelo.ready();
}
