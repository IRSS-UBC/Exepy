package main

import (
	_ "embed"
	"fmt"
	"github.com/maja42/ember"
	"lukasolson.net/common"
)

func main() {

	embedded, err := checkIfEmbedded()
	if err != nil {
		fmt.Println("Error checking if embedded:", err)
		return
	}

	if embedded {
		fmt.Println("Project embedded. Running in installer mode.")
		bootstrap(common.PureMode)
	} else {
		fmt.Println("Project not embedded. Running in creator mode.")
		createInstaller()
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
