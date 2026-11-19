package local

import (
	"os"
	"strconv"
	"strings"
)

func StartOf(pid int) (string, error) {
	// outside: /proc/<pid>/stat is `pid (comm) state ...`, comm holding any
	// byte; its 22nd field is the start in clock ticks since boot, which a
	// boot of its own tells apart from the same tick of another boot.
	if pid <= 0 {
		return "", ErrNoProcess
	}
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return "", ErrNoProcess
	}
	end := strings.LastIndexByte(string(stat), ')')
	if end < 0 {
		return "", ErrNoProcess
	}
	fields := strings.Fields(string(stat[end+1:]))
	if len(fields) < 20 || fields[0] == "Z" || fields[0] == "X" {
		return "", ErrNoProcess
	}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(boot)) + ":" + fields[19], nil
}
