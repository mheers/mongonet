package mongonet

import (
	"encoding/json"

	"gopkg.in/mgo.v2/bson"
)

func (m *UpdateMessage) HasResponse() bool {
	return false
}

type updateMessageJSON struct {
	TypeName  string
	Header    MessageHeader
	Flags     int32
	Reserved  int32
	Namespace string
	Filter    bson.M
	Update    bson.M
}

func (m *UpdateMessage) ToString() string {
	filter, _ := m.Filter.ToBSOND()
	update, _ := m.Update.ToBSOND()

	cmj := &updateMessageJSON{
		TypeName:  "UpdateMessage",
		Header:    m.header,
		Flags:     m.Flags,
		Reserved:  m.Reserved,
		Namespace: m.Namespace,
		Filter:    filter.Map(),
		Update:    update.Map(),
	}

	result, _ := json.Marshal(cmj)
	return string(result)
}

func (m *UpdateMessage) Header() MessageHeader {
	return m.header
}

func (m *UpdateMessage) Serialize() []byte {
	size := 16 /* header */ + 8 /* update header */
	size += len(m.Namespace) + 1
	size += int(m.Filter.Size)
	size += int(m.Update.Size)

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
	m.Update.Copy(&loc, buf)

	return buf
}

func parseUpdateMessage(header MessageHeader, buf []byte) (Message, error) {
	m := &UpdateMessage{}
	m.header = header

	var err error
	loc := 0

	if len(buf) < 4 {
		return m, NewStackErrorf("invalid update message -- message must have length of at least 4 bytes.")
	}
	m.Reserved = readInt32(buf[loc:])
	loc += 4

	m.Namespace, err = readCString(buf[loc:])
	if err != nil {
		return m, err
	}
	loc += len(m.Namespace) + 1

	if len(buf) < (loc + 4) {
		return m, NewStackErrorf("invalid update message -- message length is too short.")
	}
	m.Flags = readInt32(buf[loc:])
	loc += 4

	m.Filter, err = parseSimpleBSON(buf[loc:])
	if err != nil {
		return m, err
	}
	loc += int(m.Filter.Size)

	if len(buf) < loc {
		return m, NewStackErrorf("invalid update message -- message length is too short.")
	}
	m.Update, err = parseSimpleBSON(buf[loc:])
	if err != nil {
		return m, err
	}
	loc += int(m.Filter.Size)

	return m, nil
}
