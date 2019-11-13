package mongonet

import (
	"encoding/json"

	"gopkg.in/mgo.v2/bson"
)

func (m *CommandReplyMessage) HasResponse() bool {
	return false // because its a response
}

type commandReplyMessageJSON struct {
	TypeName     string
	Header       MessageHeader
	OutputDocs   []bson.M
	CommandReply bson.M
}

func (m *CommandReplyMessage) ToString() string {
	commandReply, _ := m.CommandReply.ToBSOND()

	var outputDocs []bson.M
	for _, doc := range m.OutputDocs {
		bsond, _ := doc.ToBSOND()
		outputDocs = append(outputDocs, bsond.Map())
	}

	cmj := &commandReplyMessageJSON{
		TypeName:     "CommandReplyMessage",
		Header:       m.header,
		OutputDocs:   outputDocs,
		CommandReply: commandReply.Map(),
	}

	result, _ := json.Marshal(cmj)
	return string(result)
}

func (m *CommandReplyMessage) Header() MessageHeader {
	return m.header
}

func (m *CommandReplyMessage) Serialize() []byte {
	size := 16 /* header */
	size += int(m.CommandReply.Size)
	size += int(m.Metadata.Size)
	for _, d := range m.OutputDocs {
		size += int(d.Size)
	}
	m.header.Size = int32(size)

	buf := make([]byte, size)
	m.header.WriteInto(buf)

	loc := 16

	m.CommandReply.Copy(&loc, buf)
	m.Metadata.Copy(&loc, buf)

	for _, d := range m.OutputDocs {
		d.Copy(&loc, buf)
	}

	return buf
}

func parseCommandReplyMessage(header MessageHeader, buf []byte) (Message, error) {

	rm := &CommandReplyMessage{}
	rm.header = header

	var err error

	rm.CommandReply, err = parseSimpleBSON(buf)
	if err != nil {
		return rm, err
	}
	if len(buf) < int(rm.CommandReply.Size) {
		return rm, NewStackErrorf("invalid command message -- message length is too short.")
	}
	buf = buf[rm.CommandReply.Size:]

	rm.Metadata, err = parseSimpleBSON(buf)
	if err != nil {
		return rm, err
	}
	if len(buf) < int(rm.Metadata.Size) {
		return rm, NewStackErrorf("invalid command message -- message length is too short.")
	}
	buf = buf[rm.Metadata.Size:]

	for len(buf) > 0 {
		doc, err := parseSimpleBSON(buf)
		if err != nil {
			return rm, err
		}
		buf = buf[doc.Size:]
		rm.OutputDocs = append(rm.OutputDocs, doc)
	}

	return rm, nil
}
