# Twitter Stream Reader
Opens a file or connects to the Twitter streaming API, and converts it to a 
standard JSON format, very similar to the standard Twitter API JSON format.
This can be used for development or in a production environment, and allows
the case code to be used for both experimental and live testing.

## Requirements
This requires the following packages:
 - mirgit.dcs.gla.ac.uk/JamesMcMinn/twitter

 Required packages can be installed using `go get packagepath`

## Installing
Twitter Stream Reader can be installed using a number of methods.

The easiest method is to simply run `main.go` using the `go run` command: 
    go run main.go

To  build the application binary, `go build` can be used:

    run `go build mirgit.dcs.gla.ac.uk/JamesMcMinn/twitter-stream-reader`

This will generate the binary in your current directory.

## Usage
The stream reader has 2 modes:
	- **File Mode**: Opens and reads a file, simulating the Twitter gardenhose
	for testing and development.
	- **Live Mode**: Using a set of specified keys, connect to the Twitter 
	steaming API and read the gardenhose.

### Parameters

  -ck= Consumer Key
  -cs= Consumer Secret
  -os= OAuthTokenSecret
  -ot= OAuth Token

  -if= Input File to read from for File Mode

  -port= (Optional) Port to listen on which other applications can connect to. Optional. Default: 8053


### Live Mode Example
**Note** You will need to generate OAuth keys by visiting http://developer.twitter.com

	go run main.go -ck=consumerKey -cs=consumerSecret -ot=OAuthToken -os=OAuthTokenSecret

### File Mode Example
The file should contain valid JSON, with one Tweet per line. The stream reader will read this file,
parse any JSON, and output it in the same format used the Streaming API. In this example, we also 
use port 85632 rather than the default port or 8053.
 
    go run main.go -if=/home/james/tweets.json -port=85632