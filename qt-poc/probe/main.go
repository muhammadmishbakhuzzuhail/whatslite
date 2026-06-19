// SPDX-License-Identifier: GPL-3.0-or-later
// Probe uji Fase-5: dial bridge.sock engine headless, panggil 2 method, cetak
// respons. Membuktikan engine melayani bridge TANPA Wails.
package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: probe <sock>")
		os.Exit(2)
	}
	c, err := net.Dial("unix", os.Args[1])
	if err != nil {
		fmt.Println("dial err:", err)
		os.Exit(1)
	}
	defer c.Close()
	c.Write([]byte(`{"@type":"Version","@extra":"1","args":[]}` + "\n"))
	c.Write([]byte(`{"@type":"GetChats","@extra":"2","args":[]}` + "\n"))
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	r := bufio.NewReader(c)
	for i := 0; i < 2; i++ {
		line, err := r.ReadBytes('\n')
		if err != nil {
			fmt.Println("read err:", err)
			return
		}
		fmt.Print("RESP ", string(line))
	}
}
