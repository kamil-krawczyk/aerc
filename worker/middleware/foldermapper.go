package middleware

import (
	"strings"
	"sync"

	"git.sr.ht/~rjarry/aerc/worker/types"
)

type folderMapper struct {
	sync.Mutex
	types.WorkerInteractor
	fm    folderMap
	table map[string]string
}

func NewFolderMapper(base types.WorkerInteractor, mapping map[string]string,
	order []string,
) types.WorkerInteractor {
	base.Infof("loading worker middleware: foldermapper")
	return &folderMapper{
		WorkerInteractor: base,
		fm:               folderMap{mapping, order},
		table:            make(map[string]string),
	}
}

func (f *folderMapper) incoming(msg types.WorkerMessage, dir string) string {
	f.Lock()
	defer f.Unlock()
	mapped := f.table[dir]
	return mapped
}

func (f *folderMapper) outgoing(msg types.WorkerMessage, dir string) string {
	f.Lock()
	defer f.Unlock()
	for k, v := range f.table {
		if v == dir {
			mapped := k
			return mapped
		}
	}
	return dir
}

func (f *folderMapper) store(s string) {
	f.Lock()
	defer f.Unlock()
	display := f.fm.Apply(s)
	f.table[display] = s
	f.Tracef("store display folder '%s' to '%s'", display, s)
}

func (f *folderMapper) create(s string) string {
	f.Lock()
	defer f.Unlock()
	backend := f.fm.Create(s)
	f.table[s] = backend
	f.Tracef("create display folder '%s' as '%s'", s, backend)
	return backend
}

func (f *folderMapper) ProcessAction(msg types.WorkerMessage) types.WorkerMessage {
	switch msg := msg.(type) {
	case *types.CheckMail:
		for i := range msg.Directories {
			msg.Directories[i] = f.incoming(msg, msg.Directories[i])
		}
	case *types.CopyMessages:
		msg.Destination = f.incoming(msg, msg.Destination)
	case *types.AppendMessage:
		msg.Destination = f.incoming(msg, msg.Destination)
	case *types.MoveMessages:
		msg.Destination = f.incoming(msg, msg.Destination)
	case *types.CreateDirectory:
		msg.Directory = f.create(msg.Directory)
	case *types.RemoveDirectory:
		msg.Directory = f.incoming(msg, msg.Directory)
	case *types.OpenDirectory:
		msg.Directory = f.incoming(msg, msg.Directory)
	}

	return f.WorkerInteractor.ProcessAction(msg)
}

func (f *folderMapper) PostMessage(msg types.WorkerMessage, cb func(m types.WorkerMessage)) {
	switch msg := msg.(type) {
	case *types.Done:
		switch msg := msg.InResponseTo().(type) {
		case *types.CheckMail:
			for i := range msg.Directories {
				msg.Directories[i] = f.outgoing(msg, msg.Directories[i])
			}
		case *types.CopyMessages:
			msg.Destination = f.outgoing(msg, msg.Destination)
		case *types.AppendMessage:
			msg.Destination = f.outgoing(msg, msg.Destination)
		case *types.MoveMessages:
			msg.Destination = f.outgoing(msg, msg.Destination)
		case *types.CreateDirectory:
			msg.Directory = f.outgoing(msg, msg.Directory)
		case *types.RemoveDirectory:
			msg.Directory = f.outgoing(msg, msg.Directory)
		case *types.OpenDirectory:
			msg.Directory = f.outgoing(msg, msg.Directory)
		}
	case *types.Directory:
		f.store(msg.Dir.Name)
		msg.Dir.Name = f.outgoing(msg, msg.Dir.Name)
	case *types.DirectoryInfo:
		msg.Info.Name = f.outgoing(msg, msg.Info.Name)
	}
	f.WorkerInteractor.PostMessage(msg, cb)
}

// folderMap contains the mapping between the ui and backend folder names
type folderMap struct {
	mapping map[string]string
	order   []string
}

// Apply applies the mapping from the folder map to the backend folder
func (f *folderMap) Apply(s string) string {
	for _, k := range f.order {
		v := f.mapping[k]
		strict := true
		if strings.HasSuffix(v, "*") {
			v = strings.TrimSuffix(v, "*")
			strict = false
		}
		if (strings.HasPrefix(s, v) && !strict) || (s == v && strict) {
			s = k + strings.TrimPrefix(s, v)
		}
	}
	return s
}

// Create reverses the mapping of a new folder name
func (f *folderMap) Create(s string) string {
	max, key := 0, ""
	for k := range f.mapping {
		if strings.HasPrefix(s, k) && len(k) > max {
			max, key = len(k), k
		}
	}
	if max > 0 && key != "" {
		s = strings.TrimSuffix(f.mapping[key], "*") +
			strings.TrimPrefix(s, key)
	}
	return s
}
