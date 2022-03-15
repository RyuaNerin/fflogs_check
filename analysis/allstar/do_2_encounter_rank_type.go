package allstar

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
)

type IntV struct {
	V  int
	Ok bool
}
type Float32V struct {
	V  float32
	Ok bool
}

func init() {
	jsoniter.RegisterTypeDecoderFunc(
		"allstar.IntV",
		func(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
			p := ((*IntV)(ptr))

			value := iter.Read()
			if vStr, ok := value.(string); ok {
				if vStr == "-" {
					p.Ok = true
					return
				}
			}

			i, err := cast.ToIntE(value)
			if err != nil {
				iter.Error = err
			} else {
				p.Ok = true
				p.V = i
			}
		},
	)
	jsoniter.RegisterTypeEncoderFunc(
		"allstar.IntV",
		func(ptr unsafe.Pointer, stream *jsoniter.Stream) {
			p := ((*IntV)(ptr))
			stream.WriteInt(p.V)
		},
		func(p unsafe.Pointer) bool {
			return false
		},
	)

	jsoniter.RegisterTypeDecoderFunc(
		"allstar.Float32V",
		func(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
			p := ((*Float32V)(ptr))

			value := iter.Read()
			if vStr, ok := value.(string); ok {
				if vStr == "-" {
					p.Ok = true
					return
				}
			}

			f, err := cast.ToFloat32E(value)
			if err != nil {
				iter.Error = err
			} else {
				p.Ok = true
				p.V = f
			}
		},
	)
	jsoniter.RegisterTypeEncoderFunc(
		"allstar.Float32V",
		func(ptr unsafe.Pointer, stream *jsoniter.Stream) {
			p := ((*Float32V)(ptr))
			stream.WriteFloat32(p.V)
		},
		func(p unsafe.Pointer) bool {
			return false
		},
	)
}
