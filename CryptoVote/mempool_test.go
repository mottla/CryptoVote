package main

import (
	"testing"
	"bytes"
	"io"
	"os"
	"net/http"
	"fmt"
)

func Test_showPool_HTTP(t *testing.T) {

	b := new(bytes.Buffer)
	res, _ := http.Post("http://127.0.0.1:3000/pool", "application/json; charset=utf-8",b)
	io.Copy(os.Stdout, res.Body)
	fmt.Println()

	res, _ = http.Post("http://127.0.0.1:3000/pool", "application/json; charset=utf-8",b)
	io.Copy(os.Stdout, res.Body)

}
