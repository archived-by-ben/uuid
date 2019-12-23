package logs

import "log"

// Check logs any errors and exits to the operating system with error code 1.
func Check(err error) {
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
}

// Log any errors.
func Log(err error) {
	if err != nil {
		log.Printf("! %v", err)
	}
}
