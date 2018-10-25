package register

import (
	"io"
	"net"
	"fmt"
	"bytes"
	"errors"

	"encoding/binary"

	"github.com/leesper/tao"
)

//  标识   | 版本    | 长度    | 头信息       | 自定义内容    |  正文
//  ------|--------|---------|------------|-------------|-------------
//  Flag  | Ver    | Length  | Header     | Context      | Body
//  1     | 1      | 4       | 17         | 默认0，不定   | 不定
//  C     | C      | N       |            |             |
//
//
//  其中 Header 部分包括
//
//  服务ID     | rpc请求序号  | 管理员ID      | 自定义信息长度
//  ----------|-------------|-------------|-----------------
//  ServiceId | RequestId   | AdminId     | ContextLength
//  4         | 4           | 4           | 1
//  N         | N           | N           | C

type Request struct {
	Prefix
	Header		   // 头信息
	Context []byte // 自定义内容
	Body    []byte // 正文
}

type Prefix struct {
	Flag    byte   // 标志位 成功 |= 4
	Ver     byte   // 版本
	Length  uint32 // 长度
}

type Header struct {
	AppId         uint32
	ServiceId     uint32
	RequestId     uint32
	AdminId       uint32
	ContextLength byte
}

const (
	PREFIX_LENGTH     = 6                             // Flag 1字节、 Ver 1字节、 Length 4字节
	HEADER_LENGTH     = 17                            // 默认消息头长度, 不包括 PREFIX_LENGTH
	CONTEXT_OFFSET    = PREFIX_LENGTH + HEADER_LENGTH // 自定义上下文内容所在位置，   23
	SEND_CHUNK_LENGTH = 0x200000					  // 单次发送的包大小
)

var (
	ErrReadByteEmpty = errors.New("读取数据为空")
)

// MessageNumber returns the message number.
func (req Request) MessageNumber() int32 {
	return int32(1)
}

// Serialize Request
func (req Request) Serialize() ([]byte, error) {
	var writer = new(bytes.Buffer)
	pack(writer, req.Prefix)
	pack(writer, req.Header)
	return BytesCombine(writer.Bytes(), req.Context, req.Body), nil
}

// Deserialize
func DeserializeRequest(data []byte) (tao.Message, error) {
	if data == nil {
		return nil, tao.ErrNilData
	}

	req := new(Request)
	reader := bytes.NewReader(data)
	binary.Read(reader, binary.BigEndian, req)
	return req, nil
}

// 标识   | 版本    | 长度    | 头信息       | 自定义内容    |  正文
// ------|--------|---------|------------|-------------|-------------
// Flag  | Ver    | Length  | Header     | Context      | Body
// 1     | 1      | 4       | 17         | 默认0，不定   | 不定
// C     | C      | N       |            |             |
//
//
// 其中 Header 部分包括
//
// AppId     | 服务ID      | rpc请求序号  | 管理员ID      | 自定义信息长度
// ----------|------------|------------|-------------|-----------------
// AppId     | ServiceId  | RequestId  | AdminId     | ContextLength
// 4         | 4          | 4          | 4           | 1
// N         | N          | N          | N           | C

func (req Request) Decode(raw net.Conn) (tao.Message, error) {
	byteChan := make(chan []byte)
	errorChan := make(chan error)

	go func(bc chan []byte, ec chan error) {

		buf := make([]byte, PREFIX_LENGTH)
		_, err := io.ReadFull(raw, buf)

		if err != nil {
			ec <- err
			close(bc)
			close(ec)
			Logger.Warn("read failed")
			return
		}
		bc <- buf
	}(byteChan, errorChan)

	var readBytes []byte

	select {
	case err := <-errorChan:
		return nil, err

	case readBytes = <-byteChan:
		if readBytes == nil {
			return nil, ErrReadByteEmpty
		}

		var prefix Prefix
		err := binary.Read(bytes.NewReader(readBytes), binary.BigEndian, &prefix); if err != nil {
			return nil, err
		}
		if prefix.Ver != 1 {
			return nil, errors.New("消息版本错误" + string(prefix.Ver))
		}

		if prefix.Length > uint32(tcpConfig.packageMaxLength) {
			err := fmt.Sprintf("数据长度为%d, 大于最大值%d", prefix.Length, tcpConfig.packageMaxLength)
			Logger.Error(err)
			return nil, errors.New(err)
		}

		var header Header
		hdBytes := make([]byte, HEADER_LENGTH)
		_, err = io.ReadFull(raw, hdBytes)
		if err != nil {
			return nil, err
		}
		hdBuf := bytes.NewReader(hdBytes)
		if err = binary.Read(hdBuf, binary.BigEndian, &header); err != nil {
			return nil, err
		}

		ctxLen := int(header.ContextLength)
		ctxBytes := make([]byte, ctxLen)
		_, err = io.ReadFull(raw, ctxBytes)
		if err != nil {
			return nil, err
		}

		bodyBytes := make([]byte, int(prefix.Length) - HEADER_LENGTH - ctxLen)
		_, err = io.ReadFull(raw, bodyBytes)
		if err != nil {
			return nil, err
		}

		return Request{prefix,header,ctxBytes,bodyBytes}, nil
	}
}

// Encode encodes the message into bytes data.
func (req Request) Encode(msg tao.Message) ([]byte, error) {
	data, err := msg.Serialize()
	if err != nil {
		return nil, err
	}
	return data, nil
}

/**
转发消息到其它服务
 */
func transportRpcRequest(flag byte, ver byte, header Header, context []byte, data[]byte) {

	flag = flag | FLAG_RESULT_MODE
	dataLength := len(data)

	if dataLength < SEND_CHUNK_LENGTH {
		tcpSendReq(Request{
			Prefix{
				uint8(flag | FLAG_FINISH),
				ver,
				uint32(HEADER_LENGTH + len(context) + dataLength),
			},
			header,
			context,
			data,
		})
		return
	}

	// 大于 拆包分段发送
	for i := 0; i < dataLength; i += SEND_CHUNK_LENGTH {
		chunkLen := Min(SEND_CHUNK_LENGTH, dataLength-i)
		chunk := data[i:chunkLen]
		if dataLength-i == chunkLen {
			flag |= FLAG_FINISH
		}

		tcpSendReq(Request{
			Prefix{
				uint8(flag),
				ver,
				uint32(HEADER_LENGTH + len(context) + chunkLen),
			},
			header,
			context,
			chunk,
		})
	}
}

func tcpSendReq(request Request) {
	if err := Conn.Write(request); err != nil {
		Logger.Error("error", err)
	}
}