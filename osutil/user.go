package osutil

import (
	"os/user"
	"strconv"
)

// User looks up the username u and return the uid and gid. If the username can't be found 0, 0 is returned.
func User(u string) (int64, int64) {
	u1, err := user.Lookup(u)
	if err != nil {
		return 0, 0
	}
	uid, _ := strconv.ParseInt(u1.Uid, 10, 32)
	gid, _ := strconv.ParseInt(u1.Gid, 10, 32)
	return uid, gid
}
