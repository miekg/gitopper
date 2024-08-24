package gitcmd

import (
	"bufio"
	"bytes"
	"strings"
)

// OfInterest will grep the diff stat for directories we care about. The returns boolen is true when we find a hit.
//
// We do a git pull, but we want to know if things changed that are of interest.
// This is done by parsing the diffstat on the git pull, not the best way, but it works.
// Problem is here is to keep track _when_ things changed, i.e. we can look in the revlog, but then
// we need to track that we saw a change. Parsing the diff stat seems simpler and more atomic in that
// regard.
func (g *Git) OfInterest(data []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Diff stat snippet:
	//
	// Fast-forward
	//  provisioning-systems.md | 139 +++++++++++++++++
	//  1 file changed, 139 insertions(+)
	// create mode 100644 provisioning-systems.md
	//
	// Start with a space, then non-space, and a pipe symbol somewhere in there.

	// this is O(n * m), but the number of dirs is usually very small, and the diffstat is
	// bunch of lines usually < 1000. So it's not too bad.
	for scanner.Scan() {
		text := scanner.Text()
		for _, d := range g.dirs {
			if strings.Contains(text, d) {
				return true
			}
		}
	}

	if scanner.Err() != nil {
		return false
	}
	return false
}
