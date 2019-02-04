# go-imapreader [![Build Status](https://travis-ci.org/erizocosmico/go-imapreader.svg?branch=master)](https://travis-ci.org/erizocosmico/go-imapreader) [![GoDoc](https://godoc.org/github.com/erizocosmico/go-imapreader?status.svg)](http://godoc.org/github.com/erizocosmico/go-imapreader)
Simple interface for reading IMAP emails in Golang.

## Install

```
go get github.com/erizocosmico/go-imapreader
```

## Usage

```go
import (
  "github.com/erizocosmico/go-imapreader"
)

func main() {
  r, err := imapreader.NewReader(imapreader.Options{
		Addr:     os.Getenv("TEST_ADDR"),
		Username: os.Getenv("TEST_USER"),
		Password: os.Getenv("TEST_PWD"),
		TLS:      true,
		Timeout:  60 * time.Second,
		MarkSeen: true,
	})
	if err != nil {
	  panic(err)
	}
	
	if err := r.Login(); err != nil {
	  panic(err)
	}
	defer r.Logout()
	
	// Search for all the emails in "all mail" that are unseen
	// read the docs for more search filters
	messages, err := r.List(imapreader.GMailAllMail, imapreader.SearchUnseen)
	if err != nil {
	  panic(err)
	}
	
	// do stuff with messages
}
```
