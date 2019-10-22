package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
)

var (
	debugflag  = flag.Bool("debug", false, "enable debug output")
	configflag = flag.String("config", "", "configuration file")

	defaultPATH = []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"}
)

func exit() {
	if err := recover(); err != nil {
		if _, ok := err.(runtime.Error); ok {
			log.Println(err)
		}
		if *debugflag {
			log.Printf("%s", debug.Stack())
		}
		os.Exit(1)
	}
	os.Exit(0)
}

func getPATH(config CLIConfig) string {
	dirs := filepath.SplitList(os.Getenv("PATH"))
	// directories from the hook environment have higher precedence
	dirs = append(dirs, defaultPATH...)

	if config.Root != nil {
		rootDirs := []string{}
		for _, dir := range dirs {
			rootDirs = append(rootDirs, path.Join(*config.Root, dir))
		}
		// directories with the root prefix have higher precedence
		dirs = append(rootDirs, dirs...)
	}
	return strings.Join(dirs, ":")
}

func getCLIPath(config CLIConfig) string {
	if config.Path != nil {
		return *config.Path
	}

	if err := os.Setenv("PATH", getPATH(config)); err != nil {
		log.Panicln("couldn't set PATH variable:", err)
	}

	path, err := exec.LookPath("nvidia-container-cli")
	if err != nil {
		log.Panicln("couldn't find binary nvidia-container-cli in", os.Getenv("PATH"), ":", err)
	}
	return path
}

// getRootfsPath returns an absolute path. We don't need to resolve symlinks for now.
func getRootfsPath(config containerConfig) string {
	rootfs, err := filepath.Abs(config.Rootfs)
	if err != nil {
		log.Panicln(err)
	}
	return rootfs
}

func doPrestart() {
	var err error

	defer exit()
	log.SetFlags(0)

	hook := getHookConfig()
	cli := hook.NvidiaContainerCLI

	container := getContainerConfig(hook)
	nvidia := container.Nvidia
	if nvidia == nil {
		// Not a GPU container, nothing to do.
		return
	}

	rootfs := getRootfsPath(container)

	args := []string{getCLIPath(cli)}
	if cli.Root != nil {
		args = append(args, fmt.Sprintf("--root=%s", *cli.Root))
	}
	if cli.LoadKmods {
		args = append(args, "--load-kmods")
	}
	if *debugflag {
		args = append(args, "--debug=/dev/stderr")
	} else if cli.Debug != nil {
		args = append(args, fmt.Sprintf("--debug=%s", *cli.Debug))
	}
	if cli.Ldcache != nil {
		args = append(args, fmt.Sprintf("--ldcache=%s", *cli.Ldcache))
	}
	if cli.User != nil {
		args = append(args, fmt.Sprintf("--user=%s", *cli.User))
	}
	args = append(args, "configure")

	if cli.Ldconfig != nil {
		args = append(args, fmt.Sprintf("--ldconfig=%s", *cli.Ldconfig))
	}
	if cli.NoCgroups {
		args = append(args, "--no-cgroups")
	}
	if len(nvidia.Devices) > 0 {
		args = append(args, fmt.Sprintf("--device=%s", nvidia.Devices))
	}

	for _, cap := range strings.Split(nvidia.Capabilities, ",") {
		if len(cap) == 0 {
			break
		}
		args = append(args, capabilityToCLI(cap))
	}

	if !hook.DisableRequire && !nvidia.DisableRequire {
		for _, req := range nvidia.Requirements {
			args = append(args, fmt.Sprintf("--require=%s", req))
		}
	}

	args = append(args, fmt.Sprintf("--pid=%s", strconv.FormatUint(uint64(container.Pid), 10)))
	args = append(args, rootfs)

	env := append(os.Environ(), cli.Environment...)
	err = syscall.Exec(args[0], args, env)
	log.Panicln("exec failed:", err)
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nCommands:\n")
	fmt.Fprintf(os.Stderr, "  prestart\n        run the prestart hook\n")
	fmt.Fprintf(os.Stderr, "  poststart\n        no-op\n")
	fmt.Fprintf(os.Stderr, "  poststop\n        no-op\n")
}

func main() {
	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(2)
	}

	switch args[0] {
	case "prestart":
		doPrestart()
		os.Exit(0)
	case "poststart":
		fallthrough
	case "poststop":
		os.Exit(0)
	default:
		flag.Usage()
		os.Exit(2)
	}
}
