package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

var (
	verbose = flag.Bool("v", false, "Show verbose logging")
	binary  = flag.String("D", "", "Path of the elf file to load")
	version = "0.0.0-dev" // CI will take care of it
)

func PrintlnVerbose(a ...interface{}) {
	if *verbose {
		fmt.Println(a...)
	}
}

func PrintVerbose(a ...interface{}) {
	if *verbose {
		fmt.Print(a...)
	}
}

func executeCommand(verbose_description string, command []string, print_output bool, show_spinner bool, fatal bool) error {

	if verbose_description != "" {
		PrintVerbose(verbose_description)
	}

	err, _, exe_output := launchCommandAndWaitForOutput(command, print_output, show_spinner)

	if err != nil {
		if fatal {
			PrintlnVerbose(exe_output)
			os.Exit(1)
		}
	} else if verbose_description != "" {
		PrintlnVerbose(" OK")
	}

	return err
}

func main() {
	name := filepath.Base(os.Args[0])
	path, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	flag.Parse()

	PrintlnVerbose(name + " " + version + " - compiled with " + runtime.Version())

	convert := []string{filepath.Join(path, "elf2uf2"), *binary, *binary + ".uf2"}

	executeCommand("Converting elf to uf2 ...", convert, false, false, true)

	info := []string{filepath.Join(path, "picotool"), "info"}
	err := executeCommand("Looking for RP2040 device in BOOTSEL mode ", info, false, true, false)

	max_attempts := 20

	for i := 0; i < max_attempts && err != nil; i++ {
		time.Sleep(500 * time.Millisecond)
		err = executeCommand("", info, false, true, i == (max_attempts-1))
	}

	load := []string{filepath.Join(path, "picotool"), "load", *binary + ".uf2"}
	executeCommand("", load, true, false, true)

	reboot := []string{filepath.Join(path, "picotool"), "reboot"}
	executeCommand("Rebooting RP2040 device ...", reboot, false, false, true)

	fmt.Println("")
	os.Exit(0)
}

func launchCommandAndWaitForOutput(command []string, print_output bool, show_spinner bool) (error, bool, string) {
	oscmd := exec.Command(command[0], command[1:]...)
	tellCommandNotToSpawnShell(oscmd)
	stdout, _ := oscmd.StdoutPipe()
	stderr, _ := oscmd.StderrPipe()
	multi := io.MultiReader(stdout, stderr)

	if print_output && *verbose {
		oscmd.Stdout = os.Stdout
		oscmd.Stderr = os.Stderr
	}
	err := oscmd.Start()
	if err != nil {
		return err, false, ""
	}
	in := bufio.NewScanner(multi)
	in.Split(bufio.ScanRunes)
	found := false
	out := ""
	if show_spinner {
		fmt.Printf(".")
	}
	lastPrint := time.Now()
	for in.Scan() {
		if show_spinner && time.Since(lastPrint) > time.Second {
			fmt.Printf(".")
			lastPrint = time.Now()
		}
		out += in.Text()
	}
	err = oscmd.Wait()
	return err, found, out
}
