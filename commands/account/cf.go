package account

import (
	"errors"
	"strings"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/state"
)

var history map[string]string

type ChangeFolder struct{}

func init() {
	history = make(map[string]string)
	register(ChangeFolder{})
}

func (ChangeFolder) Aliases() []string {
	return []string{"cf"}
}

func (ChangeFolder) Complete(args []string) []string {
	return commands.GetFolders(args)
}

func (ChangeFolder) Execute(args []string) error {
	if len(args) == 1 {
		return errors.New("Usage: cf <folder>")
	}
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	previous := acct.Directories().Selected()
	joinedArgs := strings.Join(args[1:], " ")
	if joinedArgs == "-" {
		if dir, ok := history[acct.Name()]; ok {
			acct.Directories().Select(dir)
		} else {
			return errors.New("No previous folder to return to")
		}
	} else {
		acct.Directories().Select(joinedArgs)
	}
	history[acct.Name()] = previous

	// reset store filtering if we switched folders
	store := acct.Store()
	if store != nil {
		store.ApplyClear()
		acct.SetStatus(state.SearchFilterClear())
	}
	return nil
}
