package checks

import (
	"fmt"
	"net/http"
	"time"
)

// A simple function to check if the device can connect to Github (where the packages are hosted)
func CheckConnectivity() error {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	_, err := client.Head("https://github.com")
	if err != nil {
		return fmt.Errorf("Can't access github")
	}
	return nil
}
