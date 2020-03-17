package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var cmd *exec.Cmd
var stdin io.WriteCloser
var stdout io.ReadCloser
var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var execLog *log.Logger
var lastEchoLine string

var returnJsonMap map[string]string

func main() {
	fileName := "exec.log"
	logFile, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0766)
	if err != nil {
		log.Fatalln("open file error !")
	}
	defer logFile.Close()
	execLog = log.New(logFile, "", log.Lshortfile)
	go runBedrock()
	time.Sleep(2 * time.Second)
	go readStdout()
	go runAPI()
	reader := bufio.NewReader(os.Stdin)
	time.Sleep(5 * time.Second)
	for {
		time.Sleep(time.Second)
		fmt.Print("> ")
		cmdString, err := reader.ReadString('\n')
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		_, err = stdin.Write([]byte(cmdString))
		if err != nil {
			log.Fatal(err)
		}
	}

}

func runBedrock() {
	cmd = exec.Command("bedrock_server")

	stdin, _ = cmd.StdinPipe()
	stdout, _ = cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func readStdout() {
	reader := bufio.NewReader(stdout)

	//实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		matched, err := regexp.MatchString("(?ims)[\\[]\\d{4}[-]([0][1-9]|(1[0-2]))[-]([1-9]|([012]\\d)|(3[01]))([ \\t\\n\\x0B\\f\\r])(([0-1]{1}[0-9]{1})|([2]{1}[0-4]{1}))([:])(([0-5]{1}[0-9]{1}|[6]{1}[0]{1}))([:])((([0-5]{1}[0-9]{1}|[6]{1}[0]{1})))([ \\t\\n\\x0B\\f\\r])(INFO)[\\]]([ \\t\\n\\x0B\\f\\r])(Player disconnected)([:])([ \\t\\n\\x0B\\f\\r])(.*)", line)
		if err != nil {
			log.Fatal(err)
		}
		if !matched {
			execLog.Println(line)
		}
		fmt.Println(line)
		lastEchoLine = line
	}
}

func runAPI() {
	r := gin.Default()
	r.GET("/exec", func(c *gin.Context) {
		accessToken := c.Query("access_token")
		command := c.Query("command")
		if accessToken != "testToken" {
			c.JSON(401, gin.H{
				"message": "Authorization failed.Please use the correct access token. ",
			})
			return
		}
		_, err := stdin.Write([]byte(command + "\n"))
		if err != nil {
			log.Fatal(err)
		}
		c.JSON(200, gin.H{
			"message": command,
		})
	})
	r.GET("/ws", func(c *gin.Context) {
		//升级get请求为webSocket协议
		ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		for {
			//读取ws中的数据
			_, message, err := ws.ReadMessage()
			if err != nil {
				break
			}
			time.Sleep(2 * time.Second)
			execLog.Println("Remote(API):" + string(message))
			fmt.Println("Remote(API):" + string(message))
			_, err = stdin.Write([]byte(string(message) + "\n"))
			if err != nil {
				log.Fatal(err)
			}
			//写入ws数据
			time.Sleep(2 * time.Second)
			returnJsonMap = make(map[string]string)
			returnJsonMap["returnString"] = lastEchoLine
			err = ws.WriteJSON(returnJsonMap)
			if err != nil {
				break
			}
		}
	})
	err := r.Run()
	if err != nil {
		log.Fatal(err)
	}
}
