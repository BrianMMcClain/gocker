package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func main() {

	if len(os.Args) < 2 {
		help()
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			help()
		} else {
			run(os.Args[2:])
		}
	}
}

func run(args []string) {
	var cmd *exec.Cmd
	if len(args) == 1 {
		cmd = exec.Command(args[0])
	} else {
		cmd = exec.Command(args[0], strings.Join(args[1:], " "))
	}
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	err := syscall.Sethostname([]byte("container"))
	if err != nil {
		log.Fatal("Could not set hostname")
	}

	err = syscall.Chroot("./fs")
	if err != nil {
		log.Fatal("Could not chroot to image filesystem")
	}

	err = syscall.Chdir("/")
	if err != nil {
		log.Fatal("Could not chdir to the root filesystem")
	}

	err = syscall.Mount("proc", "proc", "proc", 0, "")
	if err != nil {
		log.Fatal("Could not mount the proc filesystem")
	}

	err = cmd.Run()
	if err != nil {
		log.Fatal("Error starting container process:", err)
	}

	syscall.Unmount("proc", 0)
}

func help() {
	log.Fatal("Invalid command or missing parameters")
}
