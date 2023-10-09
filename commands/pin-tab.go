package commands

import (
	"fmt"

	"git.sr.ht/~rjarry/aerc/app"
)

type PinTab struct{}

func init() {
	register(PinTab{})
}

func (PinTab) Aliases() []string {
	return []string{"pin-tab", "unpin-tab"}
}

func (PinTab) Complete(args []string) []string {
	return nil
}

func (PinTab) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: %s", args[0])
	}

	switch args[0] {
	case "pin-tab":
		app.PinTab()
	case "unpin-tab":
		app.UnpinTab()
	}

	return nil
}
