package common

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func PressButtonToContinue(continueMessage string) {
	fmt.Println(continueMessage)
	fmt.Println(".")
	fmt.Print("\a")

	stop := make(chan bool)

	go func() {
		animation := []string{" ", " ", " ", "o", "O", "o", " ", " ", " "}
		i := 0
		for {
			select {
			case <-stop:
				fmt.Printf("\r%s", strings.Repeat(" ", len(strings.Join(animation, ""))))
				fmt.Print("\r")
				return
			default:
				fmt.Printf("\r%s", strings.Join(animation, ""))
				time.Sleep(100 * time.Millisecond)
				animation = append(animation[1:], animation[0])
				i++
				if i == len(animation) {
					i = 0
					animation = []string{" ", " ", " ", "o", "O", "o", " ", " ", " "}
				}
			}
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')

	stop <- true
}
