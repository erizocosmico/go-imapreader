package imapreader

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mxk/go-imap/imap"

	. "gopkg.in/check.v1"
)

type ReaderSuite struct {
}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&ReaderSuite{})

const Msg = `
Subject: Fancy ponies
From: Fancy pony <fancy@ponies.org>

Hello, ponies
`

func (s *ReaderSuite) setupFixtures(c *C, markSeen bool) (*reader, string) {
	opts := Options{
		Addr:     os.Getenv("TEST_ADDR"),
		Username: os.Getenv("TEST_USER"),
		Password: os.Getenv("TEST_PWD"),
		TLS:      true,
		Timeout:  60 * time.Second,
		MarkSeen: markSeen,
	}

	_r, err := NewReader(opts)
	c.Assert(err, IsNil)
	r := _r.(*reader)
	c.Assert(r.Login(), IsNil)

	mailbox := fmt.Sprintf("MBOX_%d", time.Now().UnixNano())
	if cmd, err := imap.Wait(r.client.Create(mailbox)); err != nil {
		if rsp, ok := err.(imap.ResponseError); ok && rsp.Status == imap.NO {
			_, _, err := r.exec(r.client.Delete(mailbox))
			c.Assert(err, IsNil)
		}
	} else {
		_, err := cmd.Result(imap.OK)
		c.Assert(err, IsNil)
	}

	msg := []byte(strings.Replace(Msg[1:], "\n", "\r\n", -1))
	_, _, err = r.exec(r.client.Append(mailbox, nil, nil, imap.NewLiteral(msg)))
	c.Assert(err, IsNil)

	return r, mailbox
}

func (s *ReaderSuite) teardownFixtures(c *C, r *reader, mailbox string) {
	_, _, err := r.exec(r.client.Delete(mailbox))
	c.Assert(err, IsNil)
	c.Assert(r.Logout(), IsNil)
}

func (s *ReaderSuite) TestListNoMarkSeen(c *C) {
	r, mailbox := s.setupFixtures(c, false)

	msgs, err := r.List(mailbox, SearchUnseen)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)

	msg := msgs[0]
	c.Assert(string(msg.Body), Equals, "Hello, ponies\r\n")
	c.Assert(msg.Header.Get("Subject"), Equals, "Fancy ponies")
	c.Assert(msg.Header.Get("From"), Equals, "Fancy pony <fancy@ponies.org>")
	c.Assert(msg.Flags, HasLen, 0)

	s.teardownFixtures(c, r, mailbox)
}

func (s *ReaderSuite) TestListMarkSeen(c *C) {
	r, mailbox := s.setupFixtures(c, true)

	msgs, err := r.List(mailbox, SearchUnseen)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)

	msg := msgs[0]
	c.Assert(string(msg.Body), Equals, "Hello, ponies\r\n")
	c.Assert(msg.Header.Get("Subject"), Equals, "Fancy ponies")
	c.Assert(msg.Header.Get("From"), Equals, "Fancy pony <fancy@ponies.org>")
	c.Assert(msg.Flags, HasLen, 0)

	msgs, err = r.List(mailbox, SearchUnseen)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 0)

	s.teardownFixtures(c, r, mailbox)
}
