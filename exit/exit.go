package exit

import (
	"os"
	"fmt"
)

type Exit struct{}

func (*Exit) Exit(status int) {
	fmt.Printf("Exiting with status code %d", status)
	os.Exit(status)
}
