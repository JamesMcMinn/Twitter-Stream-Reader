package main

import (
	"flag"
	"github.com/jamesmcminn/twitter"
	"log"
	"net"
	"runtime"
)

const RECV_BUF_LEN = 1024 * 1024
const MAX_CHAN_LEN = 1000000

var (
	consumerKey    *string               = flag.String("ck", "", "Consumer Key")
	consumerSecret *string               = flag.String("cs", "", "Consumer Secret")
	ot             *string               = flag.String("ot", "", "Oauth Token")
	osec           *string               = flag.String("os", "", "OAuthTokenSecret")
	firehose       chan twitter.Tweet    = make(chan twitter.Tweet, MAX_CHAN_LEN)
	aliveStreams   map[chan *[]byte]bool = make(map[chan *[]byte]bool)
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	flag.Parse()

	ln, err := net.Listen("tcp", ":8053")
	if err != nil {
		log.Fatal(err)
	}

	go twitter.FillStream(firehose, *consumerKey, *consumerSecret, *ot, *osec)
	go fillOutgoingStreams(aliveStreams)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	stream := make(chan *[]byte, MAX_CHAN_LEN)
	aliveStreams[stream] = true
	log.Println("Current Connections:", len(aliveStreams))

	for {
		t := <-stream
		_, err := conn.Write(*t)
		if err != nil {
			println("Closing connection: ", err.Error())
			break
		}
	}

	delete(aliveStreams, stream)
	log.Println("Current Connections:", len(aliveStreams))
}

func fillOutgoingStreams(streams map[chan *[]byte]bool) {
	for {
		tweet := <-firehose
		for r := range streams {
			if len(r) == MAX_CHAN_LEN {
				<-r
			}
			json, _ := twitter.TweetToJSON(tweet)
			json = append(json, []byte("\n")...)
			r <- &json
		}
	}
}
