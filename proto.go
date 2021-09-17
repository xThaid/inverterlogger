package inverterlogger

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Frame struct {
	PayloadLength uint16
	ControlCode   uint16
	SerialNumber  uint16
	DeviceSN      uint32
	Payload       []byte
}

func NewFrame(deviceSN uint32, payload []byte) *Frame {
	return &Frame{
		PayloadLength: uint16(len(payload)),
		ControlCode:   0x4510,
		SerialNumber:  0x0000,
		DeviceSN:      deviceSN,
		Payload:       payload,
	}
}

func (f *Frame) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteByte(0xA5) // V5 start marker
	binary.Write(&buf, binary.LittleEndian, f.PayloadLength)
	binary.Write(&buf, binary.LittleEndian, f.ControlCode)
	binary.Write(&buf, binary.BigEndian, f.SerialNumber)
	binary.Write(&buf, binary.LittleEndian, f.DeviceSN)
	buf.Write(f.Payload)
	buf.WriteByte(calcCheckSum8(buf.Bytes()[1:])) // skip start marker for checksum calculation
	buf.WriteByte(0x15)                           // V5 end marker

	return buf.Bytes(), nil
}

func (f *Frame) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)

	b, err := buf.ReadByte()
	if err != nil {
		return fmt.Errorf("can't read start marker: %w", err)
	} else if b != 0xA5 {
		return fmt.Errorf("expected 0xA5 as start marker, got: %x", b)
	}

	err = binary.Read(buf, binary.LittleEndian, &f.PayloadLength)
	if err != nil {
		return fmt.Errorf("can't read payload length: %w", err)
	}

	err = binary.Read(buf, binary.LittleEndian, &f.ControlCode)
	if err != nil {
		return fmt.Errorf("can't read control code: %w", err)
	}

	err = binary.Read(buf, binary.BigEndian, &f.SerialNumber)
	if err != nil {
		return fmt.Errorf("can't read serial number: %w", err)
	}

	err = binary.Read(buf, binary.LittleEndian, &f.DeviceSN)
	if err != nil {
		return fmt.Errorf("can't read device SN: %w", err)
	}

	f.Payload = make([]byte, f.PayloadLength)
	n, err := buf.Read(f.Payload)
	if err != nil {
		return fmt.Errorf("can't read payload: %w", err)
	} else if n != int(f.PayloadLength) {
		return fmt.Errorf("read only %d bytes of payload instead of %d", n, f.PayloadLength)
	}

	_, err = buf.ReadByte()
	if err != nil {
		return fmt.Errorf("can't read checksum: %w", err)
	}

	b, err = buf.ReadByte()
	if err != nil {
		return fmt.Errorf("can't read end marker: %w", err)
	} else if b != 0x15 {
		return fmt.Errorf("expected 0x15 as end marker, got: %x", b)
	}

	if buf.Len() != 0 {
		return fmt.Errorf("%d bytes left in the buffer", buf.Len())
	}

	return nil
}

type RequestPayload struct {
	FrameType    uint8
	SensorType   uint16
	DeliveryTime uint32
	PowerOnTime  uint32
	OffsetTime   uint32

	DeviceAddress uint8
	FunctionCode  uint8
	StartReg      uint16
	RegCount      uint16
}

func NewRequestPayload(startReg, regCount uint16) *RequestPayload {
	return &RequestPayload{
		FrameType:    0x02,
		SensorType:   0x0000,
		DeliveryTime: 0x00000000,
		PowerOnTime:  0x00000000,
		OffsetTime:   0x00000000,

		DeviceAddress: 0x01, // must be equal to 1
		FunctionCode:  0x03, // 3 - read real time data
		StartReg:      startReg,
		RegCount:      regCount,
	}
}

func (r *RequestPayload) marshalBusinessData() []byte {
	var buf bytes.Buffer
	buf.WriteByte(r.DeviceAddress)
	buf.WriteByte(r.FunctionCode)
	binary.Write(&buf, binary.BigEndian, r.StartReg)
	binary.Write(&buf, binary.BigEndian, r.RegCount)
	binary.Write(&buf, binary.LittleEndian, calcCRC16Modbus(buf.Bytes()))
	return buf.Bytes()
}

func (r *RequestPayload) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	binary.Write(&buf, binary.LittleEndian, r.FrameType)
	binary.Write(&buf, binary.LittleEndian, r.SensorType)
	binary.Write(&buf, binary.LittleEndian, r.DeliveryTime)
	binary.Write(&buf, binary.LittleEndian, r.PowerOnTime)
	binary.Write(&buf, binary.LittleEndian, r.OffsetTime)
	buf.Write(r.marshalBusinessData())

	return buf.Bytes(), nil
}

type ResponsePayload struct {
	FrameType    uint8
	StatusCode   uint8
	DeliveryTime uint32
	PowerOnTime  uint32
	OffsetTime   uint32

	DeviceAddress uint8
	FunctionCode  uint8
	ValueLength   uint8
	Value         []byte
}

func (r *ResponsePayload) unmarshalBusinessPayload(data []byte) error {
	buf := bytes.NewBuffer(data)

	err := binary.Read(buf, binary.LittleEndian, &r.DeviceAddress)
	if err != nil {
		return fmt.Errorf("can't read slave address: %w", err)
	}

	err = binary.Read(buf, binary.LittleEndian, &r.FunctionCode)
	if err != nil {
		return fmt.Errorf("can't read function code: %w", err)
	}

	err = binary.Read(buf, binary.LittleEndian, &r.ValueLength)
	if err != nil {
		return fmt.Errorf("can't read value length: %w", err)
	}

	r.Value = make([]byte, r.ValueLength)
	n, err := buf.Read(r.Value)
	if err != nil {
		return fmt.Errorf("can't read value: %w", err)
	} else if n != int(r.ValueLength) {
		return fmt.Errorf("read only %d bytes of value instead of %d", n, r.ValueLength)
	}

	var crc uint16
	err = binary.Read(buf, binary.LittleEndian, &crc)
	if err != nil {
		return fmt.Errorf("can't read value length: %w", err)
	}

	// There are two zero bytes at the end of the payload -- I couldn't find what they denote.
	buf.Next(2)

	if buf.Len() != 0 {
		return fmt.Errorf("%d bytes left in the buffer", buf.Len())
	}

	return nil
}

func (r *ResponsePayload) UnmarshalBinary(data []byte) error {
	buf := bytes.NewBuffer(data)

	err := binary.Read(buf, binary.LittleEndian, &r.FrameType)
	if err != nil {
		return fmt.Errorf("can't read frame type: %w", err)
	}

	err = binary.Read(buf, binary.LittleEndian, &r.StatusCode)
	if err != nil {
		return fmt.Errorf("can't read status code: %w", err)
	}

	err = binary.Read(buf, binary.LittleEndian, &r.DeliveryTime)
	if err != nil {
		return fmt.Errorf("can't read delivery time: %w", err)
	}

	err = binary.Read(buf, binary.LittleEndian, &r.PowerOnTime)
	if err != nil {
		return fmt.Errorf("can't read power on time: %w", err)
	}

	err = binary.Read(buf, binary.LittleEndian, &r.OffsetTime)
	if err != nil {
		return fmt.Errorf("can't read offset time: %w", err)
	}

	return r.unmarshalBusinessPayload(buf.Bytes())
}
