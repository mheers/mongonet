package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/erh/mongonet"
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

func (myi *MyInterceptor) InterceptClientToMongo(m mongonet.Message) (mongonet.Message, mongonet.ResponseInterceptor, error) {
	switch mm := m.(type) {
	case *mongonet.QueryMessage:
		if !mongonet.NamespaceIsCommand(mm.Namespace) {
			fmt.Println("!mongonet.NamespaceIsCommand")
			return m, nil, nil
		}

		query, err := mm.Query.ToBSOND()
		if err != nil || len(query) == 0 {
			// let mongod handle error message
			fmt.Println("error or len(query)==0")
			return m, nil, nil
		}

		cmdName := query[0].Name
		fmt.Println("cmdName:", cmdName)
		if strings.ToLower(cmdName) == "ismaster" {
			return m, nil, nil
		}
		if strings.ToLower(cmdName) == "sni" {
			return nil, nil, newSNIError(myi.ps.RespondToCommand(mm, myi.sniResponse()))
		}
		return m, nil, nil

	case *mongonet.CommandMessage:
		fmt.Println("mm.CmdName:", mm.CmdName)
		if mm.CmdName == "sni" {
			return nil, nil, newSNIError(myi.ps.RespondToCommand(mm, myi.sniResponse()))
		}
		if mm.CmdName == "isMaster" {
			return mm, nil, nil
		}
		return mm, nil, nil

	case *mongonet.MessageMessage:
		fmt.Println("message message:", mm.Serialize())
		// if mm.CmdName == "sni" {
		// 	return nil, nil, newSNIError(myi.ps.RespondToCommand(mm, myi.sniResponse()))
		// }
		// if mm.CmdName == "isMaster" {
		// 	return mm, nil, nil
		// }
		return m, nil, nil
	}

	fmt.Println("normal query")
	return m, nil, nil
}

func (myi *MyInterceptor) Close() {
}
func (myi *MyInterceptor) TrackRequest(mongonet.MessageHeader) {
}
func (myi *MyInterceptor) TrackResponse(mongonet.MessageHeader) {
}

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
}

func newSNIError(err error) error {
	if err == nil {
		return nil
	}

	return mongonet.NewMongoError(err, errorCode, errorCodeName)
}
