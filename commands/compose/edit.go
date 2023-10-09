package compose

import (
	"errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~sircmpwn/getopt"
)

type Edit struct{}

func init() {
	register(Edit{})
}

func (Edit) Aliases() []string {
	return []string{"edit"}
}

func (Edit) Complete(args []string) []string {
	return nil
}

func (Edit) Execute(args []string) error {
	composer, ok := app.SelectedTabContent().(*app.Composer)
	if !ok {
		return errors.New("only valid while composing")
	}

	editHeaders := config.Compose.EditHeaders
	opts, optind, err := getopt.Getopts(args, "eE")
	if err != nil {
		return err
	}
	if len(args) != optind {
		return errors.New("Usage: edit [-e|-E]")
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'e':
			editHeaders = true
		case 'E':
			editHeaders = false
		}
	}

	err = composer.ShowTerminal(editHeaders)
	if err != nil {
		return err
	}
	composer.FocusTerminal()
	return nil
}
