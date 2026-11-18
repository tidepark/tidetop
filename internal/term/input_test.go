package term

import "testing"

func TestInputSplitsReplies(t *testing.T) {
	var in Input
	var keys []byte
	replies := 0
	for _, b := range []byte("a\x1b[12;80Rq\x1b[A\x1b[1;5Rn\x1bz\x1b[0n") {
		k, isKey, reply := in.Feed(b)
		if isKey {
			keys = append(keys, k)
		}
		if reply {
			replies++
		}
	}
	if string(keys) != "aqnz" || replies != 2 {
		t.Fatalf("keys %q replies %d", keys, replies)
	}
}
