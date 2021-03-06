package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the rivine daemon",
		Long:  "Stop the rivine daemon.",
		Run:   wrap(stopcmd),
	}

	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update rivine",
		Long:  "Check for (and/or download) available updates for rivine.",
		Run:   wrap(updatecmd),
	}

	updateCheckCmd = &cobra.Command{
		Use:   "check",
		Short: "Check for available updates",
		Long:  "Check for available updates.",
		Run:   wrap(updatecheckcmd),
	}
)

type updateInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
}

// stopcmd is the handler for the command `siac stop`.
// Stops the daemon.
func stopcmd() {
	err := get("/daemon/stop")
	if err != nil {
		die("Could not stop daemon:", err)
	}
	fmt.Println("rivine daemon stopped.")
}

func updatecmd() {
	var update updateInfo
	err := getAPI("/daemon/update", &update)
	if err != nil {
		fmt.Println("Could not check for update:", err)
		return
	}
	if !update.Available {
		fmt.Println("Already up to date.")
		return
	}

	err = post("/daemon/update", "")
	if err != nil {
		fmt.Println("Could not apply update:", err)
		return
	}
	fmt.Printf("Updated to version %s! Restart rivined now.\n", update.Version)
}

func updatecheckcmd() {
	var update updateInfo
	err := getAPI("/daemon/update", &update)
	if err != nil {
		fmt.Println("Could not check for update:", err)
		return
	}
	if update.Available {
		fmt.Printf("A new release (v%s) is available! Run 'rivinec update' to install it.\n", update.Version)
	} else {
		fmt.Println("Up to date.")
	}
}
