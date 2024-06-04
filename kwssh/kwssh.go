package kwssh

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"zeus/gate"
)

const (
	_ = iota
	PASSWORD
	PUBLICKEY
)

type Task struct {
	IP      string
	Port    int32
	SSHType int
	User    string
	KeyPath string
	Pass    string
	Command []string

	// 设置ssh的超时时间
	Timeout time.Duration
}

type PlayBook struct {
	// playbookName
	name string
	// 并行任务数量
	g *gate.Gate
	// 要执行的任务
	m []*Task
}

func New(playbookname string, pNum int) *PlayBook {

	var g *gate.Gate
	if pNum > 0 && pNum < 20 {
		g = gate.New(pNum)
	} else {
		g = gate.New(1)
	}

	return &PlayBook{
		name: playbookname,
		g:    g,
		m:    make([]*Task, 0),
	}
}

func (p *PlayBook) AddTask(name string, t Task) {

	task := new(Task)

	task.Command = t.Command
	task.IP = t.IP
	task.KeyPath = t.KeyPath
	task.Pass = t.Pass
	task.Port = t.Port
	task.SSHType = t.SSHType
	task.Timeout = t.Timeout
	task.User = t.User

	if name == p.name {
		p.m = append(p.m, task)
	}
}

// 执行命令，将命令结果推送到channel
func (p *PlayBook) exec() {

	var wg sync.WaitGroup

	for _, v := range p.m {
		wg.Add(1)
		go func(v *Task) {
			defer func() {
				wg.Done()
				p.g.Leave()
			}()

			p.g.Enter()

			cli := SSH{}
			err := cli.NewClient(v)
			if err != nil {
				fmt.Println(err)
				return
			}

			_, err = cli.RunCommands(v.Command)
			if err != nil {
				fmt.Println(err)
				return
			}
		}(v)

	}

	wg.Wait()
	close(resChan)

}

// 从channel中读取执行结果，并展示
func (p *PlayBook) readResult() {
	// 读取命令执行结果
	for res := range resChan {
		for _, v := range res.res {
			fmt.Printf("IP: [%s], User: [%s], Command: [%s]\nCommand Output:\n%s\n", res.ip, res.user, v.cmd, string(v.data))
		}
	}
}

func (p *PlayBook) Run() {
	p.exec()
	p.readResult()
}

func (p *PlayBook) FetchInfo() {

	infos := make([]machineDetail, 0)

	for _, v := range p.m {
		v.Command = []string{
			productName,
			sn,
			cpuName,
			cpuCoreNum,
			memTotal,
			osName,
			kernelVersion,
			diskInfo,
			raidInfo,
			mems,
		}
	}

	p.exec()

	for res := range resChan {
		detail := machineDetail{}
		detail.ip = res.ip

		for k, v := range res.res {
			if k == 0 {
				// 处理机器型号
				detail.productName = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")
			}

			if k == 1 {
				// 处理机器SN
				detail.sn = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")
			}

			if k == 2 {
				// 处理CPU信息
				detail.cpu.cpuname = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")
			}

			if k == 3 {
				// 处理CPU数量
				detail.cpu.cpuCoreNum = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")
				detail.cpu.fullName = detail.cpu.cpuname + " x " + detail.cpu.cpuCoreNum
			}

			if k == 4 {
				// 处理内存信息
				detail.memTotal = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")
				f, err := strconv.ParseFloat(detail.memTotal, 64)
				if err == nil {
					detail.memTotal = strconv.Itoa(int(f / 1024))
				}
			}

			if k == 5 {
				// 操作系统信息 etc.. CentOS Ubuntu
				detail.osName = strings.Trim(strings.Trim(string(v.data), "\t"), "\n")
			}

			if k == 6 {
				// 内核版本
				detail.kernelVersion = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")
			}

			if k == 7 {
				// 磁盘信息

				if v.err != nil {
					// fmt.Printf("没有omreport命令, err=%#v", v.err.Error())
					break
				}
				detail.hardDisks = parseDiskInfo(string(v.data))
			}

			if k == 8 {
				// raid信息

				if v.err != nil {
					// fmt.Printf("没有omreport命令, err=%#v", v.err.Error())
					break
				}

				detail.raids = parseRaidInfo(string(v.data))
			}

			if k == 9 {
				// 内存位置信息

				if v.err != nil {
					// fmt.Printf("没有omreport命令, err=%#v", v.err.Error())
					break
				}

				detail.mems = parseMemInfo(string(v.data))
			}
		}

		infos = append(infos, detail)
	}

	for _, v := range infos {
		fmt.Printf("%-9s:\t%s\n%-7s:\t[%s]\n%-6s:\t[%s]\n%-5s:\t[%s]\n%-5s:\t[%s]\n%-9s:\t[%s]\n%-7s:\t[%s MB]\n",
			"IP", v.ip, "型号", v.productName, "序列号", v.sn, "操作系统", v.osName, "内核版本", v.kernelVersion, "CPU", v.cpu.fullName,
			"内存", v.memTotal)

		if len(v.mems) != 0 {
			fmt.Println("内存位置信息 :")
			for _, v := range v.mems {
				fmt.Printf("\t内存位置: [%s] 内存类型: [%s] 内存容量: [%s]\n", v.location, v.memType, v.size)
			}
		}

		if len(v.hardDisks) != 0 {
			fmt.Println("磁盘信息 :")
			for _, v := range v.hardDisks {
				fmt.Printf("\t磁盘: [%s] 容量: [%s] 介质: [%s]\n", v.product, v.capacity, v.media)
			}
		}

		if len(v.raids) != 0 {
			fmt.Println("RAID信息 :")
			for _, v := range v.raids {
				fmt.Printf("\tRAID Level: [%s] 容量: [%s]\n", v.raidLevel, v.size)
			}
		}

		fmt.Println()
	}
}
