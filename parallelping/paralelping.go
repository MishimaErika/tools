package parallelping

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"zeus/gate"
)

type pingResult struct {
	IP string
	ok bool
}

func parseIPFromFile(ipfile string) []string {
	ips := []string{}

	f, err := os.Open(ipfile)
	if err != nil {
		fmt.Printf("读取IP列表文件: [%s] 失败, err=%#v\n", ipfile, err)
		return ips
	}

	defer f.Close()

	reader := bufio.NewReader(f)
	for {
		ip, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		ip = ip[:len(ip)-1]
		ips = append(ips, ip)
	}
	return ips
}

func ParallelPing(ipfile string) {

	var wg sync.WaitGroup
	var w sync.WaitGroup
	g := gate.New(450)

	resChan := make(chan pingResult, 2000)

	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, ip := range parseIPFromFile(ipfile) {
			w.Add(1)

			go func(ip string) {
				defer func() {
					w.Done()
					g.Leave()
				}()
				g.Enter()
				tmp := pingResult{IP: ip, ok: false}
				cmd := exec.Command("ping", "-c3", ip)
				if _, err := cmd.Output(); err == nil {
					tmp.ok = true
				}
				resChan <- tmp
			}(ip)
		}
		w.Wait()

	}()

	go func() {
		wg.Wait()
		close(resChan)
	}()

	ret := ""
	for v := range resChan {

		if v.ok {
			ret = "success"
		} else {
			ret = "failed"
		}

		fmt.Printf("%s\t\t[%s]\n", v.IP, ret)
	}
}
