package register

import (
	"github.com/alecthomas/log4go"
	"bytes"
)

type SRegister struct {
	console
	Logger      *log4go.Logger                 // 创建的tcp客户端对象
	Client      *Client                        // 创建的tcp客户端对象
	RegSuccess  bool                           // 注册成功标志
	ServiceData (map[string]map[string]string) // console 注册成功返回的页面服务器信息
}

/**
	可配置参数
 */
type RegConfig struct {
	IsDebug 	bool		`是否debug模式`
	LogName 	string		`log文件名`
	ConfigPath 	string		`配置文件路径`
	ConsolePath []string	`console前端文件目录`
}

const (
	DefaultHost    = "www.xdapp.com:8900"
	DefaultSSl     = true
	DefaultApp     = "test"
	DefaultName    = "console"
	DefaultKey     = ""
	DefaultLogName = "test.log"
)

var (
	MyLog *log4go.Logger // log 日志
	MyRpc *sMyRpc		 // rpc 服务
)

/**
	工厂创建
 */
func NewRegister(rfg RegConfig) *SRegister {


	if rfg.LogName == "" {
		rfg.LogName = DefaultLogName
	}

	if rfg.ConfigPath == "" {
		rfg.ConfigPath = defaultBaseDir() + "/config.yml"
	}

	if rfg.ConsolePath != nil {
		addConsolePath(rfg.ConsolePath)
	}

	MyRpc  = NewMyRpc()
	MyLog  = NewLog4go(rfg.IsDebug, rfg.LogName)

	conf   := LoadConfig(rfg.ConfigPath)
	client := NewClient(conf.Console.Host, *tcpConf)

	return &SRegister{
		conf.Console,
		MyLog,
		client,
		false,
		make (map[string]map[string]string)}
}

func (reg *SRegister) GetApp() string {
	return reg.App
}

func (reg *SRegister) GetName() string {
	return reg.Name
}
func (reg *SRegister) GetKey() string {
	return reg.Key
}

func (reg *SRegister) SetRegSuccess(status bool) {
	reg.RegSuccess = status
}

func (reg *SRegister) SetServiceData(data map[string]map[string]string) {
	reg.ServiceData = data
}

func (reg *SRegister) CloseClient() {
	reg.Client.Close(reg.RegSuccess)
}

func (reg *SRegister) Info(arg0 interface{}, args ...interface{}) {
	reg.Logger.Info(arg0, args ...)
}

func (reg *SRegister) Debug(arg0 interface{}, args ...interface{}) {
	reg.Logger.Debug(arg0, args ...)
}

func (reg *SRegister) Warn(arg0 interface{}, args ...interface{}) {
	reg.Logger.Warn(arg0, args ...)
}

func (reg *SRegister) Error(arg0 interface{}, args ...interface{}) {
	reg.Logger.Error(arg0, args ...)
}

/**
	tcp client
 */
func (reg *SRegister) CreateClient() {

	debugSuccessService()

	reg.Client.OnReceive(func(message []byte) {

		request := new(ReqestData)
		request.Unpack(bytes.NewReader(message))

		//myRpc.context.BaseContext.Set("receiveParam")

		// 执行rpc返回
		rpcData := MyRpc.handle(request.Data, MyRpc.context)

		rs := string(PackId(request.Id)) + string(rpcData)

		dataLen := len(rs)

		if dataLen < tcpConf.packageMaxLength {
			Send(reg.Client, request.Flag | 4, request.Fd, string(rs))

		} else {
			for i := 0; i < dataLen; i += tcpConf.packageMaxLength {

				chunkLength := Min(tcpConf.packageMaxLength, dataLen - i)
				chunk := Substr(string(rs), i, chunkLength)

				flag := request.Flag
				if dataLen - i == chunkLength {
					flag |= 4
				}
				Send(reg.Client, flag, request.Fd, chunk)
			}
		}
	})

	reg.Client.Connect()
}

/**
	获取key
 */
func (reg *SRegister) getKey() string {
	return reg.ServiceData["pageServer"]["key"]
}

/**
	获取host
 */
func (reg *SRegister) getHost() string {
	return reg.ServiceData["pageServer"]["host"]
}

/**
	log4go对象设置
 */
func NewLog4go(isDebug bool, logName string) *log4go.Logger {

	log4 := make(log4go.Logger)
	cw := log4go.NewConsoleLogWriter()

	// 非debug模式
	if isDebug == false {
		cw.SetFormat("[%T %D] [%L] %M")
	}
	log4.AddFilter("stdout", log4go.DEBUG, cw)
	log4.AddFilter("file", log4go.ERROR, log4go.NewFileLogWriter(logName, false))
	return &log4
}