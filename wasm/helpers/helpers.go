package helpers

import (
	"github.com/ipfs/go-cid"
	"syscall/js"
)

func JsStringArrayToStringSlice(jsStrArray js.Value) []string {
	len := jsStrArray.Length()
	strs := make([]string, len)
	for i := 0; i < len; i++ {
		strs[i] = jsStrArray.Index(i).String()
	}
	return strs
}

func JsBufferToBytes(buf js.Value) []byte {
	len := buf.Length()
	bits := make([]byte, len)
	for i := 0; i < len; i++ {
		bits[i] = byte(uint8(buf.Index(i).Int()))
	}
	return bits
}

func SliceToJSBuffer(slice []byte) js.Value {
	return js.Global().Get("Buffer").Call("from", js.TypedArrayOf(slice))
}

func CidToJSCID(c cid.Cid) js.Value {
	jsCids := js.Global().Call("require", js.ValueOf("cids"))
	bits := c.Bytes()
	jsBits := SliceToJSBuffer(bits)
	return jsCids.New(jsBits)
}