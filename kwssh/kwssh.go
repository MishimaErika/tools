package kwssh

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"zeus/gate"
	db "zeus/model"
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

func (p *PlayBook) Run() {
	var wg sync.WaitGroup

	go p.exec()
	wg.Add(1)

	// 读取结果并输出
	go func() {
		defer wg.Done()
		for res := range resChan {
			for _, v := range res.res {
				fmt.Printf("IP: [%s], User: [%s], Command: [%s]\nCommand Output:\n%s\n", res.ip, res.user, v.cmd, strings.TrimLeft(string(v.data), " "))
			}
		}
	}()

	wg.Wait()
}

func (p *PlayBook) FetchInfo() {
	var wg sync.WaitGroup

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
			pwrsupplies,
		}
	}

	go p.exec()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// 读取命令结果写入到标准输出
		for res := range resChan {
			detail := machineDetail{}
			detail.ip = res.ip

			// 获取机器基本信息
			for k, v := range res.res {
				if k == 0 {
					// 处理机器型号
					detail.productName = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")

					if len(detail.productName) == 0 {
						detail.productName = "无权限查看"
					}
				}

				if k == 1 {
					// 处理机器SN
					detail.sn = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")
					if len(detail.sn) == 0 {
						detail.sn = "无权限查看"
					}
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
					detail.hardDisks = parseDiskInfo(string(v.data))
				}

				if k == 8 {
					// raid信息
					detail.raids = parseRaidInfo(string(v.data))
				}

				if k == 9 {
					// 内存位置信息
					detail.mems = parseMemInfo(string(v.data))
				}

				if k == 10 {
					// 电源信息
					detail.power = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")
				}
			}

			fmt.Printf("%-9s:\t%s\n%-7s:\t[%s]\n%-6s:\t[%s]\n%-5s:\t[%s]\n%-5s:\t[%s]\n%-9s:\t[%s]\n%-7s:\t[%s MB]\n",
				"IP", detail.ip, "型号", detail.productName, "序列号", detail.sn, "操作系统", detail.osName, "内核版本", detail.kernelVersion, "CPU", detail.cpu.fullName,
				"内存", detail.memTotal)

			if len(detail.power) != 0 {
				fmt.Printf("%-5s:\t[%s]\n", "电源模块", detail.power)
			}

			if len(detail.mems) != 0 {
				fmt.Println("内存位置信息 :")
				for _, v := range detail.mems {
					fmt.Printf("\t内存位置: [%s] 内存类型: [%s] 内存容量: [%s]\n", v.location, v.memType, v.size)
				}
			}

			if len(detail.hardDisks) != 0 {
				fmt.Println("磁盘信息 :")
				for _, v := range detail.hardDisks {
					fmt.Printf("\t磁盘: [%s] 容量: [%s] 介质: [%s]\n", v.product, v.capacity, v.media)
				}
			}

			if len(detail.raids) != 0 {
				fmt.Println("RAID信息 :")
				for _, v := range detail.raids {
					fmt.Printf("\tRAID Level: [%s] 容量: [%s]\n", v.raidLevel, v.size)
				}
			}

			fmt.Println()
		}

	}()

	wg.Wait()
}

func (p *PlayBook) FetchInfoToDB() {
	var wg sync.WaitGroup

	// 初始化数据库
	err := db.Init()
	if err != nil {
		fmt.Printf("初始化数据库失败, err=%#v", err)
		return
	}

	defer db.Close()

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
			pwrsupplies,
		}
	}

	go p.exec()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// 读取命令结果写入到数据库
		for res := range resChan {
			detail := machineDetail{}
			detail.ip = res.ip

			// 获取机器基本信息
			for k, v := range res.res {
				if k == 0 {
					// 处理机器型号
					detail.productName = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")

					if len(detail.productName) == 0 {
						detail.productName = "无权限查看"
					}
				}

				if k == 1 {
					// 处理机器SN
					detail.sn = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")
					if len(detail.sn) == 0 {
						detail.sn = "无权限查看"
					}
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
					detail.hardDisks = parseDiskInfo(string(v.data))
				}

				if k == 8 {
					// raid信息
					detail.raids = parseRaidInfo(string(v.data))
				}

				if k == 9 {
					// 内存位置信息
					detail.mems = parseMemInfo(string(v.data))
				}

				if k == 10 {
					// 电源信息
					detail.power = strings.Trim(strings.TrimLeft(string(v.data), " "), "\n")
				}
			}

			// 组装数据到db
			info := db.Machine_INFO{}

			info.Disks = make([]db.Machine_Disk_INFO_MODEL, 0)
			info.Raids = make([]db.Machine_RAID_INFO_MODEL, 0)
			info.Memorys = make([]db.Machine_Memory_INFO_MODEL, 0)

			info.Base.Cpu = detail.cpu.fullName
			info.Base.IP = detail.ip
			info.Base.KernelVersion = detail.kernelVersion
			info.Base.Memory = detail.memTotal + " MB"
			info.Base.Model = detail.productName
			info.Base.OS = detail.osName
			info.Base.SN = detail.sn

			if len(detail.power) != 0 {
				info.Base.Power = detail.power
			}

			if len(detail.mems) != 0 {

				for _, v := range detail.mems {

					// 组装内存信息
					info.Memorys = append(info.Memorys, db.Machine_Memory_INFO_MODEL{
						SN:       info.Base.SN,
						Size:     v.size,
						Type:     v.memType,
						Location: v.location,
					})
				}
			}

			if len(detail.hardDisks) != 0 {
				for _, v := range detail.hardDisks {

					// 组装disk信息 结构体
					info.Disks = append(info.Disks, db.Machine_Disk_INFO_MODEL{
						SN:       info.Base.SN,
						Media:    v.media,
						Capacity: v.capacity,
						Product:  v.product,
					})
				}
			}

			if len(detail.raids) != 0 {
				for _, v := range detail.raids {

					// 组装raid信息 结构体
					info.Raids = append(info.Raids, db.Machine_RAID_INFO_MODEL{
						SN:       info.Base.SN,
						Level:    v.raidLevel,
						Capacity: v.size,
					})
				}
			}

			err := db.WriteToDB(info)
			if err != nil {
				fmt.Printf("db.WriteToDB(info) err, err=%#v", err)
				continue
			}

		}

	}()

	wg.Wait()
}
