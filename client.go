package main

import "net"
import "time"
import "sync/atomic"
import log "github.com/golang/glog"
import "container/list"

type Client struct {
	Connection//必须放在结构体首部
	public_ip int32
}

func NewClient(conn interface{}) *Client {
	client := new(Client)

	//初始化Connection
	client.conn = conn // conn is net.Conn or engineio.Conn

	if net_conn, ok := conn.(net.Conn); ok {
		addr := net_conn.LocalAddr()
		if taddr, ok := addr.(*net.TCPAddr); ok {
			ip4 := taddr.IP.To4()
			client.public_ip = int32(ip4[0]) << 24 | int32(ip4[1]) << 16 | int32(ip4[2]) << 8 | int32(ip4[3])
		}
	}

	client.wt = make(chan *Message, 300)
	client.lwt = make(chan int, 1)//only need 1
	client.messages = list.New()
	
	atomic.AddInt64(&server_summary.nconnections, 1)

	return client
}

func (client *Client) Read() {
	for {
		tc := atomic.LoadInt32(&client.tc)
		if tc > 0 {
			log.Infof("quit read goroutine, client:%d write goroutine blocked", client.uid)
			client.HandleClientClosed()
			break
		}

		t1 := time.Now().Unix()
		msg := client.read()
		t2 := time.Now().Unix()
		if t2 - t1 > 6*60 {
			log.Infof("client:%d socket read timeout:%d %d", client.uid, t1, t2)
		}
		if msg == nil {
			client.HandleClientClosed()
			break
		}

		client.HandleMessage(msg)
		t3 := time.Now().Unix()
		if t3 - t2 > 2 {
			log.Infof("client:%d handle message is too slow:%d %d", client.uid, t2, t3)
		}
	}
}

func (client *Client) AddClient() {
	user_manager.AddUser(client.appid, client.uid)
	deliver := GetUserStateDeliver(client.uid)
	deliver.DeliverUserState(client.appid, client.uid)
}

func (client *Client) RemoveClient() {
	if client.uid > 0 {
		user_manager.RemoveUser(client.appid, client.uid)
		deliver := GetUserStateDeliver(client.uid)
		deliver.DeliverUserState(client.appid, client.uid)
	}
}

func (client *Client) HandleClientClosed() {
	atomic.AddInt64(&server_summary.nconnections, -1)
	if client.uid > 0 {
		atomic.AddInt64(&server_summary.nclients, -1)
	
	}
	atomic.StoreInt32(&client.closed, 1)

	client.RemoveClient()
	//quit when write goroutine received
	client.wt <- nil
}

func (client *Client) HandleMessage(msg *Message) {
	log.Info("msg cmd:", Command(msg.cmd))
	switch msg.cmd {
	case MSG_AUTH_TOKEN:
		client.HandleAuthToken(msg.body.(*AuthenticationToken), msg.version)
	case MSG_PING:
		client.HandlePing()
	}
}


func (client *Client) AuthToken(token string) (int64, int64, int, bool, error) {
	appid, uid, forbidden, notification_on, err := LoadUserAccessToken(token)

	if err != nil {
		return 0, 0, 0, false, err
	}

	return appid, uid, forbidden, notification_on, nil
}


func (client *Client) HandleAuthToken(login *AuthenticationToken, version int) {
	if client.uid > 0 {
		log.Info("repeat login")
		return
	}

	var err error
	appid, uid, _, _, err := client.AuthToken(login.token)
	if err != nil {
		log.Infof("auth token:%s err:%s", login.token, err)
		msg := &Message{cmd: MSG_AUTH_STATUS, version:version, body: &AuthenticationStatus{1}}
		client.EnqueueMessage(msg)
		return
	}
	if  uid == 0 {
		log.Info("auth token uid==0")
		msg := &Message{cmd: MSG_AUTH_STATUS, version:version, body: &AuthenticationStatus{1}}
		client.EnqueueMessage(msg)
		return
	}
	
	client.appid = appid
	client.uid = uid
	client.version = version

	client.tm = time.Now()
	log.Infof("auth token:%s appid:%d uid:%d", login.token, client.appid, client.uid)

	msg := &Message{cmd: MSG_AUTH_STATUS, version:version, body: &AuthenticationStatus{0}}
	client.EnqueueMessage(msg)

	client.AddClient()

	atomic.AddInt64(&server_summary.nclients, 1)
}


func (client *Client) HandlePing() {
	m := &Message{cmd: MSG_PONG}
	client.EnqueueMessage(m)
	if client.uid == 0 {
		log.Warning("client has't been authenticated")
		return
	}
}


//发送等待队列中的消息
func (client *Client) SendMessages(seq int) int {
	var messages *list.List
	client.mutex.Lock()
	if (client.messages.Len() == 0) {
		client.mutex.Unlock()		
		return seq
	}
	messages = client.messages
	client.messages = list.New()
	client.mutex.Unlock()

	e := messages.Front();	
	for e != nil {
		msg := e.Value.(*Message)
		
		seq++
		//以当前客户端所用版本号发送消息
		vmsg := &Message{msg.cmd, seq, client.version, msg.flag, msg.body}
		client.send(vmsg)
		
		e = e.Next()
	}
	return seq
}

func (client *Client) Write() {
	seq := 0
	running := true
	
	//发送在线消息
	for running {
		select {
		case msg := <-client.wt:
			if msg == nil {
				client.close()
				running = false
				log.Infof("client:%d socket closed", client.uid)
				break
			}
			
			seq++
			//以当前客户端所用版本号发送消息
			vmsg := &Message{msg.cmd, seq, client.version, msg.flag, msg.body}
			client.send(vmsg)
		case <- client.lwt:
			seq = client.SendMessages(seq)
			break
		}
	}

	//等待200ms,避免发送者阻塞
	t := time.After(200 * time.Millisecond)
	running = true
	for running {
		select {
		case <- t:
			running = false
		case <- client.wt:
			log.Warning("msg is dropped")
		}
	}

	log.Info("write goroutine exit")
}

func (client *Client) Run() {
	go client.Write()
	go client.Read()
}
