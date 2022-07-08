# ndkenv - Configures environment variables for cross-compiling cgo projects with the Android NDK

## Installing:
```
go install github.com/iamcalledrob/ndkenv@latest
```

## Usage:
```
Usage:
  ndkenv [-a abi] [-s sdk version] command
Example:
  ndkenv -a arm64-v8a -s 21 -- go build .

Configures environment variables for cross-compiling cgo projects with the Android NDK:
- CGO_ENABLED: 1
- CC: C compiler and flags for relevant ABI and SDK version
- CGO_CFLAGS: Passes -isystem in order to locate header files
- GOOS: android
- GOARCH: Architecture used by go build, mapped from ABI
- GOARM: ARM version, set when needed based on ABI

Application Options:
  -v, --verbose          Print the env to stdout before running command
  -a, --abi=             Android ABI to target, e.g. arm64-v8a
      --ndk=             Path to NDK install. Optional, if unspecified then NDK will be located automatically
  -s, --min-sdk-version= Minimum android SDK version
```