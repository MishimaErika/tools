package kwssh

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSH struct {
	client *ssh.Client
}

var (
	resChan = make(chan commandResult, 100)
)

type commandResult struct {
	ip   string
	user string
	res  []struct {
		cmd  string
		data []byte
	}
}

func (s *SSH) NewClient(target *Task) error {
	auth := []ssh.AuthMethod{}
	var timeout time.Duration = 0

	if target.SSHType == PUBLICKEY {
		key, err := os.ReadFile(target.KeyPath)
		if err != nil {
			return fmt.Errorf("kwssh: ParsekeyPath err: %#v", err.Error())
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return fmt.Errorf("kwssh: ParseKey err: %#v", err.Error())
		}

		auth = append(auth, ssh.PublicKeys(signer))
	}

	if target.SSHType == PASSWORD {
		auth = append(auth, ssh.Password(target.Pass))
	}

	if target.Timeout != 0 {
		timeout = target.Timeout
	}

	sshConfig := &ssh.ClientConfig{
		Auth:            auth,
		User:            target.User,
		Timeout:         timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	server := fmt.Sprintf("%s:%d", target.IP, target.Port)
	client, err := ssh.Dial("tcp", server, sshConfig)
	if err != nil {
		return fmt.Errorf("kwssh: connect to [%s] failed, err=%#v", server, err.Error())
	}

	s.client = client
	// fmt.Println("连接到", s.client.RemoteAddr().String(), "成功")
	return nil
}

func (s *SSH) RunCommands(cmds []string) (commandResult, error) {

	r := commandResult{}
	if len(cmds) == 0 {
		return r, fmt.Errorf("kw_ssh: commands 不能为0")
	}

	defer s.client.Close()

	r.ip = strings.Split(s.client.RemoteAddr().String(), ":")[0]
	r.user = s.client.User()

	r.res = make([]struct {
		cmd  string
		data []byte
	}, 0)

	for _, cmd := range cmds {

		session, err := s.client.NewSession()
		if err != nil {
			return r, fmt.Errorf("kw_ssh: session create failed, err=%#v", err.Error())
		}
		defer session.Close()

		output, _ := session.CombinedOutput(cmd)

		r.res = append(r.res, struct {
			cmd  string
			data []byte
		}{
			cmd:  cmd,
			data: output,
		})

	}

	resChan <- r
	return r, nil
}
