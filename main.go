package main

import (
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
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
	case "child":
		child(os.Args[2:])
	}
}

func run(args []string) {
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, args[0:]...)...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	cmd.Run()
}

func child(args []string) {
	containerID := generateContainerID()
	setCGroup(containerID)

	err := syscall.Sethostname([]byte(containerID))
	if err != nil {
		log.Fatal("Could not set hostname")
	}

	err = syscall.Chroot("./fs")
	if err != nil {
		log.Fatal("Could not chroot to image filesystem")
	}

	err = os.Chdir("/")
	if err != nil {
		log.Fatal("Could not chdir to the root filesystem")
	}

	err = syscall.Mount("proc", "proc", "proc", 0, "")
	if err != nil {
		log.Fatal("Could not mount the proc filesystem")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Run()

	syscall.Unmount("proc", 0)
}

func setCGroup(id string) {
	cgroups := "/sys/fs/cgroup/"
	pids := filepath.Join(cgroups, "pids")
	os.Mkdir(pids, 0755) // Make the pids dir if it doesn't exist
	os.Mkdir(filepath.Join(pids, id), 0755)
	ioutil.WriteFile(filepath.Join(pids, id, "notify_on_release"), []byte("1"), 0700)
	ioutil.WriteFile(filepath.Join(pids, id, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700)
}

func generateContainerID() string {
	rand.Seed(time.Now().UnixNano())
	vals := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	id := make([]byte, 32)
	for i := range id {
		id[i] = vals[rand.Intn(len(vals))]
	}
	return string(id)
}

func help() {
	log.Fatal("Invalid command or missing parameters")
}
