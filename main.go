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
	CheckOrigin: func (r *http.Request) bool {
		return true
	},
}

var lastEchoLine string
var stdoutLogs [5]string
var execLog *log.Logger
func main() {
	fileName := "exec.log"
	logFile,err  := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE, 0766)
	if err != nil {
		log.Fatalln("open file error !")
	}
	defer logFile.Close()
	execLog = log.New(logFile,"",log.Lshortfile)
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
			fmt.Fprintln(os.Stderr, err)
		}
		stdin.Write([]byte(cmdString))
	}

}

func runBedrock() {
	cmd = exec.Command("bedrock_server")

	stdin, _ = cmd.StdinPipe()
	stdout, _ = cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	cmd.Run()
}

func readStdout(){
	reader := bufio.NewReader(stdout)

	//实时循环读取输出流中的一行内容
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		matched, err := regexp.MatchString("(?ims)[\\[]\\d{4}[-]([0][1-9]|(1[0-2]))[-]([1-9]|([012]\\d)|(3[01]))([ \\t\\n\\x0B\\f\\r])(([0-1]{1}[0-9]{1})|([2]{1}[0-4]{1}))([:])(([0-5]{1}[0-9]{1}|[6]{1}[0]{1}))([:])((([0-5]{1}[0-9]{1}|[6]{1}[0]{1})))([ \\t\\n\\x0B\\f\\r])(INFO)[\\]]([ \\t\\n\\x0B\\f\\r])(Player disconnected)([:])([ \\t\\n\\x0B\\f\\r])(.*)", str)

		execLog.Println(line)
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
		stdin.Write([]byte(command + "\n"))
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
			mt, message, err := ws.ReadMessage()
			if err != nil {
				break
			}
			time.Sleep(2*time.Second)
			execLog.Println("Remote(API):"+string(message))
			fmt.Println("Remote(API):"+string(message))
			stdin.Write([]byte(string(message)+ "\n"))
			//写入ws数据
			time.Sleep(2*time.Second)
			err = ws.WriteMessage(mt, []byte("ok"))
			if err != nil {
				break
			}
		}
	})
	r.Run()
}
