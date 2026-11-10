package procid

// Proc is the owner of a socket or one running process.
// Known is false when the platform cannot prove the owner.
// Start is the process start time from the kernel. A recycled PID has a new Start.
type Proc struct {
	PID   uint32
	Start uint64
	Name  string
	Path  string
	Known bool
}

// SameProcess is true only when both observations carry the same non-zero start time.
func SameProcess(a, b Proc) bool {
	return a.PID != 0 && a.PID == b.PID && a.Start != 0 && a.Start == b.Start
}
