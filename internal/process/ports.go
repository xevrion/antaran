package process

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// scanPorts returns a map of pid -> []port by reading /proc/net/tcp and /proc/net/tcp6.
// Ports are in host byte order (human-readable).
func scanPorts() (map[uint32][]uint16, error) {
	inodeToPorts := map[uint64][]uint16{}
	for _, f := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		if err := parseNetTCP(f, inodeToPorts); err != nil {
			continue // file may not exist (e.g. no IPv6)
		}
	}

	pidToInodes, err := mapPIDsToInodes()
	if err != nil {
		return nil, err
	}

	result := map[uint32][]uint16{}
	for pid, inodes := range pidToInodes {
		for _, inode := range inodes {
			if ports, ok := inodeToPorts[inode]; ok {
				result[pid] = append(result[pid], ports...)
			}
		}
	}
	return result, nil
}

// parseNetTCP parses /proc/net/tcp or /proc/net/tcp6.
// The local_address field (column 1, 0-indexed) is "HEXIP:HEXPORT" in little-endian hex.
// State 0A = TCP_LISTEN. We only care about listening ports.
func parseNetTCP(path string, out map[uint64][]uint16) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		state := fields[3]
		if state != "0A" { // 0A = LISTEN
			continue
		}
		localAddr := fields[1]
		port, err := parseHexPort(localAddr)
		if err != nil {
			continue
		}
		inode, err := parseInode(fields[9])
		if err != nil {
			continue
		}
		out[inode] = append(out[inode], port)
	}
	return nil
}

// parseHexPort extracts the port from "HEXADDR:HEXPORT".
// Ports in /proc/net/tcp are big-endian hex regardless of arch.
func parseHexPort(addrPort string) (uint16, error) {
	parts := strings.SplitN(addrPort, ":", 2)
	if len(parts) != 2 || len(parts[1]) != 4 {
		return 0, fmt.Errorf("bad addr:port %q", addrPort)
	}
	b, err := hex.DecodeString(parts[1])
	if err != nil {
		return 0, err
	}
	return uint16(b[0])<<8 | uint16(b[1]), nil
}

func parseInode(s string) (uint64, error) {
	var n uint64
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// mapPIDsToInodes reads /proc/<pid>/fd/* for every process and maps
// pid -> socket inodes it holds open.
func mapPIDsToInodes() (map[uint32][]uint64, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	result := map[uint32][]uint64{}
	for _, e := range entries {
		var pid uint32
		if _, err := fmt.Sscanf(e.Name(), "%d", &pid); err != nil {
			continue
		}
		inodes := socketInodesForPID(pid)
		if len(inodes) > 0 {
			result[pid] = inodes
		}
	}
	return result, nil
}

func socketInodesForPID(pid uint32) []uint64 {
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return nil // process may have exited
	}

	var inodes []uint64
	for _, e := range entries {
		link, err := os.Readlink(fmt.Sprintf("%s/%s", fdDir, e.Name()))
		if err != nil {
			continue
		}
		var inode uint64
		if _, err := fmt.Sscanf(link, "socket:[%d]", &inode); err == nil {
			inodes = append(inodes, inode)
		}
	}
	return inodes
}
