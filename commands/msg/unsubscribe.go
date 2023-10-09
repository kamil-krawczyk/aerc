package msg

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~sircmpwn/getopt"
	"github.com/emersion/go-message/mail"
)

// Unsubscribe helps people unsubscribe from mailing lists by way of the
// List-Unsubscribe header.
type Unsubscribe struct{}

func init() {
	register(Unsubscribe{})
}

// Aliases returns a list of aliases for the :unsubscribe command
func (Unsubscribe) Aliases() []string {
	return []string{"unsubscribe"}
}

// Complete returns a list of completions
func (Unsubscribe) Complete(args []string) []string {
	return nil
}

// Execute runs the Unsubscribe command
func (Unsubscribe) Execute(args []string) error {
	editHeaders := config.Compose.EditHeaders
	opts, optind, err := getopt.Getopts(args, "eE")
	if err != nil {
		return err
	}
	if len(args) != optind {
		return errors.New("Usage: unsubscribe [-e|-E]")
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'e':
			editHeaders = true
		case 'E':
			editHeaders = false
		}
	}
	widget := app.SelectedTabContent().(app.ProvidesMessage)
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}
	headers := msg.RFC822Headers
	if !headers.Has("list-unsubscribe") {
		return errors.New("No List-Unsubscribe header found")
	}
	text, err := headers.Text("list-unsubscribe")
	if err != nil {
		return err
	}
	methods := parseUnsubscribeMethods(text)
	if len(methods) == 0 {
		return fmt.Errorf("no methods found to unsubscribe")
	}
	log.Debugf("unsubscribe: found %d methods", len(methods))

	unsubscribe := func(method *url.URL) {
		log.Debugf("unsubscribe: trying to unsubscribe using %s", method.Scheme)
		var err error
		switch strings.ToLower(method.Scheme) {
		case "mailto":
			err = unsubscribeMailto(method, editHeaders)
		case "http", "https":
			err = unsubscribeHTTP(method)
		default:
			err = fmt.Errorf("unsubscribe: skipping unrecognized scheme: %s", method.Scheme)
		}
		if err != nil {
			app.PushError(err.Error())
		}
	}

	var title string = "Select method to unsubscribe"
	if msg != nil && msg.Envelope != nil && len(msg.Envelope.From) > 0 {
		title = fmt.Sprintf("%s from %s", title, msg.Envelope.From[0])
	}

	options := make([]string, len(methods))
	for i, method := range methods {
		options[i] = method.Scheme
	}

	dialog := app.NewSelectorDialog(
		title,
		"Press <Enter> to confirm or <ESC> to cancel",
		options, 0, app.SelectedAccountUiConfig(),
		func(option string, err error) {
			app.CloseDialog()
			if err != nil {
				if errors.Is(err, app.ErrNoOptionSelected) {
					app.PushStatus("Unsubscribe: "+err.Error(),
						5*time.Second)
				} else {
					app.PushError("Unsubscribe: " + err.Error())
				}
				return
			}
			for _, m := range methods {
				if m.Scheme == option {
					unsubscribe(m)
					return
				}
			}
			app.PushError("Unsubscribe: selected method not found")
		},
	)
	app.AddDialog(dialog)

	return nil
}

// parseUnsubscribeMethods reads the list-unsubscribe header and parses it as a
// list of angle-bracket <> deliminated URLs. See RFC 2369.
func parseUnsubscribeMethods(header string) (methods []*url.URL) {
	r := bufio.NewReader(strings.NewReader(header))
	for {
		// discard until <
		_, err := r.ReadSlice('<')
		if err != nil {
			return
		}
		// read until <
		m, err := r.ReadSlice('>')
		if err != nil {
			return
		}
		m = m[:len(m)-1]
		if u, err := url.Parse(string(m)); err == nil {
			methods = append(methods, u)
		}
	}
}

func unsubscribeMailto(u *url.URL, editHeaders bool) error {
	widget := app.SelectedTabContent().(app.ProvidesMessage)
	acct := widget.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	h := &mail.Header{}
	h.SetSubject(u.Query().Get("subject"))
	if to, err := mail.ParseAddressList(u.Opaque); err == nil {
		h.SetAddressList("to", to)
	}

	composer, err := app.NewComposer(

		acct,
		acct.AccountConfig(),
		acct.Worker(),
		editHeaders,
		"",
		h,
		nil,
		strings.NewReader(u.Query().Get("body")),
	)
	if err != nil {
		return err
	}
	composer.Tab = app.NewTab(composer, "unsubscribe")
	composer.FocusTerminal()
	return nil
}

func unsubscribeHTTP(u *url.URL) error {
	confirm := app.NewSelectorDialog(
		"Do you want to open this link?",
		u.String(),
		[]string{"No", "Yes"}, 0, app.SelectedAccountUiConfig(),
		func(option string, _ error) {
			app.CloseDialog()
			switch option {
			case "Yes":
				go func() {
					defer log.PanicHandler()
					mime := fmt.Sprintf("x-scheme-handler/%s", u.Scheme)
					if err := lib.XDGOpenMime(u.String(), mime, nil); err != nil {
						app.PushError("Unsubscribe:" + err.Error())
					}
				}()
			default:
				app.PushError("Unsubscribe: link will not be opened")
			}
		},
	)
	app.AddDialog(confirm)
	return nil
}
