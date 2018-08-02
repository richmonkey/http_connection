
package main
import "net"
import "fmt"
import "flag"
import "time"
import "runtime"
import "math/rand"
import "net/http"
import "github.com/gomodule/redigo/redis"
import log "github.com/golang/glog"


var redis_pool *redis.Pool
var config *Config
var server_summary *ServerSummary
var user_manager *UserManager
var user_state_delivers []*UserStateDeliver

const USER_STATE_DELIVER_COUNT = 128

func init() {
	server_summary = NewServerSummary()
	user_manager = NewUserManager()
}

func handle_client(conn net.Conn) {
	log.Infoln("handle_client")
	client := NewClient(conn)
	client.Run()
}

func Listen(f func(net.Conn), port int) {
	listen_addr := fmt.Sprintf("0.0.0.0:%d", port)
	listen, err := net.Listen("tcp", listen_addr)
	if err != nil {
		fmt.Println("初始化失败", err.Error())
		return
	}
	tcp_listener, ok := listen.(*net.TCPListener)
	if !ok {
		fmt.Println("listen error")
		return
	}

	for {
		client, err := tcp_listener.AcceptTCP()
		if err != nil {
			return
		}
		f(client)
	}
}

func ListenClient() {
	Listen(handle_client, config.port)
}

func GetUserStateDeliver(uid int64) *UserStateDeliver {
	return user_state_delivers[uid%USER_STATE_DELIVER_COUNT]
}



func NewRedisPool(server, password string, db int) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     100,
		MaxActive:   500,
		IdleTimeout: 480 * time.Second,
		Dial: func() (redis.Conn, error) {
			timeout := time.Duration(2)*time.Second
			c, err := redis.DialTimeout("tcp", server, timeout, 0, 0)
			if err != nil {
				return nil, err
			}
			if len(password) > 0 {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			if db > 0 && db < 16 {
				if _, err := c.Do("SELECT", db); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
	}
}


type loggingHandler struct {
	handler http.Handler
}

func (h loggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Infof("http request:%s %s %s", r.RemoteAddr, r.Method, r.URL)
	h.handler.ServeHTTP(w, r)
}

func StartHttpServer(addr string) {
	http.HandleFunc("/summary", Summary)
	http.HandleFunc("/stack", Stack)

	handler := loggingHandler{http.DefaultServeMux}
	
	err := http.ListenAndServe(addr, handler)
	if err != nil {
		log.Fatal("http server err:", err)
	}
}


func main() {
	rand.Seed(time.Now().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if len(flag.Args()) == 0 {
		fmt.Println("usage: im config")
		return
	}

	config = read_cfg(flag.Args()[0])
	log.Infof("port:%d\n", config.port)
	log.Infof("http listen address:%s\n", config.http_listen_address)
	log.Infof("redis address:%s password:%s db:%d\n", 
		config.redis_address, config.redis_password, config.redis_db)
	
	redis_pool = NewRedisPool(config.redis_address, config.redis_password, 
		config.redis_db)

	
	user_state_delivers = make([]*UserStateDeliver, USER_STATE_DELIVER_COUNT)
	for i := 0; i < USER_STATE_DELIVER_COUNT; i++ {
		d := NewUserStateDeliver()
		d.Start()
		user_state_delivers[i] = d

	}
	
	go StartHttpServer(config.http_listen_address)

	ListenClient()
}
