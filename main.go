package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/jamesmcminn/twitter"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const RECV_BUF_LEN = 1024 * 1024
const MAX_CHAN_LEN = 10000

const MODE_STREAM = 0
const MODE_FILE = 1

const FORMAT_SNOW = 0
const FORMAT_STREAM = 1

var (
	consumerKey    *string               = flag.String("ck", "", "Consumer Key")
	consumerSecret *string               = flag.String("cs", "", "Consumer Secret")
	ot             *string               = flag.String("ot", "", "OAuth Token")
	osec           *string               = flag.String("os", "", "OAuthTokenSecret")
	inputFile      *string               = flag.String("if", "", "Input File")
	format         *string               = flag.String("format", "", "File Format")
	port           *int                  = flag.Int("port", 8053, "Port to listen on. Default: 8053")
	firehose       chan twitter.Tweet    = make(chan twitter.Tweet, MAX_CHAN_LEN)
	aliveStreams   map[chan *[]byte]bool = make(map[chan *[]byte]bool)
	mode           int                   = -1
	fileFormat     int                   = -1
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	flag.Parse()

	if *inputFile != "" {
		mode = MODE_FILE
		if *format == "snow" {
			fileFormat = FORMAT_SNOW
		} else if *format == "stream" {
			fileFormat = FORMAT_STREAM
		} else {
			fmt.Println("Must specify file type as either -snow or -stream. See -help for details.")
			return
		}
	} else if *consumerKey != "" || *consumerSecret != "" || *ot != "" || *osec != "" {
		if *consumerKey == "" || *consumerSecret == "" || *ot == "" || *osec == "" {
			fmt.Println("Must specify all of -ck, -cs, -ot and -os. See -help for details.")
			return
		}
	} else {
		fmt.Println("Must specify either Twitter OAuth details or file location and format. See -help for details.")
		return
	}

	// Listen on whatever port was specified
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(*port))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening on port", *port)

	if mode == MODE_STREAM {
		// Open a connection the the firehose and fill output streams
		go twitter.FillStream(firehose, *consumerKey, *consumerSecret, *ot, *osec)
		go fillOutgoingStreams(aliveStreams)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
		}
		go handleConnection(conn)
	}
}

func readFileInto(into chan *[]byte) {
	f, err := os.Open(*inputFile)
	if err != nil {
		log.Fatal(err)
	}

	bf := bufio.NewReaderSize(f, 20000)
	for {
		line, isPrefix, err := bf.ReadLine()
		switch {
		case err == io.EOF:
			break
		case err != nil:
			log.Fatal(err)
		case isPrefix:
			log.Fatal("Error: Unexpected long line reading", f.Name())
		}

		// Check the connection is still active
		if aliveStreams[into] != true {
			break
		}

		var t twitter.Tweet
		if fileFormat == FORMAT_STREAM {
			t = twitter.JSONtoTweet(line)
		} else {
			t = parseSNOW(line)
		}

		j, err := twitter.TweetToJSON(t)
		if err != nil {
			log.Println(err)
		}
		j = append(j, []byte("\n")...)
		into <- &j
	}
}

func parseSNOW(line []byte) twitter.Tweet {
	// TODO: Handle ParseInt errors
	t := new(twitter.Tweet)
	parts := strings.SplitN(string(line), "\t", 11)
	code := parts[5]
	if code == "200" {
		id, _ := strconv.ParseInt(parts[6], 10, 64)
		username := parts[7]
		text := parts[8]
		time, _ := strconv.ParseUint(parts[9], 10, 64)
		t.Id = id
		t.User.Name = username
		t.Text = text
		t.Timestamp = time
		return *t
	} else {
		return nil
	}
}

func handleConnection(conn net.Conn) {
	stream := make(chan *[]byte, MAX_CHAN_LEN)

	aliveStreams[stream] = true
	log.Println("Current Connections:", len(aliveStreams))

	if mode == MODE_FILE {
		go readFileInto(stream)
	}

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
