package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	c "github.com/opensds/opensds/client"
	"github.com/opensds/opensds/pkg/model"
)

var (
	cli *c.Client
)

func init() {
	cli = c.NewClient(&c.Config{
		Endpoint: "http://127.0.0.1:50040",
	})
}

func main() {
	volumebody := &model.VolumeSpec{
		Name: "test-sample",
		Size: 2,
	}
	vol, err := cli.CreateVolume(volumebody)
	if err != nil {
		fmt.Println("failed to CreateVolume:", err)
		return
	}

	hostName, _ := os.Hostname()
	initiators, _ := GetInitiator()
	attachReq := &model.VolumeAttachmentSpec{
		VolumeId: vol.Id,
		HostInfo: model.HostInfo{
			Host:      hostName,
			Platform:  runtime.GOARCH,
			OsType:    runtime.GOOS,
			Ip:        GetHostIp(),
			Initiator: initiators[0],
		},
		Metadata: vol.Metadata,
	}
	atcResp, errAttach := cli.CreateVolumeAttachment(attachReq)
	if errAttach != nil {
		fmt.Printf("the volume %s failed to publish to node %s.\n", vol.Id, hostName)
		cli.DeleteVolume(vol.Id, nil)
		return
	}
	atc, _ := cli.GetVolumeAttachment(atcResp.Id)

	fmt.Printf("host info is: %+v\n", atc.HostInfo)
	fmt.Printf("atc info is: %+v\n", atc.ConnectionInfo)

	resp, errAttach := cli.AttachVolume(atc)
	if errAttach != nil {
		fmt.Printf("the volume attachment %s failed to attach to node %s.\n", atc.Id, hostName)
		cli.DetachVolume(atc)
		cli.DeleteVolumeAttachment(atc.Id, nil)
		cli.DeleteVolume(vol.Id, nil)
		return
	}

	fmt.Println(resp["device"])
}

// GetInitiator returns all the ISCSI Initiator Name
func GetInitiator() ([]string, error) {
	res, err := exec.Command("cat", "/etc/iscsi/initiatorname.iscsi").CombinedOutput()
	iqns := []string{}
	if err != nil {
		log.Printf("Error encountered gathering initiator names: %v", err)
		return iqns, nil
	}

	lines := strings.Split(string(res), "\n")
	for _, l := range lines {
		if strings.Contains(l, "InitiatorName=") {
			iqns = append(iqns, strings.Split(l, "=")[1])
		}
	}

	log.Printf("Found the following iqns: %s", iqns)
	return iqns, nil
}

// GetHostIp return Host IP
func GetHostIp() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			return ipnet.IP.String()
		}
	}

	return "127.0.0.1"
}
