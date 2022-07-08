package main

import (
	"errors"
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const description = `
Configures environment variables for cross-compiling cgo projects with the Android NDK:
- CGO_ENABLED: 1
- CC: C compiler and flags for relevant ABI and SDK version
- CGO_CFLAGS: Passes -isystem in order to locate header files
- GOOS: android
- GOARCH: Architecture used by go build, mapped from ABI
- GOARM: ARM version, set when needed based on ABI
`

var opts struct {
	Verbose       bool   `short:"v" long:"verbose" description:"Print the env to stdout before running command"`
	ABI           string `short:"a" long:"abi" description:"Android ABI to target, e.g. arm64-v8a" required:"true"`
	NDK           string `long:"ndk" description:"Path to NDK install. Optional, if unspecified then NDK will be located automatically"`
	MinSDKVersion int    `short:"s" long:"min-sdk-version" description:"Minimum android SDK version" required:"true"`
}

func main() {
	parser := flags.NewParser(&opts, flags.Default|flags.IgnoreUnknown)
	parser.Usage = "[-a abi] [-s sdk version] command\nExample:\n  ndkenv -a arm64-v8a -s 21 -- go build ."
	parser.LongDescription = description

	leftoverArgs, err := parser.Parse()
	if err != nil {
		os.Exit(1)
	}
	if len(leftoverArgs) == 0 {
		parser.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	if opts.NDK == "" {
		opts.NDK, err = findNDK(opts.MinSDKVersion)
		if err != nil {
			fmt.Printf("Fatal: Automatically locating NDK: %s\n", err)
			os.Exit(1)
		}
	}

	var cfg abiCfg
	cfg, err = buildCfg(opts.ABI)
	if err != nil {
		fmt.Printf("Fatal: %s", err)
		os.Exit(1)
	}

	// NDK currently only supports x86_64
	// https://developer.android.com/ndk/guides/other_build_systems
	ndkOS := fmt.Sprintf("%s-x86_64", runtime.GOOS)

	toolchain := filepath.Join(opts.NDK, "toolchains", "llvm", "prebuilt", ndkOS)
	sysroot := filepath.Join(toolchain, "sysroot")
	iSystem := filepath.Join(sysroot, "usr", "include", cfg.triple)
	clang := filepath.Join(toolchain, "bin", "clang")

	GOARCH := fmt.Sprintf("GOARCH=%s", cfg.GOARCH)
	GOARM := fmt.Sprintf("GOARM=%s", cfg.GOARM)
	CC := fmt.Sprintf("CC=%s -target %s%d --sysroot=%s",
		clang, cfg.target, opts.MinSDKVersion, sysroot)
	CGO_CFLAGS := fmt.Sprintf("CGO_CFLAGS=-isystem %s/ %s", iSystem, os.Getenv("CGO_CFLAGS"))

	newEnv := []string{"CGO_ENABLED=1", "GOOS=android", GOARCH, GOARM, CC, CGO_CFLAGS}
	if opts.Verbose {
		fmt.Printf("Using env:\n%s\n", strings.Join(newEnv, "\n"))
	}

	cmd := exec.Command(leftoverArgs[0], leftoverArgs[1:]...)
	cmd.Env = append(os.Environ(), newEnv...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	var exitError *exec.ExitError
	if err = cmd.Run(); errors.As(err, &exitError) {
		os.Exit(exitError.ExitCode())
	}
	os.Exit(0)
}

func defaultSdkFolder() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Android", "sdk")
	case "windows":
		return filepath.Join(home, "AppData", "Local", "Android", "Sdk")
	case "linux":
		return filepath.Join(home, "Android", "Sdk")
	default:
		return ""
	}
}

func findNDK(minSdkVersion int) (string, error) {
	// Look for an NDK containing folder in the default Android Studio location
	ndkFolder := filepath.Join(defaultSdkFolder(), "ndk")
	entries, err := os.ReadDir(ndkFolder)
	if err != nil {
		return "", fmt.Errorf("listing %s: %w", ndkFolder, err)
	}
	// Return the first NDK that matches the minSdkVersion, e.g. 21.4.7075529 for "21"
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), strconv.Itoa(minSdkVersion)) {
			return filepath.Join(ndkFolder, entry.Name()), nil
		}
	}
	return "", os.ErrNotExist
}

type abiCfg struct {
	target string
	triple string
	GOARCH string
	GOARM  string
}

//http://android-doc.github.io/ndk/guides/standalone_toolchain.html
func buildCfg(abi string) (abiCfg, error) {
	switch abi {
	case "armeabi-v7a":
		return abiCfg{
			target: "armv7-none-linux-androideabi",
			triple: "armv7a-linux-androideabi",
			GOARCH: "arm",
			GOARM:  "7",
		}, nil
	case "arm64-v8a":
		return abiCfg{
			target: "aarch64-none-linux-android",
			triple: "aarch64-linux-android",
			GOARCH: "arm64",
		}, nil
	case "x86":
		return abiCfg{
			target: "i686-none-linux-android",
			triple: "i686-linux-android",
			GOARCH: "386",
		}, nil
	case "x86-64":
		return abiCfg{
			target: "x86_64-none-linux-android",
			triple: "x86_64-linux-android",
			GOARCH: "amd64",
		}, nil
	default:
		return abiCfg{}, fmt.Errorf("unknown abi: %s", abi)
	}
}
