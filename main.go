package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/JackMordaunt/icns"
	"github.com/pkg/errors"
)

const (
	E_ARGS = iota
	E_ERROR
)

type infoListData struct {
	Name               string
	Executable         string
	Identifier         string
	Version            string
	InfoString         string
	ShortVersionString string
	IconFile           string
}

var (
	name       = flag.String("name", "My Go Application", "app name")
	author     = flag.String("author", "Appify by Machine Box", "author")
	version    = flag.String("version", "1.0", "app version")
	identifier = flag.String("id", "", "bundle identifier")
	icon       = flag.String("icon", "", "icon image file (.icns|.png|.jpg|.jpeg)")
	distDir    = flag.String("dist", ".", "directory in which to build App")
)

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprint(os.Stderr, errors.New("missing executable argument"))

		os.Exit(E_ARGS)
	}

	bin := args[0]

	err := run(bin)
	if err != nil {
		fmt.Fprint(os.Stderr, err)

		os.Exit(E_ERROR)
	}
}

func run(bin string) error {
	appName := fmt.Sprintf("%s.app", *name)

	basePath := filepath.Join(*distDir, appName)
	contentsPath := filepath.Join(basePath, "Contents")
	appPath := filepath.Join(contentsPath, "MacOS")
	resouresPath := filepath.Join(contentsPath, "Resources")
	binPath := filepath.Join(appPath, appName)

	if err := os.MkdirAll(appPath, 0777); err != nil {
		return errors.Wrap(err, "os.MkdirAll appPath")
	}

	fdst, err := os.Create(binPath)
	if err != nil {
		return errors.Wrap(err, "create bin")
	}

	defer fdst.Close()

	fsrc, err := os.Open(bin)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New(bin + " not found")
		}
		return errors.Wrap(err, "os.Open")
	}

	defer fsrc.Close()

	if _, err := io.Copy(fdst, fsrc); err != nil {
		return errors.Wrap(err, "copy bin")
	}

	if err := exec.Command("chmod", "+x", appPath).Run(); err != nil {
		return errors.Wrap(err, "chmod: "+appPath)
	}

	if err := exec.Command("chmod", "+x", binPath).Run(); err != nil {
		return errors.Wrap(err, "chmod: "+binPath)
	}

	id := *identifier
	if id == "" {
		id = *author + "." + *name
	}

	info := infoListData{
		Name:               *name,
		Executable:         filepath.Join("MacOS", appName),
		Identifier:         id,
		Version:            *version,
		InfoString:         *name + " by " + *author,
		ShortVersionString: *version,
	}

	if *icon != "" {
		iconPath, err := prepareIcons(*icon, resouresPath)
		if err != nil {
			return errors.Wrap(err, "icon")
		}
		info.IconFile = filepath.Base(iconPath)
	}

	tpl, err := template.New("template").Parse(infoPlistTemplate)
	if err != nil {
		return errors.Wrap(err, "infoPlistTemplate")
	}

	fplist, err := os.Create(filepath.Join(contentsPath, "Info.plist"))
	if err != nil {
		return errors.Wrap(err, "create Info.plist")
	}

	defer fplist.Close()

	if err := tpl.Execute(fplist, info); err != nil {
		return errors.Wrap(err, "execute Info.plist template")
	}

	if err := ioutil.WriteFile(filepath.Join(contentsPath, "README"), []byte(readme), 0666); err != nil {
		return errors.Wrap(err, "ioutil.WriteFile")
	}

	return nil
}

func prepareIcons(iconPath, resourcesPath string) (string, error) {
	ext := filepath.Ext(strings.ToLower(iconPath))

	fsrc, err := os.Open(iconPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("icon file not found")
		}
		return "", errors.Wrap(err, "open icon file")
	}

	defer fsrc.Close()

	if err := os.MkdirAll(resourcesPath, 0777); err != nil {
		return "", errors.Wrap(err, "os.MkdirAll resourcesPath")
	}

	destFile := filepath.Join(resourcesPath, "icon.icns")
	fdst, err := os.Create(destFile)
	if err != nil {
		return "", errors.Wrap(err, "create icon.icns file")
	}

	defer fdst.Close()

	switch ext {
	case ".icns": // just copy the .icns file
		_, err := io.Copy(fdst, fsrc)
		if err != nil {
			return destFile, errors.Wrap(err, "copying "+iconPath)
		}
	case ".png", ".jpg", ".jpeg", ".gif": // process any images
		srcImg, _, err := image.Decode(fsrc)
		if err != nil {
			return destFile, errors.Wrap(err, "decode image")
		}
		if err := icns.Encode(fdst, srcImg); err != nil {
			return destFile, errors.Wrap(err, "generate icns file")
		}
	default:
		return destFile, errors.New(ext + " icons not supported")
	}

	return destFile, nil
}

// readme goes into a README file inside the package for
// future reference.
const readme = `Made with Appify by Machine Box
https://github.com/machinebox/appify

Inspired by https://gist.github.com/anmoljagetia/d37da67b9d408b35ac753ce51e420132 
`
