package messages

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *ActorPID) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Address":
			z.Address, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Address")
				return
			}
		case "Id":
			z.Id, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Id")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z ActorPID) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "Address"
	err = en.Append(0x82, 0xa7, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73)
	if err != nil {
		return
	}
	err = en.WriteString(z.Address)
	if err != nil {
		err = msgp.WrapError(err, "Address")
		return
	}
	// write "Id"
	err = en.Append(0xa2, 0x49, 0x64)
	if err != nil {
		return
	}
	err = en.WriteString(z.Id)
	if err != nil {
		err = msgp.WrapError(err, "Id")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z ActorPID) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "Address"
	o = append(o, 0x82, 0xa7, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73)
	o = msgp.AppendString(o, z.Address)
	// string "Id"
	o = append(o, 0xa2, 0x49, 0x64)
	o = msgp.AppendString(o, z.Id)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *ActorPID) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Address":
			z.Address, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Address")
				return
			}
		case "Id":
			z.Id, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Id")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z ActorPID) Msgsize() (s int) {
	s = 1 + 8 + msgp.StringPrefixSize + len(z.Address) + 3 + msgp.StringPrefixSize + len(z.Id)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *CurrentState) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Signature":
			if dc.IsNil() {
				err = dc.ReadNil()
				if err != nil {
					err = msgp.WrapError(err, "Signature")
					return
				}
				z.Signature = nil
			} else {
				if z.Signature == nil {
					z.Signature = new(Signature)
				}
				err = z.Signature.DecodeMsg(dc)
				if err != nil {
					err = msgp.WrapError(err, "Signature")
					return
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *CurrentState) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "Signature"
	err = en.Append(0x81, 0xa9, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65)
	if err != nil {
		return
	}
	if z.Signature == nil {
		err = en.WriteNil()
		if err != nil {
			return
		}
	} else {
		err = z.Signature.EncodeMsg(en)
		if err != nil {
			err = msgp.WrapError(err, "Signature")
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *CurrentState) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Signature"
	o = append(o, 0x81, 0xa9, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65)
	if z.Signature == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.Signature.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "Signature")
			return
		}
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *CurrentState) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Signature":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.Signature = nil
			} else {
				if z.Signature == nil {
					z.Signature = new(Signature)
				}
				bts, err = z.Signature.UnmarshalMsg(bts)
				if err != nil {
					err = msgp.WrapError(err, "Signature")
					return
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *CurrentState) Msgsize() (s int) {
	s = 1 + 10
	if z.Signature == nil {
		s += msgp.NilSize
	} else {
		s += z.Signature.Msgsize()
	}
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Error) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Source":
			z.Source, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Source")
				return
			}
		case "Code":
			z.Code, err = dc.ReadInt()
			if err != nil {
				err = msgp.WrapError(err, "Code")
				return
			}
		case "Memo":
			z.Memo, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Memo")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z Error) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "Source"
	err = en.Append(0x83, 0xa6, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65)
	if err != nil {
		return
	}
	err = en.WriteString(z.Source)
	if err != nil {
		err = msgp.WrapError(err, "Source")
		return
	}
	// write "Code"
	err = en.Append(0xa4, 0x43, 0x6f, 0x64, 0x65)
	if err != nil {
		return
	}
	err = en.WriteInt(z.Code)
	if err != nil {
		err = msgp.WrapError(err, "Code")
		return
	}
	// write "Memo"
	err = en.Append(0xa4, 0x4d, 0x65, 0x6d, 0x6f)
	if err != nil {
		return
	}
	err = en.WriteString(z.Memo)
	if err != nil {
		err = msgp.WrapError(err, "Memo")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Error) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "Source"
	o = append(o, 0x83, 0xa6, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65)
	o = msgp.AppendString(o, z.Source)
	// string "Code"
	o = append(o, 0xa4, 0x43, 0x6f, 0x64, 0x65)
	o = msgp.AppendInt(o, z.Code)
	// string "Memo"
	o = append(o, 0xa4, 0x4d, 0x65, 0x6d, 0x6f)
	o = msgp.AppendString(o, z.Memo)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Error) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Source":
			z.Source, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Source")
				return
			}
		case "Code":
			z.Code, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Code")
				return
			}
		case "Memo":
			z.Memo, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Memo")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Error) Msgsize() (s int) {
	s = 1 + 7 + msgp.StringPrefixSize + len(z.Source) + 5 + msgp.IntSize + 5 + msgp.StringPrefixSize + len(z.Memo)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *GetTip) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "ObjectID":
			z.ObjectID, err = dc.ReadBytes(z.ObjectID)
			if err != nil {
				err = msgp.WrapError(err, "ObjectID")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *GetTip) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "ObjectID"
	err = en.Append(0x81, 0xa8, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x49, 0x44)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.ObjectID)
	if err != nil {
		err = msgp.WrapError(err, "ObjectID")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *GetTip) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "ObjectID"
	o = append(o, 0x81, 0xa8, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x49, 0x44)
	o = msgp.AppendBytes(o, z.ObjectID)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *GetTip) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "ObjectID":
			z.ObjectID, bts, err = msgp.ReadBytesBytes(bts, z.ObjectID)
			if err != nil {
				err = msgp.WrapError(err, "ObjectID")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *GetTip) Msgsize() (s int) {
	s = 1 + 9 + msgp.BytesPrefixSize + len(z.ObjectID)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Ping) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Msg":
			z.Msg, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Msg")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z Ping) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "Msg"
	err = en.Append(0x81, 0xa3, 0x4d, 0x73, 0x67)
	if err != nil {
		return
	}
	err = en.WriteString(z.Msg)
	if err != nil {
		err = msgp.WrapError(err, "Msg")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Ping) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Msg"
	o = append(o, 0x81, 0xa3, 0x4d, 0x73, 0x67)
	o = msgp.AppendString(o, z.Msg)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Ping) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Msg":
			z.Msg, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Msg")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Ping) Msgsize() (s int) {
	s = 1 + 4 + msgp.StringPrefixSize + len(z.Msg)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Pong) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Msg":
			z.Msg, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Msg")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z Pong) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 1
	// write "Msg"
	err = en.Append(0x81, 0xa3, 0x4d, 0x73, 0x67)
	if err != nil {
		return
	}
	err = en.WriteString(z.Msg)
	if err != nil {
		err = msgp.WrapError(err, "Msg")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Pong) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Msg"
	o = append(o, 0x81, 0xa3, 0x4d, 0x73, 0x67)
	o = msgp.AppendString(o, z.Msg)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Pong) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Msg":
			z.Msg, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Msg")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Pong) Msgsize() (s int) {
	s = 1 + 4 + msgp.StringPrefixSize + len(z.Msg)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Signature) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "TransactionID":
			z.TransactionID, err = dc.ReadBytes(z.TransactionID)
			if err != nil {
				err = msgp.WrapError(err, "TransactionID")
				return
			}
		case "ObjectID":
			z.ObjectID, err = dc.ReadBytes(z.ObjectID)
			if err != nil {
				err = msgp.WrapError(err, "ObjectID")
				return
			}
		case "PreviousTip":
			z.PreviousTip, err = dc.ReadBytes(z.PreviousTip)
			if err != nil {
				err = msgp.WrapError(err, "PreviousTip")
				return
			}
		case "Height":
			z.Height, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "Height")
				return
			}
		case "NewTip":
			z.NewTip, err = dc.ReadBytes(z.NewTip)
			if err != nil {
				err = msgp.WrapError(err, "NewTip")
				return
			}
		case "View":
			z.View, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "View")
				return
			}
		case "Cycle":
			z.Cycle, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "Cycle")
				return
			}
		case "Signers":
			z.Signers, err = dc.ReadBytes(z.Signers)
			if err != nil {
				err = msgp.WrapError(err, "Signers")
				return
			}
		case "Signature":
			z.Signature, err = dc.ReadBytes(z.Signature)
			if err != nil {
				err = msgp.WrapError(err, "Signature")
				return
			}
		case "Type":
			z.Type, err = dc.ReadString()
			if err != nil {
				err = msgp.WrapError(err, "Type")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Signature) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 10
	// write "TransactionID"
	err = en.Append(0x8a, 0xad, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x44)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.TransactionID)
	if err != nil {
		err = msgp.WrapError(err, "TransactionID")
		return
	}
	// write "ObjectID"
	err = en.Append(0xa8, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x49, 0x44)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.ObjectID)
	if err != nil {
		err = msgp.WrapError(err, "ObjectID")
		return
	}
	// write "PreviousTip"
	err = en.Append(0xab, 0x50, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x54, 0x69, 0x70)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.PreviousTip)
	if err != nil {
		err = msgp.WrapError(err, "PreviousTip")
		return
	}
	// write "Height"
	err = en.Append(0xa6, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.Height)
	if err != nil {
		err = msgp.WrapError(err, "Height")
		return
	}
	// write "NewTip"
	err = en.Append(0xa6, 0x4e, 0x65, 0x77, 0x54, 0x69, 0x70)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.NewTip)
	if err != nil {
		err = msgp.WrapError(err, "NewTip")
		return
	}
	// write "View"
	err = en.Append(0xa4, 0x56, 0x69, 0x65, 0x77)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.View)
	if err != nil {
		err = msgp.WrapError(err, "View")
		return
	}
	// write "Cycle"
	err = en.Append(0xa5, 0x43, 0x79, 0x63, 0x6c, 0x65)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.Cycle)
	if err != nil {
		err = msgp.WrapError(err, "Cycle")
		return
	}
	// write "Signers"
	err = en.Append(0xa7, 0x53, 0x69, 0x67, 0x6e, 0x65, 0x72, 0x73)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.Signers)
	if err != nil {
		err = msgp.WrapError(err, "Signers")
		return
	}
	// write "Signature"
	err = en.Append(0xa9, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.Signature)
	if err != nil {
		err = msgp.WrapError(err, "Signature")
		return
	}
	// write "Type"
	err = en.Append(0xa4, 0x54, 0x79, 0x70, 0x65)
	if err != nil {
		return
	}
	err = en.WriteString(z.Type)
	if err != nil {
		err = msgp.WrapError(err, "Type")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Signature) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 10
	// string "TransactionID"
	o = append(o, 0x8a, 0xad, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x44)
	o = msgp.AppendBytes(o, z.TransactionID)
	// string "ObjectID"
	o = append(o, 0xa8, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x49, 0x44)
	o = msgp.AppendBytes(o, z.ObjectID)
	// string "PreviousTip"
	o = append(o, 0xab, 0x50, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x54, 0x69, 0x70)
	o = msgp.AppendBytes(o, z.PreviousTip)
	// string "Height"
	o = append(o, 0xa6, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74)
	o = msgp.AppendUint64(o, z.Height)
	// string "NewTip"
	o = append(o, 0xa6, 0x4e, 0x65, 0x77, 0x54, 0x69, 0x70)
	o = msgp.AppendBytes(o, z.NewTip)
	// string "View"
	o = append(o, 0xa4, 0x56, 0x69, 0x65, 0x77)
	o = msgp.AppendUint64(o, z.View)
	// string "Cycle"
	o = append(o, 0xa5, 0x43, 0x79, 0x63, 0x6c, 0x65)
	o = msgp.AppendUint64(o, z.Cycle)
	// string "Signers"
	o = append(o, 0xa7, 0x53, 0x69, 0x67, 0x6e, 0x65, 0x72, 0x73)
	o = msgp.AppendBytes(o, z.Signers)
	// string "Signature"
	o = append(o, 0xa9, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65)
	o = msgp.AppendBytes(o, z.Signature)
	// string "Type"
	o = append(o, 0xa4, 0x54, 0x79, 0x70, 0x65)
	o = msgp.AppendString(o, z.Type)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Signature) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "TransactionID":
			z.TransactionID, bts, err = msgp.ReadBytesBytes(bts, z.TransactionID)
			if err != nil {
				err = msgp.WrapError(err, "TransactionID")
				return
			}
		case "ObjectID":
			z.ObjectID, bts, err = msgp.ReadBytesBytes(bts, z.ObjectID)
			if err != nil {
				err = msgp.WrapError(err, "ObjectID")
				return
			}
		case "PreviousTip":
			z.PreviousTip, bts, err = msgp.ReadBytesBytes(bts, z.PreviousTip)
			if err != nil {
				err = msgp.WrapError(err, "PreviousTip")
				return
			}
		case "Height":
			z.Height, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Height")
				return
			}
		case "NewTip":
			z.NewTip, bts, err = msgp.ReadBytesBytes(bts, z.NewTip)
			if err != nil {
				err = msgp.WrapError(err, "NewTip")
				return
			}
		case "View":
			z.View, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "View")
				return
			}
		case "Cycle":
			z.Cycle, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Cycle")
				return
			}
		case "Signers":
			z.Signers, bts, err = msgp.ReadBytesBytes(bts, z.Signers)
			if err != nil {
				err = msgp.WrapError(err, "Signers")
				return
			}
		case "Signature":
			z.Signature, bts, err = msgp.ReadBytesBytes(bts, z.Signature)
			if err != nil {
				err = msgp.WrapError(err, "Signature")
				return
			}
		case "Type":
			z.Type, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Type")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Signature) Msgsize() (s int) {
	s = 1 + 14 + msgp.BytesPrefixSize + len(z.TransactionID) + 9 + msgp.BytesPrefixSize + len(z.ObjectID) + 12 + msgp.BytesPrefixSize + len(z.PreviousTip) + 7 + msgp.Uint64Size + 7 + msgp.BytesPrefixSize + len(z.NewTip) + 5 + msgp.Uint64Size + 6 + msgp.Uint64Size + 8 + msgp.BytesPrefixSize + len(z.Signers) + 10 + msgp.BytesPrefixSize + len(z.Signature) + 5 + msgp.StringPrefixSize + len(z.Type)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Store) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Key":
			z.Key, err = dc.ReadBytes(z.Key)
			if err != nil {
				err = msgp.WrapError(err, "Key")
				return
			}
		case "Value":
			z.Value, err = dc.ReadBytes(z.Value)
			if err != nil {
				err = msgp.WrapError(err, "Value")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Store) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 2
	// write "Key"
	err = en.Append(0x82, 0xa3, 0x4b, 0x65, 0x79)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.Key)
	if err != nil {
		err = msgp.WrapError(err, "Key")
		return
	}
	// write "Value"
	err = en.Append(0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.Value)
	if err != nil {
		err = msgp.WrapError(err, "Value")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Store) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "Key"
	o = append(o, 0x82, 0xa3, 0x4b, 0x65, 0x79)
	o = msgp.AppendBytes(o, z.Key)
	// string "Value"
	o = append(o, 0xa5, 0x56, 0x61, 0x6c, 0x75, 0x65)
	o = msgp.AppendBytes(o, z.Value)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Store) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Key":
			z.Key, bts, err = msgp.ReadBytesBytes(bts, z.Key)
			if err != nil {
				err = msgp.WrapError(err, "Key")
				return
			}
		case "Value":
			z.Value, bts, err = msgp.ReadBytesBytes(bts, z.Value)
			if err != nil {
				err = msgp.WrapError(err, "Value")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Store) Msgsize() (s int) {
	s = 1 + 4 + msgp.BytesPrefixSize + len(z.Key) + 6 + msgp.BytesPrefixSize + len(z.Value)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *TipSubscription) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Unsubscribe":
			z.Unsubscribe, err = dc.ReadBool()
			if err != nil {
				err = msgp.WrapError(err, "Unsubscribe")
				return
			}
		case "ObjectID":
			z.ObjectID, err = dc.ReadBytes(z.ObjectID)
			if err != nil {
				err = msgp.WrapError(err, "ObjectID")
				return
			}
		case "TipValue":
			z.TipValue, err = dc.ReadBytes(z.TipValue)
			if err != nil {
				err = msgp.WrapError(err, "TipValue")
				return
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *TipSubscription) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 3
	// write "Unsubscribe"
	err = en.Append(0x83, 0xab, 0x55, 0x6e, 0x73, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65)
	if err != nil {
		return
	}
	err = en.WriteBool(z.Unsubscribe)
	if err != nil {
		err = msgp.WrapError(err, "Unsubscribe")
		return
	}
	// write "ObjectID"
	err = en.Append(0xa8, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x49, 0x44)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.ObjectID)
	if err != nil {
		err = msgp.WrapError(err, "ObjectID")
		return
	}
	// write "TipValue"
	err = en.Append(0xa8, 0x54, 0x69, 0x70, 0x56, 0x61, 0x6c, 0x75, 0x65)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.TipValue)
	if err != nil {
		err = msgp.WrapError(err, "TipValue")
		return
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *TipSubscription) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "Unsubscribe"
	o = append(o, 0x83, 0xab, 0x55, 0x6e, 0x73, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x62, 0x65)
	o = msgp.AppendBool(o, z.Unsubscribe)
	// string "ObjectID"
	o = append(o, 0xa8, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x49, 0x44)
	o = msgp.AppendBytes(o, z.ObjectID)
	// string "TipValue"
	o = append(o, 0xa8, 0x54, 0x69, 0x70, 0x56, 0x61, 0x6c, 0x75, 0x65)
	o = msgp.AppendBytes(o, z.TipValue)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *TipSubscription) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "Unsubscribe":
			z.Unsubscribe, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Unsubscribe")
				return
			}
		case "ObjectID":
			z.ObjectID, bts, err = msgp.ReadBytesBytes(bts, z.ObjectID)
			if err != nil {
				err = msgp.WrapError(err, "ObjectID")
				return
			}
		case "TipValue":
			z.TipValue, bts, err = msgp.ReadBytesBytes(bts, z.TipValue)
			if err != nil {
				err = msgp.WrapError(err, "TipValue")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *TipSubscription) Msgsize() (s int) {
	s = 1 + 12 + msgp.BoolSize + 9 + msgp.BytesPrefixSize + len(z.ObjectID) + 9 + msgp.BytesPrefixSize + len(z.TipValue)
	return
}

// DecodeMsg implements msgp.Decodable
func (z *Transaction) DecodeMsg(dc *msgp.Reader) (err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, err = dc.ReadMapKeyPtr()
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "ObjectID":
			z.ObjectID, err = dc.ReadBytes(z.ObjectID)
			if err != nil {
				err = msgp.WrapError(err, "ObjectID")
				return
			}
		case "PreviousTip":
			z.PreviousTip, err = dc.ReadBytes(z.PreviousTip)
			if err != nil {
				err = msgp.WrapError(err, "PreviousTip")
				return
			}
		case "Height":
			z.Height, err = dc.ReadUint64()
			if err != nil {
				err = msgp.WrapError(err, "Height")
				return
			}
		case "NewTip":
			z.NewTip, err = dc.ReadBytes(z.NewTip)
			if err != nil {
				err = msgp.WrapError(err, "NewTip")
				return
			}
		case "Payload":
			z.Payload, err = dc.ReadBytes(z.Payload)
			if err != nil {
				err = msgp.WrapError(err, "Payload")
				return
			}
		case "State":
			var zb0002 uint32
			zb0002, err = dc.ReadArrayHeader()
			if err != nil {
				err = msgp.WrapError(err, "State")
				return
			}
			if cap(z.State) >= int(zb0002) {
				z.State = (z.State)[:zb0002]
			} else {
				z.State = make([][]byte, zb0002)
			}
			for za0001 := range z.State {
				z.State[za0001], err = dc.ReadBytes(z.State[za0001])
				if err != nil {
					err = msgp.WrapError(err, "State", za0001)
					return
				}
			}
		default:
			err = dc.Skip()
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *Transaction) EncodeMsg(en *msgp.Writer) (err error) {
	// map header, size 6
	// write "ObjectID"
	err = en.Append(0x86, 0xa8, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x49, 0x44)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.ObjectID)
	if err != nil {
		err = msgp.WrapError(err, "ObjectID")
		return
	}
	// write "PreviousTip"
	err = en.Append(0xab, 0x50, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x54, 0x69, 0x70)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.PreviousTip)
	if err != nil {
		err = msgp.WrapError(err, "PreviousTip")
		return
	}
	// write "Height"
	err = en.Append(0xa6, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74)
	if err != nil {
		return
	}
	err = en.WriteUint64(z.Height)
	if err != nil {
		err = msgp.WrapError(err, "Height")
		return
	}
	// write "NewTip"
	err = en.Append(0xa6, 0x4e, 0x65, 0x77, 0x54, 0x69, 0x70)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.NewTip)
	if err != nil {
		err = msgp.WrapError(err, "NewTip")
		return
	}
	// write "Payload"
	err = en.Append(0xa7, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64)
	if err != nil {
		return
	}
	err = en.WriteBytes(z.Payload)
	if err != nil {
		err = msgp.WrapError(err, "Payload")
		return
	}
	// write "State"
	err = en.Append(0xa5, 0x53, 0x74, 0x61, 0x74, 0x65)
	if err != nil {
		return
	}
	err = en.WriteArrayHeader(uint32(len(z.State)))
	if err != nil {
		err = msgp.WrapError(err, "State")
		return
	}
	for za0001 := range z.State {
		err = en.WriteBytes(z.State[za0001])
		if err != nil {
			err = msgp.WrapError(err, "State", za0001)
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *Transaction) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "ObjectID"
	o = append(o, 0x86, 0xa8, 0x4f, 0x62, 0x6a, 0x65, 0x63, 0x74, 0x49, 0x44)
	o = msgp.AppendBytes(o, z.ObjectID)
	// string "PreviousTip"
	o = append(o, 0xab, 0x50, 0x72, 0x65, 0x76, 0x69, 0x6f, 0x75, 0x73, 0x54, 0x69, 0x70)
	o = msgp.AppendBytes(o, z.PreviousTip)
	// string "Height"
	o = append(o, 0xa6, 0x48, 0x65, 0x69, 0x67, 0x68, 0x74)
	o = msgp.AppendUint64(o, z.Height)
	// string "NewTip"
	o = append(o, 0xa6, 0x4e, 0x65, 0x77, 0x54, 0x69, 0x70)
	o = msgp.AppendBytes(o, z.NewTip)
	// string "Payload"
	o = append(o, 0xa7, 0x50, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64)
	o = msgp.AppendBytes(o, z.Payload)
	// string "State"
	o = append(o, 0xa5, 0x53, 0x74, 0x61, 0x74, 0x65)
	o = msgp.AppendArrayHeader(o, uint32(len(z.State)))
	for za0001 := range z.State {
		o = msgp.AppendBytes(o, z.State[za0001])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Transaction) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "ObjectID":
			z.ObjectID, bts, err = msgp.ReadBytesBytes(bts, z.ObjectID)
			if err != nil {
				err = msgp.WrapError(err, "ObjectID")
				return
			}
		case "PreviousTip":
			z.PreviousTip, bts, err = msgp.ReadBytesBytes(bts, z.PreviousTip)
			if err != nil {
				err = msgp.WrapError(err, "PreviousTip")
				return
			}
		case "Height":
			z.Height, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Height")
				return
			}
		case "NewTip":
			z.NewTip, bts, err = msgp.ReadBytesBytes(bts, z.NewTip)
			if err != nil {
				err = msgp.WrapError(err, "NewTip")
				return
			}
		case "Payload":
			z.Payload, bts, err = msgp.ReadBytesBytes(bts, z.Payload)
			if err != nil {
				err = msgp.WrapError(err, "Payload")
				return
			}
		case "State":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "State")
				return
			}
			if cap(z.State) >= int(zb0002) {
				z.State = (z.State)[:zb0002]
			} else {
				z.State = make([][]byte, zb0002)
			}
			for za0001 := range z.State {
				z.State[za0001], bts, err = msgp.ReadBytesBytes(bts, z.State[za0001])
				if err != nil {
					err = msgp.WrapError(err, "State", za0001)
					return
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *Transaction) Msgsize() (s int) {
	s = 1 + 9 + msgp.BytesPrefixSize + len(z.ObjectID) + 12 + msgp.BytesPrefixSize + len(z.PreviousTip) + 7 + msgp.Uint64Size + 7 + msgp.BytesPrefixSize + len(z.NewTip) + 8 + msgp.BytesPrefixSize + len(z.Payload) + 6 + msgp.ArrayHeaderSize
	for za0001 := range z.State {
		s += msgp.BytesPrefixSize + len(z.State[za0001])
	}
	return
}
