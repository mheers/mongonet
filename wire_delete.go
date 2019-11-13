package mongonet

import (
	"encoding/json"

	"gopkg.in/mgo.v2/bson"
)

func (m *DeleteMessage) HasResponse() bool {
	return false
}

type deleteMessageJSON struct {
	TypeName  string
	Header    MessageHeader
	Namespace string
	Reserved  int32
	Flags     int32
	Filter    bson.M
}

func (m *DeleteMessage) ToString() string {

	filter, _ := m.Filter.ToBSOND()

	cmj := &deleteMessageJSON{
		TypeName:  "DeleteMessage",
		Header:    m.header,
		Namespace: m.Namespace,
		Reserved:  m.Reserved,
		Flags:     m.Flags,
		Filter:    filter.Map(),
	}

	result, _ := json.Marshal(cmj)
	return string(result)
}

func (m *DeleteMessage) Header() MessageHeader {
	return m.header
}

func (m *DeleteMessage) Serialize() []byte {
	size := 16 /* header */ + 8 /* update header */
	size += len(m.Namespace) + 1
	size += int(m.Filter.Size)

	m.header.Size = int32(size)

	buf := make([]byte, size)
	m.header.WriteInto(buf)

	loc := 16

	writeInt32(0, buf, loc)
	loc += 4

	writeCString(m.Namespace, buf, &loc)

	writeInt32(m.Flags, buf, loc)
	loc += 4

	m.Filter.Copy(&loc, buf)

	return buf
}

func parseDeleteMessage(header MessageHeader, buf []byte) (Message, error) {
	m := &DeleteMessage{}
	m.header = header

	var err error
	loc := 0

	if len(buf) < 4 {
		return m, NewStackErrorf("invalid delete message -- message must have length of at least 4 bytes.")
	}
	m.Reserved = readInt32(buf[loc:])
	loc += 4

	m.Namespace, err = readCString(buf[loc:])
	if err != nil {
		return m, err
	}
	loc += len(m.Namespace) + 1

	if len(buf) < loc+4 {
		return m, NewStackErrorf("invalid delete message -- message length is too short.")
	}
	m.Flags = readInt32(buf[loc:])
	loc += 4

	m.Filter, err = parseSimpleBSON(buf[loc:])
	if err != nil {
		return m, err
	}
	loc += int(m.Filter.Size)

	return m, nil
}
