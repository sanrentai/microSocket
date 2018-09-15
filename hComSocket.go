package microSocket

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"microSocket/util"
	"time"
)

const (
	CONSTHEADER       = "Header"
	CONSTHEADERLENGTH = 6
	CONSTMLENGTH      = 4
)

type CommSocket struct {
}

func (this *CommSocket) ConnHandle(msf *Msf, sess *Session) {
	defer func() {
		msf.SessionMaster.DelSessionById(sess.Id)
		//调用断开链接事件
		msf.MsfEvent.OnClose(sess.Id)
	}()
	var errs error
	tempBuff := make([]byte, 0)
	readBuff := make([]byte, 14)
	data := make([]byte, 20)
	//设置最迟期限（心跳包）
	sess.Con.SetReadDeadline(time.Now().Add(time.Duration(8) * time.Second))
	for {
		n, err := sess.Con.Read(readBuff)
		//设置最迟期限（心跳包）
		sess.Con.SetReadDeadline(time.Now().Add(time.Duration(8) * time.Second))
		if err != nil {
			return
		}
		tempBuff = append(tempBuff, readBuff[:n]...)
		tempBuff, data, errs = this.Depack(tempBuff)
		if errs != nil {
			log.Println(errs)
			return
		}

		if len(data) == 0 {
			continue
		}
		//把请求的到数据转化为map
		requestData := util.String2Map(string(data))
		if requestData["module"] == "" || requestData["action"] == "" || msf.EventPool.ModuleExit(requestData["module"]) == false {
			log.Println("not find module ", requestData)
			continue
		}
		requestData["fd"] = fmt.Sprintf("%d", sess.Id)

		//调用接收消息事件
		if msf.MsfEvent.OnMessage(sess.Id, requestData) == false {
			return
		}
		//路由
		msf.EventPool.Hook(requestData["module"], requestData["action"], requestData)
	}
}

func (this *CommSocket) Pack(message []byte) []byte {
	return append(append([]byte(CONSTHEADER), this.IntToBytes(len(message))...), message...)
}

//解包
func (this *CommSocket) Depack(buff []byte) ([]byte, []byte, error) {
	length := len(buff)

	//如果包长小于header 就直接返回 因为接收的数据不完整
	if length < CONSTHEADERLENGTH+CONSTMLENGTH {
		return buff, nil, nil
	}

	//如果header不是 指定的header 说明此数据已经被污染 直接返回错误
	if string(buff[:CONSTHEADERLENGTH]) != CONSTHEADER {
		return []byte{}, nil, errors.New("header is not safe")
	}

	msgLength := this.BytesToInt(buff[CONSTHEADERLENGTH : CONSTHEADERLENGTH+CONSTMLENGTH])
	if length < CONSTHEADERLENGTH+CONSTMLENGTH+msgLength {
		return buff, nil, nil
	}

	data := buff[CONSTHEADERLENGTH+CONSTMLENGTH : CONSTHEADERLENGTH+CONSTMLENGTH+msgLength]
	buffs := buff[CONSTHEADERLENGTH+CONSTMLENGTH+msgLength:]
	return buffs, data, nil
}

func (this *CommSocket) IntToBytes(n int) []byte {
	x := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, x)
	return bytesBuffer.Bytes()
}

func (this *CommSocket) BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)
	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)
	return int(x)
}
