/*

// (c) Copy jordi collell jordic@gmail.com
// OpenSOurce License, whatever you want..
(I don't know a lot of them)

Package sshtunnel provides a simple ssh tunnel for
default net connections.
Work with it tunneling a mysql connection:

    config := ssh.ClientConfig{
        User: "yourusername",
        Auth: []ssh.AuthMethod{
            ssh.Password("yourpass"),
        },
    }

    tunelConf := sshtunnel.TunnelConf{
        Remote_addr:       "sshserver",
        Remote_local_addr: "localhost:3306",
        Local_addr:        "localhost:3306",
        Ssh_Config:        config,
    }

    sshtunnel.CreateTunnel(tunelConf)

    // also you need to block, till ssh coneection is established
    // because tunnel is launched in a goroutine
    <-sshtunnel.WaitTunnel()

    and finally, when you finsih with your tunnel..

    sshtunnel.CloseTunnel()

*/

package sshtunnel

import (
	"code.google.com/p/go.crypto/ssh"
	"io"
	"net"
)

type TunnelConf struct {
	Remote_addr       string // remote server address (SSH Server)
	Remote_local_addr string // address to tunnel on remote host
	Local_addr        string // local address to tunnel to remote
	Ssh_Config        ssh.ClientConfig
}

var (
	quit  chan bool // channel for quiting tunnel
	ready chan bool // channel for waiting remote connection
)

func init() {
	ready = make(chan bool) // true when connection is ready for use
	quit = make(chan bool)  // receive a true, for closing ssh tunnel
}

func CreateTunnel(t TunnelConf) {
	go create_tunnel(t)
}

func create_tunnel(t TunnelConf) {
	// login to remote ssh server
	conn, err := ssh.Dial("tcp", t.Remote_addr, &t.Ssh_Config)
	if err != nil {
		return
	}

	// local endpoint that will forward
	local, err := net.Listen("tcp", t.Local_addr)
	if err != nil {
		return
	}
	// send signal ssh connected
	ready <- true
	for {

		l, err := local.Accept()
		if err != nil {
			return
		}

		go func() {
			remote, err := conn.Dial("tcp", t.Remote_local_addr)
			if err != nil {
				return
			}
			// copy bytes from local to remote
			go copy_network(l, remote)
			go copy_network(remote, l)

			if <-quit {
				conn.Close()
				remote.Close()
				local.Close()
				return
			}

		}()

	}
}

// ends tunnel connection, by signaling gooroutine
func CloseTunnel() {
	quit <- true
	//close(quit)
}

// get channel for waiting tunnel ready
func WaitTunnel() chan bool {
	return ready
}

func copy_network(in net.Conn, out net.Conn) {
	buf := make([]byte, 10240)
	for {
		n, err := in.Read(buf)
		if io.EOF == err {
			//log.Printf("io.EOF")
			return
		} else if nil != err {
			//log.Printf("resend err\n", err)
			return
		}
		out.Write(buf[:n])
	}
}
