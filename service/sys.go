package service

import (
	"fmt"
	"strings"
)

type SysService struct {
	NormalService
}

func NewSysService(register IRegister) *SysService {
	return &SysService{NormalService{register: register}}
}

/**
	注册服务，在连接到 console 微服务系统后，会收到一个 sys_reg() 的rpc回调
  */
func (service *SysService) Reg(time int64, rand string, hash string) []interface{} {

	// 当前方法名
	fun := strings.ToLower(GetFuncName())

	// 验证hash
	if Sha1(fmt.Sprintf("%s.%s.%s", IntToStr(time), rand, "xdapp.com")) != hash {
		return []interface{} {fun, false};
	}

	// 超时
	if Time() - time > 180 {
		return []interface{} {fun, false};
	}

	app  := service.getApp()
	key  := service.getKey()
	name := service.getName()
	time  = Time()
	hash  = getHash(app, name, IntToStr(time), rand, key)

	arr := map[string]interface{}{"app": app, "name": name , "time": time, "rand": rand, "hash": hash}

	return []interface{} {fun, arr}
}

/**
	获取菜单列表
 */
func (service *SysService) Menu() {

}

/**
	注册失败
 */
func (service *SysService) RegErr(msg string, data interface{}) {
	service.register.SetRegSuccess(false)
	service.register.Debug("注册失败", msg, data)
}

/**
	注册成功回调
 */
func (service *SysService) RegOk(data map[string]map[string]string, time int, rand string, hash string) {

	app  := service.getApp()
	key  := service.getKey()
	name := service.getName()

	if getHash(app, name, IntToStr(time), rand, key) != hash {
		service.register.SetRegSuccess(false)
		service.register.CloseClient()
		return
	}

	// 注册成功
	service.register.SetRegSuccess(true)
	service.register.SetServiceData(data)

	service.register.Debug("RPC服务注册成功，服务名:" + app + "-> " + name)

	// 同步页面
	service.register.ConsolePageSync()
}

/**
	测试接口
 */
func (service *SysService) Test(str string) {
	fmt.Println(str)
}