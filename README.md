# go-imapreader [![Build Status](https://travis-ci.org/mvader/go-imapreader.svg?branch=master)](https://travis-ci.org/mvader/go-imapreader) [![GoDoc](https://godoc.org/github.com/mvader/go-imapreader?status.svg)](http://godoc.org/github.com/mvader/go-imapreader)
Simple interface for reading IMAP emails in Golang

## Install

```
go get gopkg.in/mvader/go-imapreader.v1
```

## Usage

```go
import (
  "gopkg.in/mvader/go-imapreader.v1"
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
