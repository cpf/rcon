// Package rcon implements the communication protocol for communicating
// with RCON servers.
package rcon

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
)

const (
	packetPaddingSize int32 = 2 // Size of Packet's padding.
	packetHeaderSize  int32 = 8 // Size of Packet's header.
)

const (
	terminationSequence = "\x00" // Null empty ASCII string suffix.
)

// Packet type constants.
// https://developer.valvesoftware.com/wiki/Source_RCON_Protocol#Packet_Type
const (
	exec          int32 = 2
	auth          int32 = 3
	authResponse  int32 = 2
	responseValue int32 = 0
)

// Rcon package errors.
var (
	ErrInvalidWrite        = errors.New("Failed to write the payload corretly to remote connection.")
	ErrInvalidRead         = errors.New("Failed to read the response corretly from remote connection.")
	ErrInvalidChallenge    = errors.New("Server failed to mirror request challenge.")
	ErrUnauthorizedRequest = errors.New("Client not authorized to remote server.")
	ErrFailedAuthorization = errors.New("Failed to authorize to the remote server.")
)

type Client struct {
	Host       string   // The IP address of the remote server.
	Port       int      // The Port the remote server's listening on.
	authorized bool     // Has the client been authorized by the server?
	connection net.Conn // The TCP connection to the server.
}

type header struct {
	size       int32 // The size of the payload.
	challenge  int32 // The challenge ths server should mirror.
	headerType int32 // The type of request being sent.
}

type Packet struct {
	Header header // Packet header.
	Body   string // Body of packet.
}

// NewClient creates a new Client type, creating the connection
// to the server specified by the host and port arguements. If
// the connection fails, an error is returned.
func NewClient(host string, port int) (client *Client, err error) {
	client = &Client{Host: host, Port: port}
	return
}

func (this *Client) Connect() (err error) {
	this.connection, err = net.Dial("tcp", fmt.Sprintf("%v:%v", this.Host, this.Port))
	return
}

// Authorize calls Send with the appropriate command type and the provided
// password.  The response packet is returned if authorization is successful
// or a potential error.
func (this *Client) Authorize(password string) (response *Packet, err error) {
	if response, err = this.send(auth, password); nil == err {
		if response.Header.headerType == authResponse {
			this.authorized = true
		} else {
			err = ErrFailedAuthorization
			response = nil
			return
		}
	}

	return
}

// Execute calls Send with the appropriate command type and the provided
// command.  The response packet is returned if the command executed successfully
// or a potential error.
func (this *Client) Execute(command string) (response *Packet, err error) {
	return this.send(exec, command)
}

// NewPacket returns a pointer to a new Packet type.
func newPacket(challenge, typ int32, body string) (packet *Packet) {
	size := int32(len([]byte(body)) + int(packetHeaderSize+packetPaddingSize))
	return &Packet{header{size, challenge, typ}, body}
}

// Sends accepts the commands type and its string to execute to the clients server,
// creating a packet with a random challenge id for the server to mirror,
// and compiling its payload bytes in the appropriate order. The resonse is
// decompiled from its bytes into a Packet type for return. An error is returned
// if send fails.
func (this *Client) send(typ int32, command string) (response *Packet, err error) {
	if typ != auth && !this.authorized {
		err = ErrUnauthorizedRequest
		return
	}

	// Create a random challenge for the server to mirror in its response.
	var challenge int32
	binary.Read(rand.Reader, binary.LittleEndian, &challenge)

	// Create the packet from the challenge, typ and command
	// and compile it to its byte payload
	packet := newPacket(challenge, typ, command)
	payload, err := packet.compile()

	var n int

	if nil != err {
		return
	} else if n, err = this.connection.Write(payload); nil != err {
		return
	} else if n != len(payload) {
		err = ErrInvalidWrite
		return
	}

	var header header

	if err = binary.Read(this.connection, binary.LittleEndian, &header.size); nil != err {
		return
	} else if err = binary.Read(this.connection, binary.LittleEndian, &header.challenge); nil != err {
		return
	} else if err = binary.Read(this.connection, binary.LittleEndian, &header.headerType); nil != err {
		return
	}

	if packet.Header.headerType == auth && header.headerType == responseValue {
		// Discard, empty SERVERDATA_RESPOSE_VALUE from authorization.
		this.connection.Read(make([]byte, header.size-packetHeaderSize))

		// Reread the packet header.
		if err = binary.Read(this.connection, binary.LittleEndian, &header.size); nil != err {
			return
		} else if err = binary.Read(this.connection, binary.LittleEndian, &header.challenge); nil != err {
			return
		} else if err = binary.Read(this.connection, binary.LittleEndian, &header.headerType); nil != err {
			return
		}
	}

	if header.challenge != packet.Header.challenge {
		err = ErrInvalidChallenge
		return
	}

	body := make([]byte, header.size-packetHeaderSize)

	n, err = this.connection.Read(body)

	if nil != err {
		return
	} else if n != len(body) {
		err = ErrInvalidRead
		return
	}

	response = new(Packet)
	response.Header = header
	response.Body = strings.TrimRight(string(body), terminationSequence)

	return
}

// Compile converts a packets header and body into its approriate
// byte array payload, returning an error if the binary packages
// Write method fails to write the header bytes in their little
// endian byte order.
func (this Packet) compile() (payload []byte, err error) {
	var size int32 = this.Header.size
	var buffer bytes.Buffer
	var padding [packetPaddingSize]byte

	if err = binary.Write(&buffer, binary.LittleEndian, &size); nil != err {
		return
	} else if err = binary.Write(&buffer, binary.LittleEndian, &this.Header.challenge); nil != err {
		return
	} else if err = binary.Write(&buffer, binary.LittleEndian, &this.Header.headerType); nil != err {
		return
	}

	buffer.WriteString(this.Body)
	buffer.Write(padding[:])

	return buffer.Bytes(), nil
}
