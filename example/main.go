package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/xThaid/inverterlogger"
)

var (
	loggerAddress = "192.168.XXX.XXX"
	loggerSN      = uint32(1700000000)

	connectionTimeout = 5 * time.Second
)

func sendRequest(startReg, regCnt int) (map[int]uint16, error) {
	conn, err := net.DialTimeout("tcp", loggerAddress, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(connectionTimeout))
	conn.SetWriteDeadline(time.Now().Add(connectionTimeout))

	requestPayload, _ := inverterlogger.NewRequestPayload(uint16(startReg), uint16(regCnt)).MarshalBinary()
	requestFrame, _ := inverterlogger.NewFrame(loggerSN, requestPayload).MarshalBinary()

	_, err = conn.Write(requestFrame)
	if err != nil {
		return nil, fmt.Errorf("write to server failed: %w", err)
	}

	reply := make([]byte, 512)
	replyLen, err := conn.Read(reply)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	reply = reply[:replyLen]

	responseFrame := inverterlogger.Frame{}
	err = responseFrame.UnmarshalBinary(reply)
	if err != nil {
		return nil, fmt.Errorf("can't unmarshal response: %w", err)
	}

	responsePayload := inverterlogger.ResponsePayload{}
	err = responsePayload.UnmarshalBinary(responseFrame.Payload)
	if err != nil {
		return nil, fmt.Errorf("can't unmarshal response payload: %w", err)
	}

	fmt.Printf("Response frame: %+v; payload: %+v", responseFrame, responsePayload)

	buf := bytes.NewBuffer(responsePayload.Value)

	res := make(map[int]uint16)
	for i := 0; i < regCnt; i++ {
		var val uint16
		binary.Read(buf, binary.BigEndian, &val)
		res[startReg+i] = val
	}

	return res, nil
}

func main() {
	registers, err := sendRequest(0x3f, 20)
	if err != nil {
		fmt.Printf("Error while sending request: %v\n", err)
		os.Exit(1)
	}

	power := int(registers[0x50]) / 10
	totalEnergy := (int(registers[0x3f]) + (int(registers[0x40]) << 16)) * 100

	fmt.Printf("Current power: %d, total energy: %d\n", power, totalEnergy)
}
