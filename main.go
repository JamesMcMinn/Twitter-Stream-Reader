package main

import (
	"flag"
	"github.com/araddon/httpstream"
	"github.com/jamesmcminn/twitter"
	"github.com/mrjones/oauth"
	"log"
	"net"
	"runtime"
)

const RECV_BUF_LEN = 1024 * 1024
const MAX_CHAN_LEN = 1000000

var (
	consumerKey    *string              = flag.String("ck", "", "Consumer Key")
	consumerSecret *string              = flag.String("cs", "", "Consumer Secret")
	ot             *string              = flag.String("ot", "", "Oauth Token")
	osec           *string              = flag.String("os", "", "OAuthTokenSecret")
	firehose       chan []byte          = make(chan []byte, MAX_CHAN_LEN)
	aliveStreams   map[chan []byte]bool = make(map[chan []byte]bool)
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	flag.Parse()
	done := make(chan bool)

	httpstream.OauthCon = oauth.NewConsumer(
		*consumerKey,
		*consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "http://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})

	at := oauth.AccessToken{
		Token:  *ot,
		Secret: *osec,
	}

	client := httpstream.NewOAuthClient(&at, httpstream.OnlyTweetsFilter(func(line []byte) {
		if len(firehose) == MAX_CHAN_LEN {
			<-firehose
		}
		firehose <- line
	}))

	err := client.Sample(done)
	if err != nil {
		httpstream.Log(httpstream.ERROR, err.Error())
	}

	ln, err := net.Listen("tcp", ":8053")
	if err != nil {
		log.Fatal(err)
	}

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
	stream := make(chan []byte, MAX_CHAN_LEN)
	aliveStreams[stream] = true
	log.Println("Current Connections:", len(aliveStreams))

	for {
		_, err := conn.Write(<-stream)
		if err != nil {
			println("Closing connection: ", err.Error())
			break
		}
	}

	delete(aliveStreams, stream)
	log.Println("Current Connections:", len(aliveStreams))
}

func fillOutgoingStreams(streams map[chan []byte]bool) {
	for {
		item := <-firehose
		for r := range streams {
			if len(r) == MAX_CHAN_LEN {
				<-r
			}
			r <- append(item, []byte("\n")...)
		}
	}
}
