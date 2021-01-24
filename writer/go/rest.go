package gowriter

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func dorest(method string, endpoint string) {
	client := http.Client{}
	request, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		log.Fatalln(err)
	}

	response, err := client.Do(request)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Fprintln(os.Stdout, response.Status)

	defer response.Body.Close()
	io.Copy(os.Stdout, response.Body)
}
