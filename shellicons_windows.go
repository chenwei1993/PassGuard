//go:build windows

package main

import "syscall"

var shChangeNotify = syscall.NewLazyDLL("shell32.dll").NewProc("SHChangeNotify")

// refreshShellIcons tells the shell that file associations changed, which
// makes Explorer reread the icons of the app's shortcuts and executable.
// SHCNE_ASSOCCHANGED, SHCNF_IDLIST.
func refreshShellIcons() {
	const (
		shcneAssocChanged = 0x08000000
		shcnfIdList       = 0
	)
	_, _, _ = shChangeNotify.Call(shcneAssocChanged, shcnfIdList, 0, 0)
}
