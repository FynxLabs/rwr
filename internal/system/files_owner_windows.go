//go:build windows

package system

import "os"

// fileOwnedByEUID: Windows has no unix ownership; os.FileMode carries no
// security there either (files report 0666), so mode inheritance is a no-op
// concern and the check always passes.
func fileOwnedByEUID(os.FileInfo) bool {
	return true
}

// ownedByRootOrEUID: see fileOwnedByEUID - Windows security is ACLs, which
// os.FileInfo cannot express, so the unix ownership check always passes.
func ownedByRootOrEUID(os.FileInfo) bool {
	return true
}
