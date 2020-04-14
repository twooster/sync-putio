package main

import "fmt"

const KIB = 1024
const MIB = KIB * 1024
const GIB = MIB * 1024

func bytesToHuman(bytes int64) string {
	if bytes > GIB {
		return fmt.Sprintf("%.3f GiB", float64(bytes)/GIB)
	} else if bytes > MIB {
		return fmt.Sprintf("%.3f MiB", float64(bytes)/MIB)
	} else if bytes > KIB {
		return fmt.Sprintf("%.3f KiB", float64(bytes)/KIB)
	}
	return fmt.Sprintf("%v B", bytes)
}
