package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"zeus/kwssh"
)

// 自定义类型实现flag.Value接口
type ipList []string

func (i *ipList) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *ipList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	ips      ipList
	filename = flag.String("filename", "", "批量执行命令的IP文件")
	username = flag.String("username", "root", "用户名")
	password = flag.String("password", "", "密码")
	port     = flag.Int("port", 22, "端口号")
	command  = flag.String("command", "", "要执行的命令")
	key      = flag.String("key", "", "私钥路径")
)

func main() {

	b1 := kwssh.New("n1", 5)
	flag.Var(&ips, "ip", "IP 地址列表，可以提供多个")

	// 解析命令行参数
	flag.Parse()

	if len(os.Args) == 1 {
		flag.Usage()
		return
	}
	task := kwssh.Task{}

	if *username != "" {
		task.User = *username
	}

	if *password != "" {
		task.Pass = *password
	}

	if *port != 22 {
		task.Port = int32(*port)
	} else {
		task.Port = 22
	}

	if *key != "" {
		// 使用公钥登录
		task.SSHType = kwssh.PUBLICKEY
		task.KeyPath = *key
	}

	if *password != "" {
		// 使用密码登录
		task.SSHType = kwssh.PASSWORD
		task.Pass = *password
	}

	if *key == "" && *password == "" {
		fmt.Println("请指定密码或key登录")
		return
	}

	if *command != "" {

		task.Command = []string{*command}

	} else {
		fmt.Println("执行命令不能为空")
		return
	}

	if len(ips) != 0 {
		for _, v := range ips {
			task.IP = v
			b1.AddTask("n1", task)
		}
	}

	if *filename != "" {
		data, err := os.Open(*filename)
		if err != nil {
			fmt.Printf("read ip list file err, err=%#v", err)
			return
		}

		scanner := bufio.NewScanner(data)

		for scanner.Scan() {
			if scanner.Text() == "" {
				continue
			}

			task.IP = strings.Trim(scanner.Text(), " ")
			b1.AddTask("n1", task)
		}

	}

	switch *command {
	case "fetch":
		b1.FetchInfo()
	case "fetchToDB":
		b1.FetchInfoToDB()
	default:
		b1.Run()
	}
}
