package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	flag.Usage = func() {
		fmt.Println("vul webhook -- -url=<webhook-url> -- <vul args>")
		flag.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(os.Stderr, "     %v\n", f.Usage)
		})
	}
	webhookUrl := flag.String("url", "", "webhook endpoint url")
	flag.Parse()

	if len(*webhookUrl) <= 0 {
		flag.Usage()
		log.Fatal("vul webhook plugin expects a webhook endpoint url")
	}

	log.Println("running vul...")
	out, err := runScan(os.Args, exec.Command)
	if err != nil {
		flag.Usage()
		log.Fatal("vul returned an error: ", err, " output: ", string(out))
	}

	log.Println("sending results to webhook...")
	resp, err := sendToWebhook(*webhookUrl, &http.Client{
		Timeout: time.Second * 30,
	}, out)
	if err != nil {
		log.Fatal("failed to send to webhook: ", err)
	}

	log.Println("webhook returned: ", string(resp))
}

func sendToWebhook(webhookUrl string, nc *http.Client, body []byte) ([]byte, error) {
	resp, err := nc.Post(webhookUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build post request: %w", err)
	}
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return b, nil
}

func runScan(args []string, execCmd func(string, ...string) *exec.Cmd) ([]byte, error) {
	vulArgsIndex := findVulSep(args)
	if vulArgsIndex < 0 {
		return nil, fmt.Errorf("invalid arguments specified")
	}

	vulArgs := os.Args[vulArgsIndex:]
	if !containsSlice(vulArgs, "format") {
		vulArgs = append(vulArgs, []string{"--format=json"}...)
	}
	vulArgs = append(vulArgs, []string{"--quiet", "--timeout=30s"}...)

	log.Println("running vul with args: ", vulArgs)
	out, err := execCmd("vul", vulArgs...).CombinedOutput()
	if err != nil {
		return out, err
	}

	log.Println("vul returned: ", string(out))
	return out, err
}

func findVulSep(args []string) int {
	for i, a := range args {
		// vul args separator is "--"
		if a == "--" {
			if i+1 >= len(args) {
				return -1 // bad case if someone specifies no vul args
			} else {
				return i + 1 // common case with good args
			}
		}
	}
	return -1 // bad case if no vul sep & args specified
}

func containsSlice(haystack []string, needle string) bool {
	for _, item := range haystack {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}
