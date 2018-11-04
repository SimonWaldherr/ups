package ups

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

func HttpPostRequest(url, msgbody string) (string, error) {
	client := &http.Client{}
	body := bytes.NewBufferString(msgbody)
	clength := strconv.Itoa(len(msgbody))
	r, _ := http.NewRequest("POST", url, body)
	r.Header.Add("User-Agent", "NicerWatch")
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Content-Length", clength)

	rsp, err := client.Do(r)

	if err != nil {
		return "", err
	}

	defer func() {
		rsp.Body.Close()
	}()
	if rsp.StatusCode == 200 {
		bodyBytes, err := ioutil.ReadAll(rsp.Body)
		return fmt.Sprintf("%v", bodyBytes), err
	} else if err != nil {
		return "", err
	} else {
		return "", fmt.Errorf("The remote end did not return a HTTP 200 (OK) response:%#v\n", rsp)
	}
}
