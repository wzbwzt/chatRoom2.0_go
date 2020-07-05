package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"
)

/*全局常量*/
var (
	//退出命令管道
	chanQuit = make(chan bool, 0)
	//服务端连接对象
	conn     net.Conn
)

/*错误处理逻辑：有错误就暴力退出程序*/
func CHandleError(err error, why string) {
	if err != nil {
		fmt.Println(why, err)
		os.Exit(1)
	}
}

/*客户端主程序*/
func main() {
	//在命令行参数中携带昵称
	nameInfo := [3]interface{}{"name", "Mr. Unknow", "昵称"}
	retValuesMap := GetCmdlineArgs(nameInfo)
	name := retValuesMap["name"].(string)

	//拨号连接，获得connection
	var e error
	conn, e = net.Dial("tcp", "127.0.0.1:8888")
	CHandleError(e, "net.Dial")
	defer func() {
		conn.Close()
	}()

	//在一条独立的协程中输入，并发送消息
	go handleSend(conn, name)
	//在一条独立的协程中接收服务端消息
	go handleReceive(conn)
	//设置优雅退出逻辑
	<-chanQuit

}

/*处理收到的消息*/
func handleReceive(conn net.Conn) {
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != io.EOF {
			CHandleError(err, "conn.Read")
		}
		if n > 0 {
			msg := string(buffer[:n])
			fmt.Println(msg)
		}
	}

}

/*处理消息发送*/
func handleSend(conn net.Conn, name string) {
	//发送昵称到服务端
	_, err := conn.Write([]byte(name))
	CHandleError(err, "conn.Write([]byte(name))")
	reader := bufio.NewReader(os.Stdin)
	for {
		//读取标准输入
		lineBytes, _, _ := reader.ReadLine()
		lineStr := string(lineBytes)
		//上传文件
		//upload#文件名#文件路径
		if strings.Index(lineStr,"upload") == 0{
			//拿到文件名和文件路径
			strs := strings.Split(lineStr,"#")
			if len(strs)!=3{
				fmt.Println("上传信息格式有误！")
				continue
			}
			fileName := strs[1]
			filePath := strs[2]
			//构造数据包
			dataPack := make([]byte, 0)
			//写入数据包头部（upload#文件名）
			header := make([]byte, 100)
			copy(header,[]byte("upload#"+fileName+"#"))
			dataPack = append(dataPack, header...)
			//写入数据包身体（文件字节）
			fileBytes, _ := ioutil.ReadFile(filePath)
			dataPack = append(dataPack,fileBytes...)
			//写给服务端
			conn.Write(dataPack)
		}else{
			//发送到服务端
			_, err := conn.Write(lineBytes)
			CHandleError(err, "conn.Write")
			//正常退出
			if lineStr == "exit" {
				os.Exit(0)
			}
		}

	}
}

/*获取命令行参数*/
func GetCmdlineArgs(argInfos ...[3]interface{}) (retValuesMap map[string]interface{}) {
	//fmt.Printf("type=%T,value=%v\n", argInfos, argInfos)
	//初始化返回结果
	retValuesMap = map[string]interface{}{}

	//预定义【用户可能输入的各种类型的指针】
	var strValuePtr *string
	var intValuePtr *int

	//预定义【用户可能输入的各种类型的指针】的容器
	//用户可能输入好几个string型的参数值，存放在好几个string型的指针中，将这些同种类型的指针放在同种类型的map中
	//例如：flag.Parse()了以后，可以根据【strValuePtrsMap["cmd"]】拿到【存放"cmd"值的指针】
	var strValuePtrsMap = map[string]*string{}
	var intValuePtrsMap = map[string]*int{}

	/*	var floatValuePtr *float32
		var floatValuePtrsMap []*float32
		var boolValuePtr *bool
		var boolValuePtrsMap []*bool*/

	//遍历用户需要接受的所有命令定义
	for _, argArray := range argInfos {

		/*
			先把每个命令的名称和用法拿出来,
			这俩货都是string类型的，所有都可以通过argArray[i].(string)轻松愉快地获得其字符串
			一个叫“cmd”，一个叫“你想干嘛”
			"cmd"一会会用作map的key
		*/
		//[3]interface{}
		//["cmd" "未知类型" "你想干嘛"]
		//["gid"     0     "要查询的商品ID"]
		//上面的破玩意类型[string 可能是任意类型 string]
		nameValue := argArray[0].(string)  //拿到第一个元素的string值,是命令的name
		usageValue := argArray[2].(string) //拿到最后一个元素的string值，是命令的usage

		//判断argArray[1]的具体类型
		switch argArray[1].(type) {
		case string:
			//得到【存放cmd的指针】，cmd的值将在flag.Parse()以后才会有
			//cmdValuePtr = flag.String("cmd", argArray[1].(string), "你想干嘛")
			strValuePtr = flag.String(nameValue, argArray[1].(string), usageValue)

			//将这个破指针以"cmd"为键，存在【专门放置string型指针的map，即strValuePtrsMap】中
			strValuePtrsMap[nameValue] = strValuePtr

		case int:
			//得到【存放gid的指针】，gid的值将在flag.Parse()以后才会有
			//gidValuePtr = flag.String("gid", argArray[1].(int), "商品ID")
			intValuePtr = flag.Int(nameValue, argArray[1].(int), usageValue)

			//将这个破指针以"gid"为键，存在【专门放置int型指针的map，即intValuePtrsMap】中
			intValuePtrsMap[nameValue] = intValuePtr
		}

	}

	/*
		程序运行到这里，所有不同类型的【存值指针】都放在对相应类型的map中了
		flag.Parse()了以后，可以从map中以参数名字获取出【存值指针】，进而获得【用户输入的值】
	*/

	//用户输入完了，解析，【用户输入的值】全都放在对应的【存值指针】中
	flag.Parse()

	/*
		遍历各种可能类型的【存值指针的map】
	*/
	if len(strValuePtrsMap) > 0 {
		//从【cmd存值指针的map】中拿取cmd的值，还以cmd为键存入结果map中
		for k, vPtr := range strValuePtrsMap {
			retValuesMap[k] = *vPtr
		}
	}
	if len(intValuePtrsMap) > 0 {
		//从【gid存值指针的map】中拿取gid的值，还以gid为键存入结果map中
		for k, vPtr := range intValuePtrsMap {
			retValuesMap[k] = *vPtr
		}
	}
	//返回结果map
	return
}


