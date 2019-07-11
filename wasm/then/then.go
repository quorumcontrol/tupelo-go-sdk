// +build wasm

package then

import (
	"fmt"
	"reflect"
	"syscall/js"
)

// then implements the javascript thenable interface https://javascript.info/async-await
// so that go can return something to js and js can `await` it

type Then struct {
	result       interface{}
	err          interface{}
	resolveCbs   []js.Value
	errorCbs     []js.Value
	jsObj        map[string]interface{}
	wrappedJsObj js.Value
}

func New() *Then {
	t := new(Then)
	jsObj := map[string]interface{}{
		"then": js.FuncOf(func(_this js.Value, args []js.Value) interface{} {
			switch len(args) {
			case 1:
				return t.handleJSThenCall(args[0], js.Null())
			case 2:
				return t.handleJSThenCall(args[0], args[1])
			default:
				return fmt.Errorf("error, must specify at least a resolve")
			}
		},
		)}
	wrappedObj := js.ValueOf(jsObj)

	t.jsObj = jsObj
	t.wrappedJsObj = wrappedObj
	return t
}

func (t *Then) handleJSThenCall(resolve js.Value, reject js.Value) error {
	if t.result != nil {
		go resolve.Invoke(js.ValueOf(t.result))
		return nil
	}

	if t.err != nil {
		if reject.Truthy() {
			go reject.Invoke(js.ValueOf(t.err))
		}
		return nil
	}

	t.resolveCbs = append(t.resolveCbs, resolve)
	if reject.Truthy() {
		t.errorCbs = append(t.errorCbs, reject)
	}
	return nil
}

func (t *Then) Resolve(res interface{}) {
	fmt.Println("resolving with: ", reflect.TypeOf(res).String())
	t.result = res
	for _, cb := range t.resolveCbs {
		go func(cb js.Value) {
			cb.Invoke(js.ValueOf(res))
		}(cb)
	}
	t.resolveCbs = nil
}

func (t *Then) Reject(err interface{}) {
	fmt.Println("rejecting with: ", reflect.TypeOf(err).String())
	t.err = err
	for _, cb := range t.errorCbs {
		go func(cb js.Value) {
			cb.Invoke(js.ValueOf(err))
		}(cb)
	}
	t.errorCbs = nil
}

func (t *Then) ToJS() js.Value {
	return t.wrappedJsObj
}
