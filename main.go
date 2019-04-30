package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/harrisonthorne/go-sway"
)

var globalWallpaper string
var blurAmount = "0x2"
var blurBools = make(map[string]bool)
var mtx = &sync.Mutex{}

func main() {
	outputWalls := make(map[string]string)

	// TODO check if the program is already running and
	// update it

	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	wayblDir := homeDir + "/.waybl"
	println(wayblDir)

	os.Mkdir(wayblDir, os.ModeDir|0755)

	// run through all of the arguments
	for i := 0; i < len(os.Args); i++ {
		if i == 0 {
			continue
		}

		arg := os.Args[i]

		switch arg {
		case "-b", "--blur":
			blurAmount = os.Args[i+1]
			i++
			continue
		}

		// if we have arguments such as "output-name:./wallpaper/image.png",
		// then we assign those to a map. otherwise, we
		// assume a global wallpaper argument has been
		// passed in. output-specific arguments take
		// precedence. the global wallpaper will not take
		// effect at all if output-specific wallpapers are
		// passed in.
		if strings.Contains(arg, ":") {
			split := strings.Split(arg, ":")
			outputWalls[split[0]] = os.ExpandEnv(split[1])
		} else {
			globalWallpaper = arg
		}

		makeBlurs(wayblDir, outputWalls)
	}

	// init
	checkEntireTree(wayblDir, outputWalls)

	// listen for window updates
	println("Listening...")
	swayEvents := sway.Subscribe(sway.WindowEventType, sway.WorkspaceEventType)
	for swayEvents.Next() {
		ev := swayEvents.Event()
		switch ev.(type) {
		case *sway.WindowEvent:
			windowEv := ev.(*sway.WindowEvent)
			if windowEv.Change != "title" {
				checkEntireTree(wayblDir, outputWalls)
			}
		case *sway.WorkspaceEvent:
			checkEntireTree(wayblDir, outputWalls)
		}
	}
}

func makeBlurs(dir string, outputWalls map[string]string) {
	for output, path := range outputWalls {
		go makeSingleBlur(dir, output, path)
	}
}

func makeSingleBlur(dir, output, wallpaperPath string) {
	err := exec.Command(
        // TODO get geometry of output
		"convert", wallpaperPath,
		"-geometry", "1920x1080^",
		"-gravity", "center",
		"-crop", "1920x1080+0+0",
		"-resize", "5%",
		"-blur", blurAmount,
		"-resize", "1000%",
		getBlurredWallpaperPath(output, dir),
	).Run()

	if err != nil {
		panic(err.Error())
	}
}

func setWallpaper(output, wallpaperPath string) {
	println("Setting wallpaper of " + output + " to " + wallpaperPath)

	retriesLeft := 5

	err := fmt.Errorf("no attempt at setting wallpaper yet")
	for err != nil && retriesLeft > 0 {
		cmd := exec.Command("swaymsg", "output", output, "bg", "\""+wallpaperPath+"\"", "fill", "#000000")
		cmdOutput, err := cmd.Output()
		println("cmd output: " + string(cmdOutput))

		if err != nil {
			println(err.Error())
			println("Retrying setWallpaper for " + output)
			time.Sleep(2 * time.Second)
			retriesLeft--

			// if error is still present after the set
			// amount of retries, give up
			if retriesLeft <= 0 {
				panic("wallpaper couldn't be set after 5 retries")
			}
			// continue
		} else {
			break
		}
	}
}

// also sets wallpaper
func setBlur(output string, newBlur bool, wayblDir string, outputWalls map[string]string) {
	if (blurBools[output] && newBlur) || (!blurBools[output] && !newBlur) {
		return
	}

	blurBools[output] = newBlur

	if newBlur {
		// set all wallpapers to blur
		println("blur on " + output + " is now on")
		setWallpaper(output, getBlurredWallpaperPath(output, wayblDir))
	} else {
		// set all wallpapers back to normal
		println("blur on " + output + " now off")
		setWallpaper(output, outputWalls[output])
	}
}

func getBlurredWallpaperPath(output string, wayblDir string) string {
	return wayblDir + "/" + output + ".jpg"
}

// returns true if a node in the tree is found to be focused
func isDescendantFocused(root *sway.Node) bool {
	switch root.Type {
	case sway.Con, sway.FloatingCon:
		// stop when we find a visible node
		if root.Visible {
			println("Visible node was:", root.Name)
			return true
		}
	}

	// recursive traversal if no visible nodes were found
	for _, n := range root.Nodes {
		if isDescendantFocused(n) {
			return true
		}
	}

	return false
}

func checkEntireTree(wayblDir string, outputWalls map[string]string) {
	tree, _ := sway.GetTree()
	checkOutputs(tree.Root.Nodes, wayblDir, outputWalls)
}

func checkOutputs(outputs []*sway.Node, wayblDir string, outputWalls map[string]string) {
	for _, o := range outputs {
		println("Checking " + o.Name)
		if o.Type == sway.OutputNode {
			go setBlur(o.Name, isDescendantFocused(o), wayblDir, outputWalls)
		}
	}
}