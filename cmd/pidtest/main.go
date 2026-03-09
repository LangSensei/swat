package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func main() {
	cmd := exec.Command("sleep", "30")
	logFile, _ := os.Create("/tmp/pidtest2.log")
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	cmd.Start()
	pid := cmd.Process.Pid
	fmt.Printf("child PID: %d\n", pid)

	go func() {
		defer logFile.Close()
		err := cmd.Wait()
		fmt.Printf("[wait] returned: %v (%s)\n", err, time.Now().Format("15:04:05"))
	}()

	for i := 0; i < 8; i++ {
		time.Sleep(5 * time.Second)

		// Method 1: Signal(nil)
		proc, _ := os.FindProcess(pid)
		errNil := proc.Signal(nil)

		// Method 2: Signal(syscall.Signal(0))
		proc2, _ := os.FindProcess(pid)
		errSig0 := proc2.Signal(syscall.Signal(0))

		// Method 3: syscall.Kill(pid, 0)
		errKill := syscall.Kill(pid, 0)

		fmt.Printf("[%s] Signal(nil)=%-30v  Signal(0)=%-30v  Kill(0)=%v\n",
			time.Now().Format("15:04:05"), errNil, errSig0, errKill)
	}
}
