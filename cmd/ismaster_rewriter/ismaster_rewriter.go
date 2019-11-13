package main

import (
	"flag"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/erh/mongonet"
	"github.com/mheers/mgosniff/mgosniff"
	"gopkg.in/mgo.v2/bson"
)

const (
	errorCode     = 20000
	errorCodeName = "SNITesterError"
)

type MyFactory struct {
}

func (myf *MyFactory) NewInterceptor(ps *mongonet.ProxySession) (mongonet.ProxyInterceptor, error) {
	return &MyInterceptor{ps}, nil
}

type MyInterceptor struct {
	ps *mongonet.ProxySession
}

func (myi *MyInterceptor) sniResponse() mongonet.SimpleBSON {
	doc := bson.D{{"sniName", "myi.ps.SSLServerName"}, {"ok", 1}}
	raw, err := mongonet.SimpleBSONConvert(doc)
	if err != nil {
		panic(err)
	}
	return raw
}

func (myi *MyInterceptor) isMasterResponse() mongonet.SimpleBSON {
	// doc := bson.D{
	// 	{"msg", "isdbgrid"},
	// 	{"maxBsonObjectSize", 16777216},
	// 	{"maxMessageSizeBytes", 48000000},
	// 	{"maxWriteBatchSize", 100000},
	// 	// {"localTime", new Date()},
	// 	{"maxWireVersion", 7},
	// 	{"minWireVersion", 5},
	// 	{"ok", 1},
	// 	{"readOnly", false},
	// }
	doc := bson.D{
		{"maxWireVersion", 5}, // very important to set this to 5! otherwise it does not work. from wireversion 6 the MessageMessage is used.
		{"minWireVersion", 0},
		{"ok", 1},
	}
	raw, err := mongonet.SimpleBSONConvert(doc)
	if err != nil {
		panic(err)
	}
	return raw
}

func (myi *MyInterceptor) InterceptClientToMongo(m mongonet.Message) (mongonet.Message, mongonet.ResponseInterceptor, error) {
	switch mm := m.(type) {
	case *mongonet.QueryMessage:
		if !mongonet.NamespaceIsCommand(mm.Namespace) {
			log.Println("!mongonet.NamespaceIsCommand")
			return m, nil, nil
		}

		query, err := mm.Query.ToBSOND()
		if err != nil || len(query) == 0 {
			// let mongod handle error message
			log.Println("error or len(query)==0")
			return m, nil, nil
		}

		cmdName := strings.ToLower(query[0].Name)
		log.Println("cmdName:", cmdName)
		switch cmdName {
		case "ismaster":
			err := myi.ps.RespondToCommand(mm, myi.isMasterResponse())
			return nil, nil, err

		case "sni":
			return nil, nil, newSNIError(myi.ps.RespondToCommand(mm, myi.sniResponse()))
		}
		return m, nil, nil

	case *mongonet.CommandMessage:
		cmdName := strings.ToLower(mm.CmdName)
		log.Println("cmdName:", cmdName)
		switch cmdName {
		case "ismaster":
			err := myi.ps.RespondToCommand(mm, myi.isMasterResponse())
			return nil, nil, err

		case "sni":
			return nil, nil, newSNIError(myi.ps.RespondToCommand(mm, myi.sniResponse()))
		}
		return mm, nil, nil

	case *mongonet.MessageMessage:
		log.Println("message message:", mm.Serialize())
		// if mm.CmdName == "sni" {
		// 	return nil, nil, newSNIError(myi.ps.RespondToCommand(mm, myi.sniResponse()))
		// }
		// if mm.CmdName == "isMaster" {
		// 	return mm, nil, nil
		// }
		return m, nil, nil
	}

	log.Println("normal query")
	return m, nil, nil
}

func (myi *MyInterceptor) Close() {
	log.Println("closing")
}
func (myi *MyInterceptor) TrackRequest(mh mongonet.MessageHeader) {
	// log.Println("TrackRequest", mh.Size, mh.RequestID, mh.ResponseTo, mh.OpCode)
}
func (myi *MyInterceptor) TrackResponse(mh mongonet.MessageHeader) {
	// log.Println("TrackResponse", mh.Size, mh.RequestID, mh.ResponseTo, mh.OpCode)
}
func (myi *MyInterceptor) TrackRequestMessage(m mongonet.Message) {
	// log.Println("TrackRequestMessage", m.Serialize())
	typeName := reflect.TypeOf(m).Name()
	log.Println("[REQUEST]", typeName, m.ToString())
}
func (myi *MyInterceptor) TrackResponseMessage(m mongonet.Message) {
	typeName := reflect.TypeOf(m).Name()
	log.Println("[RESPONSE]", typeName, m.ToString())
}

// func MessageToString(m mongonet.Message) string {
// 	switch mm := m.(type) {
// 	case *mongonet.QueryMessage:
// 		return mm.ToString()
// 	}
// 	return strconv.Itoa(int(m.Header().OpCode))
// }

func (myi *MyInterceptor) CheckConnection() error {
	return nil
}

func (myi *MyInterceptor) CheckConnectionInterval() time.Duration {
	return 0
}

func main() {

	bindHost := flag.String("host", "127.0.0.1", "what to bind to")
	bindPort := flag.Int("port", 9999, "what to bind to")
	mongoHost := flag.String("mongoHost", "127.0.0.1", "host mongo is on")
	mongoPort := flag.Int("mongoPort", 27017, "port mongo is on")

	flag.Parse()

	log.Printf(`Starting ismater_rewriter with 
	bindHost: %s
	bindPort: %d
	mongoHost: %s
	mongoPort: %d
	`, *bindHost, *bindPort, *mongoHost, *mongoPort)

	pc := mongonet.NewProxyConfig(*bindHost, *bindPort, *mongoHost, *mongoPort)

	pc.UseSSL = false
	// if len(flag.Args()) < 2 {
	// 	fmt.Printf("need to specify ssl cert and key\n")
	// 	os.Exit(-1)
	// }

	// pc.SSLKeys = []mongonet.SSLPair{
	// 	{flag.Arg(0), flag.Arg(1)},
	// }

	pc.InterceptorFactory = &MyFactory{}

	// pc.MongoSSLSkipVerify = true

	proxy := mongonet.NewProxy(pc)

	proxy.InitializeServer()
	proxy.OnSSLConfig(nil)

	err := proxy.Run()
	if err != nil {
		panic(err)
	}

	// Parser{}
	t := mgosniff.MsgHeader{}
	fmt.Println(t.MessageLength)
	// fmt.Println(MyName)
}

func newSNIError(err error) error {
	if err == nil {
		return nil
	}

	return mongonet.NewMongoError(err, errorCode, errorCodeName)
}
