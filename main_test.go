package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
)

var (
	derivedAppName string
)

type file struct {
	path string
	perm string
	hash string
}

func TestRun(t *testing.T) {
	for _, test := range []struct {
		name            string
		executable      string
		distDir         string
		appName         string
		author          string
		version         string
		identifier      string
		icon            string
		expectError     bool
		expectPlistHash string
	}{
		// Non erroring tests
		{"Happy pathed app with all vars", "testdata/app", "testdata/dist", "Test", "a. gopher", "0.0.1", "test", "testdata/machina-square.png", false, "26201bfbb9eb0c2ed207f7835641bb4d"},
		{"Missing/ empty ID", "testdata/app", "testdata/dist", "Test", "a. gopher", "0.0.1", "", "testdata/machina-square.png", false, "4b132bc552075589ce3db6a3ea9afe57"},

		// Tests which return error(s)
		{"No such binary", "testdata/nonsuch", "testdata/dist", "", "", "", "", "", true, ""},
		{"Incorrect, pre-existing app directory", "testdata/app", "testdata/dist", "TestTwo", "", "", "", "", true, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			derivedAppName = filepath.Join(test.distDir, fmt.Sprintf("%s.app", test.appName))

			flag.Set("name", test.appName)
			flag.Set("author", test.author)
			flag.Set("version", test.version)
			flag.Set("id", test.identifier)
			flag.Set("icon", test.icon)
			flag.Set("dist", test.distDir)

			err := run(test.executable)

			if err != nil && !test.expectError {
				t.Errorf("unexpected error: %+v", err)
			}

			if err == nil {
				if test.expectError {
					t.Errorf("expected error")
				} else {
					defer os.RemoveAll(derivedAppName)

					actualAppHash := filehash(t, test.executable)

					for _, f := range []struct {
						path string
						perm string
						hash string
					}{
						{path: derivedAppName, perm: "drwxr-xr-x"},
						{path: derivedAppName + "/Contents", perm: "drwxr-xr-x"},
						{path: derivedAppName + "/Contents/MacOS", perm: "drwxr-xr-x"},
						{path: derivedAppName + "/Contents/MacOS/Test.app", perm: "-rwxr-xr-x", hash: actualAppHash},
						{path: derivedAppName + "/Contents/Info.plist", perm: "-rw-r--r--", hash: test.expectPlistHash},
						{path: derivedAppName + "/Contents/README", perm: "-rw-r--r--", hash: "afeb10df47c7f189b848ae44a54e7e06"},
						{path: derivedAppName + "/Contents/Resources", perm: "drwxr-xr-x"},
						{path: derivedAppName + "/Contents/Resources/icon.icns", perm: "-rw-r--r--"},
					} {
						t.Run(f.path, func(t *testing.T) {
							is := is.New(t)

							info, err := os.Stat(f.path)

							is.NoErr(err)
							is.Equal(info.Mode().String(), f.perm) // perm

							if f.hash != "" {
								actual := filehash(t, f.path)
								is.Equal(actual, f.hash) // hash
							}
						})
					}
				}
			}
		})
	}
}

// filehash gets an md5 hash of the file at path.
func filehash(t *testing.T, path string) string {
	is := is.New(t)

	f, err := os.Open(path)
	is.NoErr(err)

	defer f.Close()

	h := md5.New()

	_, err = io.Copy(h, f)
	is.NoErr(err)

	return fmt.Sprintf("%x", h.Sum(nil))
}
