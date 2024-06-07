package model

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	kwDB *sqlx.DB
)

/*
```USE idc;

CREATE TABLE  machine_base_info (
	sn VARCHAR(255) NOT NULL PRIMARY KEY,
    ip VARCHAR(15),
    model VARCHAR(255),
    operating_system VARCHAR(255),
    kernel_version VARCHAR(255),
    cpu VARCHAR(255),
    memory VARCHAR(255),
	power VARCHAR(255)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE machine_disk_info (
    sn VARCHAR(255),
    disk VARCHAR(255),
    capacity VARCHAR(255),
    media VARCHAR(50),
    FOREIGN KEY (sn) REFERENCES machine_base_info (sn)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE machine_memory_info (
    sn VARCHAR(255),
    location VARCHAR(255),
    type VARCHAR(255),
    size VARCHAR(50),
    FOREIGN KEY (sn) REFERENCES machine_base_info (sn)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE machine_raid_info (
    sn VARCHAR(255),
    raid_level VARCHAR(50),
    capacity VARCHAR(255),
    FOREIGN KEY (sn) REFERENCES machine_base_info (sn)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;```
*/

type idc_machine_info struct {
	SN           string `db:"sn"`
	Label        string `db:"label"`
	ExternalIP   string `db:"external_ip"`
	InternalIP   string `db:"internal_ip"`
	IDRAC_IP     string `db:"idrac_ip"`
	ServiceName  string `db:"service_name"`
	ServiceOwner string `db:"service_owner"`
	MachineOwner string `db:"machine_owner"`
	Leader       string `db:"leader"`
	Cabinet      string `db:"cabinet"`
	UNumber      string `db:"U_number"`
	MachineModel string `db:"machine_model"`
	Comment      string `db:"comment"`
}

type Machine_INFO struct {
	Base    Machine_Base_INFO_MODEL
	Memorys []Machine_Memory_INFO_MODEL
	Disks   []Machine_Disk_INFO_MODEL
	Raids   []Machine_RAID_INFO_MODEL
}
type Machine_Base_INFO_MODEL struct {
	SN            string `db:"sn"`
	IP            string `db:"ip"`
	Model         string `db:"model"`
	OS            string `db:"operating_system"`
	KernelVersion string `db:"kernel_version"`
	Cpu           string `db:"cpu"`
	Memory        string `db:"memory"`
	Power         string `db:"power"`
}

type Machine_Memory_INFO_MODEL struct {
	SN       string `db:"sn"`
	Location string `db:"location"`
	Type     string `db:"type"`
	Size     string `db:"size"`
}

type Machine_Disk_INFO_MODEL struct {
	SN       string `db:"sn"`
	Product  string `db:"disk"`
	Capacity string `db:"capacity"`
	Media    string `db:"media"`
}

type Machine_RAID_INFO_MODEL struct {
	SN       string `db:"sn"`
	Level    string `db:"raid_level"`
	Capacity string `db:"capacity"`
}

func Init() error {

	dsn := "idc:kuwo@123@tcp(127.0.0.1:3306)/idc?charset=utf8"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		fmt.Printf("connect to DB error, err=%#v", err)
		return err
	}
	kwDB = db
	return nil
}

func WriteToDB(info Machine_INFO) error {

	tx, err := kwDB.Beginx()

	if err != nil {
		fmt.Printf("start tx err, err=%#v", err)
		return err
	}

	defer tx.Rollback()

	// 机器基本硬件信息到db
	base := info.Base
	_, err = tx.NamedExec("INSERT INTO idc.machine_base_info values (:sn, :ip, :model, :operating_system, :kernel_version, :cpu, :memory, :power)", base)
	if err != nil {
		fmt.Printf("insert base data to db err, err=%#v", err)
		return err
	}

	// 内存信息
	if len(info.Memorys) > 0 {
		for _, memory := range info.Memorys {
			_, err = tx.NamedExec("INSERT INTO machine_memory_info VALUES(:sn, :location, :type, :size)", memory)
			if err != nil {
				fmt.Printf("insert memory data to db err, err=%#v", err)
				return err
			}
		}
	}
	// 硬盘信息
	if len(info.Disks) > 0 {
		for _, disk := range info.Disks {
			_, err = tx.NamedExec("INSERT INTO machine_disk_info VALUES(:sn, :disk, :capacity, :media)", disk)
			if err != nil {
				fmt.Printf("insert disk data to db err, err=%#v", err)
				return err
			}
		}
	}

	// Raid信息
	if len(info.Raids) > 0 {
		for _, raid := range info.Raids {
			_, err = tx.NamedExec("INSERT INTO machine_raid_info VALUES(:sn, :raid_level, :capacity)", raid)
			if err != nil {
				fmt.Printf("insert raid data to db err, err=%#v", err)
				return err
			}
		}
	}

	tx.Commit()
	return nil

}

func Close() {
	kwDB.Close()
}
