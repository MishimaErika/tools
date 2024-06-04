package kwssh

import (
	"regexp"
	"strings"
)

type machineDetail struct {
	// 服务器型号
	productName string
	// 服务器序列号
	sn string
	// ip
	ip string
	// 操作系统
	osName string
	// 内核版本
	kernelVersion string

	cpu struct {
		fullName   string
		cpuname    string
		cpuCoreNum string
	}
	// 内存总量
	memTotal string

	// 内存位置信息
	mems []meminfo
	// 硬盘信息
	hardDisks []diskinfo

	// Raid 信息
	raids []raidinfo
}

type diskinfo struct {

	// 硬盘厂商
	product string
	// 硬盘容量
	capacity string
	// 磁盘介质
	media string
}

type raidinfo struct {
	// raid等级
	raidLevel string
	// raid大小
	size string
}

type meminfo struct {
	location string
	memType  string
	size     string
}

// 硬盘信息解析函数
func parseDiskInfo(data string) []diskinfo {
	var hardDisks []diskinfo

	// 正则表达式匹配每个硬盘信息块
	re := regexp.MustCompile(`(?s)Media\s+:\s+(?P<Media>\w+)\s+Capacity\s+:\s+(?P<Capacity>[\d,\.]+\s+GB)\s+\([\d]+\s+bytes\)\s+Product ID\s+:\s+(?P<Product>\w+)`)
	matches := re.FindAllStringSubmatch(data, -1)

	for _, match := range matches {
		hardDisks = append(hardDisks, diskinfo{
			media:    match[1],
			capacity: strings.ReplaceAll(match[2], ",", ""),
			product:  match[3],
		})
	}

	return hardDisks
}

// Raid信息解析函数
func parseRaidInfo(data string) []raidinfo {
	var raidInfos []raidinfo

	// 正则表达式匹配每个RAID信息块
	re := regexp.MustCompile(`(?s)Layout\s+:\s+(?P<RaidLevel>RAID-\d+)\s+Size\s+:\s+(?P<Size>[\d,\.]+\s+GB)\s+\([\d]+\s+bytes\)`)
	matches := re.FindAllStringSubmatch(data, -1)

	for _, match := range matches {
		raidInfos = append(raidInfos, raidinfo{
			raidLevel: match[1],
			size:      strings.ReplaceAll(match[2], ",", ""),
		})
	}

	return raidInfos
}

// 解析内存位置容量信息
func parseMemInfo(data string) []meminfo {

	var memInfoList []meminfo = make([]meminfo, 0)
	mem := new(meminfo)

	for i, v := range strings.Split(data, "\n") {

		data := strings.Split(v, ":")

		if i%3 == 0 {
			if len(data) == 2 {
				// fmt.Printf("localtion: %#v\n", data[1])
				mem.location = strings.Trim(data[1], " ")
			}

		}

		if i%3 == 1 {
			if len(data) == 2 {
				mem.memType = strings.Trim(data[1], " ")
				// fmt.Printf("memType: %#v\n", data[1])
			}

		}

		if i%3 == 2 {
			if len(data) == 2 {
				// fmt.Printf("Size: %#v\n", data[1])
				mem.size = strings.Trim(data[1], " ")

				if mem.size != "" {
					memInfoList = append(memInfoList, meminfo{
						location: mem.location,
						size:     mem.size,
						memType:  mem.memType,
					})
				}
			}

		}

	}

	return memInfoList

}

const (
	cpuCoreNum    = `grep 'physical id' /proc/cpuinfo | sort -u | wc -l`
	cpuName       = `cat /proc/cpuinfo |grep "model name" | uniq -c | awk -F: '{print $2}'`
	sn            = `dmidecode -t1 |grep "Serial Number" | awk '{print $3}'`
	kernelVersion = `uname -r`
	osName        = `cat /etc/os-release  |grep "PRETTY_NAME" |awk -F= '{print $2}' | tr -d '"'`
	productName   = `dmidecode -t1 |grep "Product Name" | awk -F: '{print $2}'`
	memTotal      = `cat /proc/meminfo  | grep "MemTotal" | awk  '{print $2 }'`
	raidInfo      = `omreport storage vdisk controller=0   | grep -E "Layout|^Size"`
	diskInfo      = `omreport storage pdisk controller=0  | grep -E "Product ID|Capacity|Media"`
	mems          = `omreport chassis memory  |grep -E "Connector Name|Type|Size"`
)
