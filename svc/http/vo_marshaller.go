// Code generated by go generate; DO NOT EDIT.
// This file was generated by go-doudou at
// 2021-06-13 00:55:23.36453 +0800 CST m=+0.026791888
package ddhttp

import (
	"encoding/json"
	"github.com/unionj-cloud/go-doudou/name/strategies"
)


func (object HttpLog) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	objectMap[strategies.LowerCaseConvert("ClientIp")] = object.ClientIp
	objectMap[strategies.LowerCaseConvert("HttpMethod")] = object.HttpMethod
	objectMap[strategies.LowerCaseConvert("Uri")] = object.Uri
	objectMap[strategies.LowerCaseConvert("Proto")] = object.Proto
	objectMap[strategies.LowerCaseConvert("Host")] = object.Host
	objectMap[strategies.LowerCaseConvert("ReqContentLength")] = object.ReqContentLength
	objectMap[strategies.LowerCaseConvert("ReqHeader")] = object.ReqHeader
	objectMap[strategies.LowerCaseConvert("RequestId")] = object.RequestId
	objectMap[strategies.LowerCaseConvert("RawReq")] = object.RawReq
	objectMap[strategies.LowerCaseConvert("RespBody")] = object.RespBody
	objectMap[strategies.LowerCaseConvert("StatusCode")] = object.StatusCode
	objectMap[strategies.LowerCaseConvert("RespHeader")] = object.RespHeader
	objectMap[strategies.LowerCaseConvert("RespContentLength")] = object.RespContentLength
	objectMap[strategies.LowerCaseConvert("ElapsedTime")] = object.ElapsedTime
	objectMap[strategies.LowerCaseConvert("Elapsed")] = object.Elapsed
	return json.Marshal(objectMap)
}
