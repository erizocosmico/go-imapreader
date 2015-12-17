package imapreader

import (
	"bytes"
	"io/ioutil"
	"net/mail"
	"time"

	"github.com/mxk/go-imap/imap"
)

const (
	// GMail inbox mailbox
	GMailInbox string = "INBOX"
	// GMail all mail mailbox
	GMailAllMail string = "[Gmail]/All Mail"
)

var (
	// Search only the unseen messages
	SearchUnseen = []imap.Field{"UNSEEN"}
	// Search all messages
	SearchAll = []imap.Field{"ALL"}
	// Search only answered messages
	SearchAnswered = []imap.Field{"ANSWERED"}
	// Search only unanswered messages
	SearchUnanswered = []imap.Field{"UNANSWERED"}
	// Search only deleted messages
	SearchDeleted = []imap.Field{"DELETED"}
	// Search only not deleted messages
	SearchUndeleted = []imap.Field{"UNDELETED"}
	// Search only flagged messages
	SearchFlagged = []imap.Field{"FLAGGED"}
	// Search only not flagged messages
	SearchUnflagged = []imap.Field{"UNFLAGGED"}
	// Search only new messages
	SearchNew = []imap.Field{"NEW"}
	// Search only old messages
	SearchOld = []imap.Field{"OLD"}
	// Search only recent messages
	SearchRecent = []imap.Field{"RECENT"}
	// Search only seen messages
	SearchSeen = []imap.Field{"SEEN"}
)

// Reader is a client to read messages from an IMAP server
// only contains the List operation as well as login and logout operations
type Reader interface {
	// Login logs in with the options provided using the connection established
	// when the reader was created
	Login() error
	// Logout terminates the current "session"
	Logout() error
	// List retrieves the list of emails in a mailbox that satisfy the given search criteria
	List(string, []imap.Field) ([]*Email, error)
}

type reader struct {
	opts   Options
	client *imap.Client
}

// Options define the settings to perform all the reader operations
type Options struct {
	// IMAP server address with port
	Addr string
	// Username
	Username string
	// Password
	Password string
	// Use TLS for the connection
	TLS bool
	// Max timeout for logging out
	Timeout time.Duration
	// Mark all the retrieved messages as seen when retrieved
	MarkSeen bool
}

type Email struct {
	// Array of flags the message has
	Flags []string
	// Contains all the message headers
	Header mail.Header
	// Contains the message body
	Body []byte
}

// NewReader constructs a new Reader instance with the given Options
func NewReader(opts Options) (Reader, error) {
	client, err := connect(opts)
	if err != nil {
		return nil, err
	}

	return &reader{
		opts:   opts,
		client: client,
	}, nil
}

// Login initiates the IMAP session
func (r *reader) Login() error {
	cmd, err := r.client.Login(r.opts.Username, r.opts.Password)
	if err != nil {
		return err
	}

	_, err = cmd.Result(imap.OK)
	return err
}

// Logout terminates the IMAP session
func (r *reader) Logout() error {
	cmd, err := r.client.Logout(r.opts.Timeout)
	if err != nil {
		return err
	}

	_, err = cmd.Result(imap.OK)
	return err
}

// List performs a search with the given params in the given mailbox and
// returns the list of emails matching that criteria
func (r *reader) List(mailbox string, params []imap.Field) ([]*Email, error) {
	if err := r.mailbox(mailbox, true); err != nil {
		return nil, err
	}

	set, err := r.search(params)
	if err != nil {
		return nil, err
	}

	emails, err := r.fetch(set)
	if err != nil {
		return nil, err
	}

	if err := r.closeMailbox(); err != nil {
		return nil, err
	}

	if r.opts.MarkSeen && len(emails) > 0 {
		if err := r.mailbox(mailbox, false); err != nil {
			return nil, err
		}

		if err := r.markSeen(set); err != nil {
			return nil, err
		}

		if err := r.closeMailbox(); err != nil {
			return nil, err
		}
	}

	return emails, nil
}

func (r *reader) mailbox(mailbox string, readOnly bool) error {
	_, _, err := r.exec(r.client.Select(mailbox, readOnly))
	return err
}

func (r *reader) closeMailbox() error {
	_, _, err := r.exec(r.client.Close(false))
	return err
}

func (r *reader) search(params []imap.Field) (*imap.SeqSet, error) {
	if len(params) > 1 {
		for i, p := range params[1:] {
			params[i+1] = r.client.Quote(p)
		}
	}

	cmd, _, err := r.exec(r.client.UIDSearch(params...))
	if err != nil {
		return nil, err
	}

	set, _ := imap.NewSeqSet("")
	results := cmd.Data[0].SearchResults()
	if len(results) == 0 {
		return nil, nil
	}

	set.AddNum(results...)
	return set, nil
}

func (r *reader) fetch(set *imap.SeqSet) ([]*Email, error) {
	if set == nil {
		return nil, nil
	}

	cmd, _, err := r.exec(r.client.UIDFetch(set, "FLAGS", "BODY[]"))
	if err != nil {
		return nil, err
	}

	emails, err := r.emailsFromResponse(cmd.Data)
	if err != nil {
		return nil, err
	}

	return emails, nil
}

func (r *reader) markSeen(set *imap.SeqSet) error {
	_, _, err := r.exec(r.client.UIDStore(set, "+FLAGS.SILENT", imap.NewFlagSet(`\Seen`)))
	return err
}

func (r *reader) emailsFromResponse(data []*imap.Response) ([]*Email, error) {
	var emails []*Email

	for _, d := range data {
		email, err := newEmail(d)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}

	return emails, nil
}

func (r *reader) exec(cmd *imap.Command, err error) (*imap.Command, *imap.Response, error) {
	if err != nil {
		return nil, nil, err
	}

	resp, err := cmd.Result(imap.OK)
	if err != nil {
		return nil, nil, err
	}

	return cmd, resp, nil
}

func newEmail(resp *imap.Response) (*Email, error) {
	var buf bytes.Buffer
	info := resp.MessageInfo()
	if _, err := info.Attrs["BODY[]"].(imap.Literal).WriteTo(&buf); err != nil {
		return nil, err
	}

	msg, err := mail.ReadMessage(&buf)
	if err != nil {
		return nil, err
	}

	var flags []string
	for k := range info.Flags {
		flags = append(flags, k)
	}

	body, err := ioutil.ReadAll(msg.Body)
	if err != nil {
		return nil, err
	}

	return &Email{
		Header: msg.Header,
		Body:   body,
		Flags:  flags,
	}, nil
}

func connect(opts Options) (*imap.Client, error) {
	if opts.TLS {
		return imap.DialTLS(opts.Addr, nil)
	} else {
		return imap.Dial(opts.Addr)
	}
}

// BySubject returns the search paramters to perform a search by subject
func BySubject(subject string) []imap.Field {
	return []imap.Field{"SUBJECT", subject}
}

// ByFrom returns the search parameters to perform a search by FROM address
func ByFrom(from string) []imap.Field {
	return []imap.Field{"FROM", from}
}

// TODO: Implement more search filters
