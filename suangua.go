package suangua

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/fs"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/Logiase/MiraiGo-Template/bot"
	"github.com/Logiase/MiraiGo-Template/utils"
	"github.com/Mrs4s/MiraiGo/client"
	"github.com/Mrs4s/MiraiGo/message"
)

//go:embed assets/gua
var guaImages embed.FS

//go:embed assets/64.json
var guaResultJSONData []byte

var instance *logging
var logger = utils.GetModuleLogger("com.aimerneige.suangua")

type logging struct {
}

func init() {
	instance = &logging{}
	bot.RegisterModule(instance)
}

func (l *logging) MiraiGoModule() bot.ModuleInfo {
	return bot.ModuleInfo{
		ID:       "com.aimerneige.suangua",
		Instance: instance,
	}
}

// Init 初始化过程
// 在此处可以进行 Module 的初始化配置
// 如配置读取
func (l *logging) Init() {
}

// PostInit 第二次初始化
// 再次过程中可以进行跨 Module 的动作
// 如通用数据库等等
func (l *logging) PostInit() {
}

// Serve 注册服务函数部分
func (l *logging) Serve(b *bot.Bot) {
	b.GroupMessageEvent.Subscribe(func(c *client.QQClient, msg *message.GroupMessage) {
		solveSuangua(c, msg.ToString(), msg.Sender.Uin, message.Source{
			SourceType: message.SourceGroup,
			PrimaryID:  msg.GroupCode,
		})
	})
	b.PrivateMessageEvent.Subscribe(func(c *client.QQClient, msg *message.PrivateMessage) {
		solveSuangua(c, msg.ToString(), msg.Sender.Uin, message.Source{
			SourceType: message.SourcePrivate,
			PrimaryID:  msg.Sender.Uin,
		})
	})
}

// Start 此函数会新开携程进行调用
// ```go
//
//	go exampleModule.Start()
//
// ```
// 可以利用此部分进行后台操作
// 如 http 服务器等等
func (l *logging) Start(b *bot.Bot) {
}

// Stop 结束部分
// 一般调用此函数时，程序接收到 os.Interrupt 信号
// 即将退出
// 在此处应该释放相应的资源或者对状态进行保存
func (l *logging) Stop(b *bot.Bot, wg *sync.WaitGroup) {
	// 别忘了解锁
	defer wg.Done()
}

// solveSuangua 处理算卦请求
func solveSuangua(c *client.QQClient, msg string, senderUin int64, target message.Source) {
	// 解析消息指令
	status, things := msgParse(msg)
	// 解析失败，忽略消息
	if !status {
		return
	}
	// 获取算卦消息
	var sendingMsg *message.SendingMessage
	if things == "" {
		// 用户没有发送事项，使用默认消息（索引 0）
		sendingMsg = getSuanguaMessage(c, target, uint32(0))
	} else {
		// 计算事项 Hash
		thingsHash := calHash(things, senderUin)
		sendingMsg = getSuanguaMessage(c, target, thingsHash)
	}
	// 消息未赋值，返回
	if sendingMsg == nil {
		return
	}
	// 发送消息
	switch target.SourceType {
	case message.SourceGroup:
		c.SendGroupMessage(target.PrimaryID, sendingMsg)
	case message.SourcePrivate:
		c.SendPrivateMessage(target.PrimaryID, sendingMsg)
	}
}

// msgParse 解析消息
func msgParse(msg string) (bool, string) {
	msg = strings.TrimSpace(msg)
	if !strings.HasPrefix(msg, "算卦") {
		return false, ""
	}
	msg = msg[6:]
	things := strings.TrimSpace(msg)
	return true, things
}

// calHash 计算 Hash
func calHash(things string, uin int64) uint32 {
	unixTime := uint32(time.Now().Unix() / 10000)
	thingsHash := stringHash(things)
	uinHash := stringHash(fmt.Sprint(uin))
	return (unixTime+thingsHash+uinHash)%64 + 1
}

// 获取算卦结果消息
func getSuanguaMessage(c *client.QQClient, target message.Source, i uint32) *message.SendingMessage {
	var guaResultJSONObj []string
	if err := json.Unmarshal(guaResultJSONData, &guaResultJSONObj); err != nil {
		logger.Panic("Assets JSON unmarshal error!")
	}
	text := guaResultJSONObj[i]
	imgPath := path.Join("assets/gua", fmt.Sprintf("%d.jpg", i))
	imgFile, err := fs.ReadFile(guaImages, imgPath)
	if err != nil {
		logger.WithError(err).Errorf("Fail to read img %s", imgPath)
		return simpleText("卦象图片不见了呢！好像发生什么奇怪的玄学事故。")
	}
	uploadedImg, err := c.UploadImage(target, bytes.NewReader(imgFile))
	if err != nil {
		logger.WithError(err).Error("Fail to upload image.")
		return simpleText("卦象图片被玄学风控")
	}
	return simpleText(text).Append(uploadedImg)
}

func simpleText(s string) *message.SendingMessage {
	return message.NewSendingMessage().Append(message.NewText(s))
}

func stringHash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
