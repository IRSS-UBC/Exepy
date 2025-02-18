package main

import (
	_ "embed"
	"fmt"
	"github.com/maja42/ember"
	"os"
)

func main() {

	defer func() {
		// Check if a panic occurred.
		code := 0

		if r := recover(); r != nil {
			fmt.Println("Panic:", r)
			code = 1
		}

		// Pause if launched from Explorer.
		if isLaunchedFromExplorer() {
			fmt.Print("Press any key to continue...")
			var input string
			// Wait for user input before exiting.
			fmt.Scanln(&input)

			os.Exit(code)
		}
	}()

	embedded, err := checkIfEmbedded()
	if err != nil {
		fmt.Println("Error checking if embedded:", err)
		return
	}

	if embedded {
		fmt.Println("Installing Python script...")
		err := bootstrap()
		if err != nil {
			panic(err)
		}
	} else {
		PrintHeader() // Only print the header if we're in creator mode.
		// Not needed for installer as we don't want to be intrusive.

		fmt.Println("No scripts have been embedded. Running in creator mode.")
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

func PrintHeader() {
	header := `
 _____        ______      
|  ___|       | ___ \     
| |____  _____| |_/ /   _ 
|  __\ \/ / _ \  __/ | | |
| |___>  <  __/ |  | |_| |
\____/_/\_\___\_|   \__, |
                     __/ |
                    |___/ 
`
	fmt.Println(header)
}
