// Example window-names fetches a list of all top-level client windows managed
// by the currently running window manager, and prints the name and size
// of each window.
//
// This example demonstrates how to use some aspects of the ewmh and icccm
// packages. It also shows how to use the xwindow package to find the
// geometry of a client window. In particular, finding the geometry is
// intelligent, as it includes the geometry of the decorations if they exist.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os/exec"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/xprop"
	"github.com/alexflint/go-arg"
)

var args struct {
	Chooser string `arg:"positional"`
	Print   bool   `arg:"-p" help:"Print the id value instead of setting _NET_ACTIVE_WINDOW"`
}

func main() {
	arg.MustParse(&args)
	// Connect to the X server using the DISPLAY environment variable.
	X, err := xgbutil.NewConn()
	if err != nil {
		log.Fatal(err)
	}

	// Get a list of all client ids.
	clientids, err := ewmh.ClientListGet(X)
	if err != nil {
		log.Fatal(err)
	}

	var id_to_window map[string]xproto.Window = make(map[string]xproto.Window)
	names := ""
	// Iterate through each client, find its name
	for _, clientid := range clientids {
		name, err := WmGetClass(X, clientid)
		if err != nil {
			panic(err)
		} else {
			id_to_window[name] = clientid
			names += name + "\n"
		}
	}
	command := exec.Command(args.Chooser)

	stdin, err := command.StdinPipe()

	if err != nil {
		panic(err)
	}

	stdin.Write([]byte(names))
	stdin.Close()
	output, err := command.Output()

	if err != nil {
		switch e := err.(type) {
		case *exec.Error:
			fmt.Println("failed executing:", err)
		case *exec.ExitError:
			fmt.Println("command exit rc =", e.ExitCode())
		default:
			panic(err)
		}
	}
	to_focus := id_to_window[string(output)[:len(output)-1]]
	if args.Print {
		fmt.Println(to_focus)
	}
	if err := ewmh.ActiveWindowReq(X, to_focus); err != nil {
		panic(err)
	}
}
func WmGetClass(xu *xgbutil.XUtil, win xproto.Window) (full_name string, err error) {
	// Get the WM_CLASS property
	prop, err := xprop.GetProperty(xu, win, "WM_CLASS")
	if err != nil {
		return "", err
	}

	name_win, err := ewmh.WmNameGet(xu, win)
	if err != nil {
		return "", err
	}

	// Split the property value into instance and class name
	values := bytes.Split(prop.Value, []byte{0})
	if len(values) >= 2 {
		// Return the class name (application name)
		return string(values[1]) + " - " + name_win, nil
	}
	return "", errors.New("WM_CLASS property does not contain expected values")
}
