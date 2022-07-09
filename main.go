package main

import (
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {

	if len(os.Args) < 3 {
		help()
	}

	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			help()
		} else {
			run(os.Args[2], os.Args[3:])
		}
	case "child":
		child(os.Args[2], os.Args[3:])
	}
}

func run(image string, args []string) {
	cmd := exec.Command("/proc/self/exe", append([]string{"child", image}, args[0:]...)...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	cmd.Run()
}

func child(image string, args []string) {
	log.Println("Running ", args, " on image ", image)

	homeDir, _ := os.UserHomeDir()
	gockerDir := filepath.Join(homeDir, ".gocker")
	// Create the gocker directory if it doesn't exist
	if _, err := os.Stat(gockerDir); err != nil {
		os.Mkdir(gockerDir, 0755)
	}
	fsDir, manifestConfig := DownloadImage(image, gockerDir)

	containerID := generateContainerID()
	setCGroup(containerID)

	err := syscall.Sethostname([]byte(containerID))
	if err != nil {
		log.Fatal("Could not set hostname")
	}

	err = syscall.Chroot(fsDir)
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

	// If the image manifest defines a working directory,
	// chdir to it
	if len(manifestConfig.WorkingDir) > 0 {
		err = os.Chdir(manifestConfig.WorkingDir)
		if err != nil {
			log.Fatal("Could not chdir to workingDir")
		}
	}

	for _, c := range manifestConfig.Env {
		sEnv := strings.SplitN(c, "=", 2)
		os.Setenv(sEnv[0], sEnv[1])
	}

	var cmdOrEntrypoint []string
	if len(args) > 0 {
		cmdOrEntrypoint = args
	} else if len(manifestConfig.Cmd) > 0 {
		cmdOrEntrypoint = manifestConfig.Cmd
	} else if len(manifestConfig.Entrypoint) > 0 {
		cmdOrEntrypoint = manifestConfig.Entrypoint
	} else {
		log.Fatal("No command provided")
	}

	var cmd *exec.Cmd
	if len(cmdOrEntrypoint) >= 2 {
		cmd = exec.Command(cmdOrEntrypoint[0], cmdOrEntrypoint[1:]...)
	} else {
		cmd = exec.Command(cmdOrEntrypoint[0])
	}

	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	log.Println("Starting container", containerID)
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
