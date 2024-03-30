package main

import (
	"fmt"
	"io"
	"sort"
)

type (
	Process struct {
		ProcessID     string
		ArrivalTime   int64
		BurstDuration int64
		Priority      int64
		Completed     bool
		Turnaround    int
		Waiting       int
		Burst         int
	}
	TimeSlice struct {
		PID   string
		Start int64
		Stop  int64
	}
)

//region Schedulers

// FCFSSchedule outputs a schedule of processes in a GANTT chart and a table of timing given:
// • an output writer
// • a title for the chart
// • a slice of processes
func FCFSSchedule(w io.Writer, title string, processes []Process) {
	var (
		serviceTime     int64
		totalWait       float64
		totalTurnaround float64
		lastCompletion  float64
		waitingTime     int64
		schedule        = make([][]string, len(processes))
		gantt           = make([]TimeSlice, 0)
	)
	for i := range processes {
		if processes[i].ArrivalTime > 0 {
			waitingTime = serviceTime - processes[i].ArrivalTime
		}
		totalWait += float64(waitingTime)

		start := waitingTime + processes[i].ArrivalTime

		turnaround := processes[i].BurstDuration + waitingTime
		totalTurnaround += float64(turnaround)

		completion := processes[i].BurstDuration + processes[i].ArrivalTime + waitingTime
		lastCompletion = float64(completion)

		schedule[i] = []string{
			fmt.Sprint(processes[i].ProcessID),
			fmt.Sprint(processes[i].Priority),
			fmt.Sprint(processes[i].BurstDuration),
			fmt.Sprint(processes[i].ArrivalTime),
			fmt.Sprint(waitingTime),
			fmt.Sprint(turnaround),
			fmt.Sprint(completion),
		}
		serviceTime += processes[i].BurstDuration

		gantt = append(gantt, TimeSlice{
			PID:   processes[i].ProcessID,
			Start: start,
			Stop:  serviceTime,
		})
	}

	count := float64(len(processes))
	aveWait := totalWait / count
	aveTurnaround := totalTurnaround / count
	aveThroughput := count / lastCompletion

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, aveThroughput)
}

func SJFSchedule(w io.Writer, title string, processes []Process) {
	// Step 1: Sort the processes by burst time
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].BurstDuration < processes[j].BurstDuration
	})

	// Use the scheduling logic from FCFSSchedule as a base,
	// since processes are now sorted by burst time,
	// making it effectively an SJF scheduler.
	var (
		currentTime     int64
		totalWait       float64
		totalTurnaround float64
	)
	gantt := make([]TimeSlice, len(processes))
	schedule := make([][]string, len(processes))

	for i, process := range processes {
		waitTime := max(0, currentTime-process.ArrivalTime)
		currentTime = max(currentTime, process.ArrivalTime) + process.BurstDuration
		turnaroundTime := currentTime - process.ArrivalTime

		totalWait += float64(waitTime)
		totalTurnaround += float64(turnaroundTime)

		schedule[i] = []string{
			process.ProcessID,
			fmt.Sprintf("%d", process.Priority),
			fmt.Sprintf("%d", process.BurstDuration),
			fmt.Sprintf("%d", process.ArrivalTime),
			fmt.Sprintf("%d", waitTime),
			fmt.Sprintf("%d", turnaroundTime),
			fmt.Sprintf("%d", currentTime),
		}

		gantt[i] = TimeSlice{
			PID:   process.ProcessID,
			Start: currentTime - process.BurstDuration,
			Stop:  currentTime,
		}
	}

	aveWait := totalWait / float64(len(processes))
	aveTurnaround := totalTurnaround / float64(len(processes))
	throughput := float64(len(processes)) / float64(currentTime)

	outputTitle(w, title)
	outputGantt(w, gantt)
	outputSchedule(w, schedule, aveWait, aveTurnaround, throughput)
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func SJFPrioritySchedule(w io.Writer, title string, processes []Process) {
	fmt.Fprintf(w, "------ %s ------\n", title)

	// Initialize the metrics
	var currentTime int64 = 0
	var totalWait, totalTurnaround float64
	var completed int = 0

	// Pre-sort processes by arrival time to improve efficiency
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].ArrivalTime < processes[j].ArrivalTime
	})

	for completed < len(processes) {
		// Filter processes that have arrived by currentTime and are not completed
		var available []int
		for i, p := range processes {
			if !p.Completed && p.ArrivalTime <= currentTime {
				available = append(available, i)
			}
		}

		// Sort available processes by burst duration, then by priority
		sort.Slice(available, func(i, j int) bool {
			if processes[available[i]].BurstDuration == processes[available[j]].BurstDuration {
				return processes[available[i]].Priority < processes[available[j]].Priority
			}
			return processes[available[i]].BurstDuration < processes[available[j]].BurstDuration
		})

		// No process is ready, increment currentTime
		if len(available) == 0 {
			currentTime++
			continue
		}

		// Select the process to run
		idx := available[0] // Index of the selected process in the original slice
		p := &processes[idx]

		// Calculate wait and turnaround times
		waitTime := currentTime - p.ArrivalTime
		turnaroundTime := waitTime + p.BurstDuration

		// Update metrics
		totalWait += float64(waitTime)
		totalTurnaround += float64(turnaroundTime)
		p.Completed = true // Mark the process as completed
		completed++

		// Move currentTime forward
		currentTime += p.BurstDuration

		// Logging for debug purposes, can be removed or commented out
		fmt.Fprintf(w, "Process %s completed at %d, Wait: %d, Turnaround: %d\n", p.ProcessID, currentTime, waitTime, turnaroundTime)
	}

	// Calculate and print the final metrics
	avgWait := totalWait / float64(len(processes))
	avgTurnaround := totalTurnaround / float64(len(processes))
	throughput := float64(len(processes)) / float64(currentTime)

	fmt.Fprintf(w, "Average wait time: %.2f\n", avgWait)
	fmt.Fprintf(w, "Average turnaround time: %.2f\n", avgTurnaround)
	fmt.Fprintf(w, "Throughput: %.2f processes/unit time\n", throughput)
}

func RRSchedule(w io.Writer, title string, processes []Process) {}

//endregion
