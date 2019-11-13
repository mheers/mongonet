package mongonet

import (
	"encoding/json"

	"gopkg.in/mgo.v2/bson"
)

func (m *InsertMessage) HasResponse() bool {
	return false
}

type insertMessageJSON struct {
	TypeName  string
	Header    MessageHeader
	Namespace string
	Docs      []bson.M
	Flags     int32
}

func (m *InsertMessage) ToString() string {
	var docs []bson.M
	for _, doc := range m.Docs {
		bsond, _ := doc.ToBSOND()
		docs = append(docs, bsond.Map())
	}
	cmj := &insertMessageJSON{
		TypeName:  "InsertMessage",
		Header:    m.header,
		Namespace: m.Namespace,
		Docs:      docs,
		Flags:     m.Flags,
	}

	result, _ := json.Marshal(cmj)
	return string(result)
}

func (m *InsertMessage) Header() MessageHeader {
	return m.header
}

func (m *InsertMessage) Serialize() []byte {
	size := 16 /* header */ + 4 /* update header */
	size += len(m.Namespace) + 1
	for _, d := range m.Docs {
		size += int(d.Size)
	}

	m.header.Size = int32(size)

	buf := make([]byte, size)
	m.header.WriteInto(buf)

	loc := 16

	writeInt32(m.Flags, buf, loc)
	loc += 4

	writeCString(m.Namespace, buf, &loc)

	for _, d := range m.Docs {
		d.Copy(&loc, buf)
	}

	return buf
}

func parseInsertMessage(header MessageHeader, buf []byte) (Message, error) {
	m := &InsertMessage{}
	m.header = header

	var err error
	loc := 0

	if len(buf) < 4 {
		return m, NewStackErrorf("invalid insert message -- message must have length of at least 4 bytes.")
	}

	m.Flags = readInt32(buf[loc:])
	loc += 4

	m.Namespace, err = readCString(buf[loc:])
	if err != nil {
		return m, err
	}
	loc += len(m.Namespace) + 1

	for loc < len(buf) {
		doc, err := parseSimpleBSON(buf[loc:])
		if err != nil {
			return nil, err
		}
		m.Docs = append(m.Docs, doc)
		loc += int(doc.Size)
	}

	return m, nil
}

func NewInsertMessage(namespace string, docs ...SimpleBSON) *InsertMessage {
	im := &InsertMessage{}

	im.header.RequestID = 17 // TODO
	im.header.ResponseTo = 0
	im.header.OpCode = OP_INSERT

	im.Flags = 0
	im.Namespace = namespace

	im.Docs = docs

	return im
}
