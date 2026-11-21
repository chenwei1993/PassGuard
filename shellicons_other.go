//go:build !windows

package main

// refreshShellIcons is a no-op where the shell has no icon cache keyed by
// shortcut path.
func refreshShellIcons() {}
