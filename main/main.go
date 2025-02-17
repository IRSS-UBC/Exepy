package main

import (
	_ "embed"
	"fmt"
	"github.com/maja42/ember"
)

func main() {

	embedded, err := checkIfEmbedded()
	if err != nil {
		fmt.Println("Error checking if embedded:", err)
		return
	}

	if embedded {
		fmt.Println("Project embedded. Running in installer mode.")
		bootstrap()
	} else {
		fmt.Println("Project not embedded. Running in creator mode.")
		createInstaller()
	}

	if isLaunchedFromExplorer() {
		fmt.Print("Press any key to continue...")
		var input string
		fmt.Scanln(&input)
	}
}

func checkIfEmbedded() (bool, error) {

	attachments, err := ember.Open()
	if err != nil {
		fmt.Println("Error opening attachments:", err)
		return false, err
	}
	defer attachments.Close()

	attachmentList := attachments.List()

	if len(attachmentList) == 0 {
		return false, nil
	} else {
		return true, nil
	}
}
