package utility

import (
	"fmt"
	"log"
	"runtime"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
)

// checks if the process is active
// returns a bool representing the outcome checking if the process is alive
// where true represents an active process,
// while false represents an unresponsive process
func IsProcessActive(pid int) bool {
	err := syscall.Kill(pid, syscall.Signal(0))

	return err == nil
}

// terminates the process gracefully and forcefully if it does not respond
// returns a bool,
// where true represents a shutdown of the process,
// and false represenst a failure to terminate the process.
func TerminateProcess(pid int) bool {
	if !IsProcessActive(pid) {
		log.Printf("Process PID %d is not active.\n", pid)
		return false
	}
	pidgrp := -pid

	log.Printf("Sending SIGTERM to PID %d...\n", pid)
	err := syscall.Kill(pidgrp, syscall.SIGTERM)
	if err != nil {
		log.Printf("Failed to send SIGTERM to PID %d: %v\n", pid, err)
	}

	// Allow some time for the process to terminate gracefully
	time.Sleep(3 * time.Second)

	if IsProcessActive(pid) {
		// If the process is still active, force termination
		log.Printf("Process PID %d did not terminate after SIGTERM; sending SIGKILL...\n", pid)
		err = syscall.Kill(pidgrp, syscall.SIGKILL)
		if err != nil {
			log.Printf("Failed to send SIGKILL to PID %d: %v\n", pid, err)
			return false
		} else {
			log.Printf("Successfully sent SIGKILL to PID %d.\n", pid)
		}
	} else {
		log.Printf("Process PID %d terminated successfully after SIGTERM.\n", pid)
	}

	log.Printf("Process PID %d terminated successfully.\n", pid)
	return true
}

// retrieves the process usage from a pid
func GetProcessUsageFromPid(pid int) (string, string, string, error) {
	
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		log.Printf("Failed to get process: %s\n", err)
		return "", "", "", fmt.Errorf("failed to get process: %s", err)
	}

	cpuPercent, err := proc.Percent(time.Second)
	if err != nil {
		log.Printf("Failed to get CPU usage: %s\n", err)
		return "", "", "", fmt.Errorf("failed to get CPU usage: %s", err)
	}

	memInfo, err := proc.MemoryInfo()
	if err != nil {
		log.Printf("Failed to get memory information: %s\n", err)
		return "", "", "", fmt.Errorf("failed to get memory information: %s", err)
	}

	vmem, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Failed to get system memory info: %s\n", err)
		return "", "", "", fmt.Errorf("failed to get system memory info: %s", err)
	}

	cpuUsage := cpuPercent / float64(runtime.NumCPU()) // Normalize CPU usage by number of CPUs
	memUsagePercent := float64(memInfo.RSS) / float64(vmem.Total) * 100


	return fmt.Sprintf("%.2f%%", cpuPercent), fmt.Sprintf("%.2f%%", cpuUsage), fmt.Sprintf("%.2f%%", memUsagePercent), nil
}

func GetGeneralUsage() (string, string, error){
	// Get CPU usage percentage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		fmt.Printf("Error getting CPU usage: %s\n", err)
		return "", "", fmt.Errorf("error getting CPU usage: %s", err)
	}

	// Get virtual memory usage
	vmem, err := mem.VirtualMemory()
	if err != nil {
		fmt.Printf("Error getting memory usage: %s\n", err)
		return "", "", fmt.Errorf("error getting memory usage: %s", err)
	}

	return fmt.Sprintf("%.2f%%", cpuPercent[0]), fmt.Sprintf("%.2f%%", vmem.UsedPercent), nil
}