package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func authGithubDownload() {
	client := http.Client{}
	req, err := http.NewRequest("GET", "https://github.com/chriskim06/kubectl-private/archive/v0.0.2.zip", nil)
	if err != nil {
		fmt.Println("error creating http request")
		return
	}

	// this is how to use basic github auth to download an asset
	token := os.Getenv("GITHUB_ACCESS_TOKEN")
	fmt.Println(token)
	if token == "" {
		fmt.Println("GITHUB_ACCESS_TOKEN not found")
		return
	}
	req.Header.Add("Authorization", "token "+token)
	res, err := client.Do(req)
	if err != nil {
		fmt.Println("error making http request")
		return
	}
	defer res.Body.Close()

	out, err := os.Create("v0.0.2.zip")
	if err != nil {
		fmt.Println("error creating new file")
		return
	}
	defer out.Close()

	_, err = io.Copy(out, res.Body)
	if err != nil {
		fmt.Println("error copying res to new file")
		return
	}
}
