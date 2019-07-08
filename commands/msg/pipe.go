package msg

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/quotedprintable"
	"os/exec"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/widgets"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/gdamore/tcell"
)

type Pipe struct{}

func init() {
	register(Pipe{})
}

func (_ Pipe) Aliases() []string {
	return []string{"pipe"}
}

func (_ Pipe) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Pipe) Execute(aerc *widgets.Aerc, args []string) error {
	var (
		background bool
		pipeFull   bool
		pipePart   bool
	)
	// TODO: let user specify part by index or preferred mimetype
	opts, optind, err := getopt.Getopts(args, "bmp")
	if err != nil {
		return err
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'b':
			background = true
		case 'm':
			if pipePart {
				return errors.New("-m and -p are mutually exclusive")
			}
			pipeFull = true
		case 'p':
			if pipeFull {
				return errors.New("-m and -p are mutually exclusive")
			}
			pipePart = true
		}
	}
	cmd := args[optind:]
	if len(cmd) == 0 {
		return errors.New("Usage: pipe [-mp] <cmd> [args...]")
	}

	provider := aerc.SelectedTab().(widgets.ProvidesMessage)
	if !pipeFull && !pipePart {
		if _, ok := provider.(*widgets.MessageViewer); ok {
			pipePart = true
		} else if _, ok := provider.(*widgets.AccountView); ok {
			pipeFull = true
		} else {
			return errors.New(
				"Neither -m nor -p specified and cannot infer default")
		}
	}

	doTerm := func(reader io.Reader, name string) {
		term, err := commands.QuickTerm(aerc, cmd, reader)
		if err != nil {
			aerc.PushError(" " + err.Error())
			return
		}
		aerc.NewTab(term, name)
	}

	doExec := func(reader io.Reader) {
		ecmd := exec.Command(cmd[0], cmd[1:]...)
		err := ecmd.Run()
		if err != nil {
			aerc.PushStatus(" "+err.Error(), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorRed)
		} else {
			aerc.PushStatus(fmt.Sprintf(
				"%s: complete", args[0]), 10*time.Second).
				Color(tcell.ColorDefault, tcell.ColorDefault)
		}
	}

	if pipeFull {
		store := provider.Store()
		msg := provider.SelectedMessage()
		store.FetchFull([]uint32{msg.Uid}, func(reader io.Reader) {
			if background {
				doExec(reader)
			} else {
				doTerm(reader, fmt.Sprintf(
					"%s <%s", cmd[0], msg.Envelope.Subject))
			}
		})
	} else if pipePart {
		p := provider.SelectedMessagePart()
		p.Store.FetchBodyPart(p.Msg.Uid, p.Index, func(reader io.Reader) {
			// email parts are encoded as 7bit (plaintext), quoted-printable, or base64
			if strings.EqualFold(p.Part.Encoding, "base64") {
				reader = base64.NewDecoder(base64.StdEncoding, reader)
			} else if strings.EqualFold(p.Part.Encoding, "quoted-printable") {
				reader = quotedprintable.NewReader(reader)
			}

			if background {
				doExec(reader)
			} else {
				name := fmt.Sprintf("%s <%s/[%d]",
					cmd[0], p.Msg.Envelope.Subject, p.Index)
				doTerm(reader, name)
			}
		})
	}

	return nil
}