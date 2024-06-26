package session

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/acarl005/stripansi"
	tea "github.com/charmbracelet/bubbletea"
	kallisti "github.com/perlsaiyan/zif/protocol"
)

// Read from the MUD stream, parse MSDP, etc
func (s *Session) mudReader() tea.Cmd {
	sub := s.Sub
	buffer := make([]byte, 1)
	var outbuf []byte

	for {
		s.Socket.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
		timeout := false
		_, err := s.Socket.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				timeout = true
				// time out
			} else {
				fmt.Println("Error: ", err)
				sub <- tea.KeyMsg.String
				s.Connected = false
				// TODO return a command to close out the session, otherwise we just hang
				return nil

			}

		}
		if timeout {
			// we timed out without reading anything
			if len(outbuf) > 0 {
				linestring := string(outbuf)
				strippedlinestring := stripansi.Strip(linestring)
				s.AddRinglogEntry(time.Now().UnixNano(), linestring, strippedlinestring)
				s.ActionParser(outbuf)
				s.Content += linestring
				sub <- UpdateMessage{Session: s.Name, Content: linestring}
				outbuf = outbuf[:0]
			}
		} else if buffer[0] == 255 {

			_, _ = s.Socket.Read(buffer) // read one char for now to eat GA
			if buffer[0] == 249 {        //this is GO AHEAD
				//store it, this is likely a prompt
				linestring := strings.TrimRight(string(outbuf), "\r\n")
				strippedlinestring := stripansi.Strip(linestring)
				s.AddRinglogEntry(time.Now().UnixNano(), linestring, strippedlinestring)

				s.Content += string(outbuf) + "\n"
				s.FireEvent("core.prompt", EventData{})

				sub <- UpdateMessage{Session: s.Name, Content: string(outbuf) + "\n"}
				s.ActionParser(outbuf)
				outbuf = outbuf[:0]
			} else if buffer[0] == 251 { // WILL
				_, _ = s.Socket.Read(buffer)
				//log.Println("Debug WILL:", buffer)
				if buffer[0] == 1 { // ECHO / password mask
					log.Printf("Got password mask request (IAC WILL ECHO)")
					if !s.PasswordMode {
						s.PasswordMode = true
						//log.Printf("Sending DO ECHO\n")
						buf := []byte{255, 253, 1} // send IAC DO ECHO
						s.Socket.Write(buf)
						sub <- TextinputMsg{Session: s.Name, Password_mode: true, Toggle_password: true}
					} else {
						//sub <- TextinputMsg{Session: s.Name, Password_mode: true, Toggle_password: true}
						//log.Printf("Skipping DO ECHO (loop protection) (currently %v)\n", s.PasswordMode)
					}

				} else if buffer[0] == 69 {
					log.Printf("Offered MSDP, accepting")
					buf := []byte{255, 253, 69, 255, kallisti.SB, kallisti.MSDP, kallisti.MSDP_VAR, 'L', 'I', 'S', 'T',
						kallisti.MSDP_VAL, 'C', 'O', 'M', 'M', 'A', 'N', 'D', 'S', 255, kallisti.SE}
					s.Socket.Write(buf)
					s.MSDP.HandleWill(s.Socket)

				} else {
					log.Printf("SERVER WILL %v (unhandled)\n", buffer)
				}
			} else if buffer[0] == 252 { // WONT
				_, _ = s.Socket.Read(buffer)
				if buffer[0] == 1 {
					log.Printf("Got password unmask request (IAC WONT ECHO)")
					sub <- TextinputMsg{Session: s.Name, Password_mode: false, Toggle_password: true}
				} else {
					log.Printf("SERVER WONT %v (unhandled)\n", buffer)
				}
			} else if buffer[0] == 253 { // DO
				_, _ = s.Socket.Read(buffer)
				//log.Printf("Got DO %v", buffer)
				if buffer[0] == 24 { // TERM TYPE
					//buf := []byte{255, 250, 24, 0, 'x', 't', 'e', 'r', 'm', '-', '2', '5', '6', 'c', 'o', 'l', 'o', 'r', 255, 240}
					buf := []byte{255, 251, 24}
					log.Printf("Sending %v", buf)
					s.Socket.Write(buf)
				}
			} else if buffer[0] == 254 { // DONT
				_, _ = s.Socket.Read(buffer)
				log.Printf("Got DONT %v", buffer)
			} else if buffer[0] == kallisti.SB {

				var sb []byte
				for {
					_, _ = s.Socket.Read(buffer)
					if buffer[0] == kallisti.SE {
						break
					}
					sb = append(sb, buffer...)
				}
				log.Printf("Good SB: %v", sb)
				switch sb[0] {
				case 69:
					s.MSDP.HandleSB(s.Socket, sb)
				case 24:
					switch s.TTCount {
					case 0:
						log.Printf("Sending zif termtype")
						s.Socket.Write([]byte{255, 250, 24, 0, 'z', 'i', 'f', 255, 240})
						s.TTCount += 1
					case 1:
						log.Printf("Sending XTERM-256COLOR termtype")
						s.Socket.Write([]byte{255, 250, 24, 0, 'X', 'T', 'E', 'R', 'M', '-', '2', '5', '6', 'C', 'O', 'L', 'O', 'R', 255, 240})
						s.TTCount += 1
					default:
						log.Printf("Sending MTTS 2831 termtype")
						s.Socket.Write([]byte{255, 250, 24, 0, 'M', 'T', 'T', 'S', ' ', '2', '8', '3', '1', 255, 240})
					}

				}
			} else {
				log.Printf("Unknown IAC %v\n", buffer[0])
			}
		} else if buffer[0] == 10 {
			// newline, print big buf and go
			linestring := strings.TrimRight(string(outbuf), "\r\n")
			strippedlinestring := stripansi.Strip(linestring)
			s.AddRinglogEntry(time.Now().UnixNano(), linestring, strippedlinestring)
			s.ActionParser(outbuf)

			s.Content += linestring + "\n"
			sub <- UpdateMessage{Session: s.Name, Content: string(outbuf) + "\n"}
			outbuf = outbuf[:0]
		} else {
			outbuf = append(outbuf, buffer[0])
		}
	}
}
