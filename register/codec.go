package register

import (
	"io"
	"fmt"
	"encoding/binary"
	"bytes"
)

// 返回字段
type ResponseData struct {
	Flag 		byte	// 标志位 成功 |= 4
	Len 		uint32	// data 长度 32 位短整形
	Fd 			uint32	// fd
	Data 		[]byte	// rpc 协议数据
}

//  请求字段
type ReqestData struct {
	Flag 		byte	// 标志位 成功 |= 4
	Len 		uint32	// id + data 长度 32 位短整形
	AdminId 	uint32	// 操作人id
	Fd 			uint32	// fd
	Id 			uint32	// 唯一id
	Data 		[]byte	// rpc 协议数据
}

// id字段
type IdData struct {
	Id 		uint32	// id 32 位短整形
}

/**
	请求数据打包
 */
func (p *ReqestData) Pack(writer io.Writer) {
	writeData(writer, &p.Flag)
	writeData(writer, &p.Len)
	writeData(writer, &p.AdminId)
	writeData(writer, &p.Fd)
	writeData(writer, &p.Id)
	writeData(writer, &p.Data)
}

/**
	请求数据解包
 */
func (p *ReqestData) Unpack(reader io.Reader) {
	readData(reader, &p.Flag)
	readData(reader, &p.Len)
	readData(reader, &p.AdminId)
	readData(reader, &p.Fd)
	readData(reader, &p.Id)
	p.Data = make([]byte, p.Len - 4)		// Len 等于id长度+data
	readData(reader, &p.Data)
}

func (p *ReqestData) String() string {
	return fmt.Sprintf("Flag:%s Length:%d AdminId:%d fd:%d id:%d data:%s",
		p.Flag,
		p.Len,
		p.AdminId,
		p.Fd,
		p.Id,
		p.Data,
	)
}

/**
	返回数据打包
 */
func (resp *ResponseData) Pack(writer io.Writer) {
	writeData(writer, &resp.Flag)
	writeData(writer, &resp.Len)
	writeData(writer, &resp.Fd)
	writeData(writer, &resp.Data)
}

/**
	返回数据解包
 */
func (resp *ResponseData) Unpack(reader io.Reader) {
	readData(reader, &resp.Flag)
	readData(reader, &resp.Len)
	readData(reader, &resp.Fd)
	resp.Data = make([]byte, resp.Len)		// Len 等于id长度+data
	readData(reader, &resp.Data)
}

/**
	id数据打包
 */
func (id *IdData) Pack(writer io.Writer) {
	writeData(writer, &id.Id)
}

func writeData(writer io.Writer, data interface{}) {
	err := binary.Write(writer, binary.BigEndian, data)
	if err != nil {
		MyLog.Error(err.Error());
		return
	}
}

func readData(reader io.Reader, data interface{}) {
	err := binary.Read(reader, binary.BigEndian, data)
	if err != nil {
		MyLog.Error(err.Error());
		return
	}
}

/**
	id 打包
 */
func PackId(id uint32) []byte {
	Id := &IdData{
		Id: id,
	}
	buf := new(bytes.Buffer)
	Id.Pack(buf)

	return buf.Bytes()
}