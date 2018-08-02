package main
import "time"
import "fmt"
import "sync/atomic"
import log "github.com/golang/glog"

type UserStateDeliver struct {
	wt chan *UserID
}

func NewUserStateDeliver() *UserStateDeliver {
	usd := &UserStateDeliver{}
	usd.wt = make(chan *UserID, 10000)
	return usd
}

func (usd *UserStateDeliver) DeliverUserState(appid int64, uid int64) bool {
	id := &UserID{appid:appid, uid:uid}
	select {
	case usd.wt <- id:
		return true
	case <- time.After(60*time.Second):
		log.Infof("deliver user state to wt timed out:%d, %d", appid, uid)
		atomic.AddInt64(&server_summary.nerrors, 1)
		return false
	}
}

func (usd *UserStateDeliver) deliver(ids map[UserID]struct{}) {
	conn := redis_pool.Get()
	defer conn.Close()
	
	begin := time.Now()	
	conn.Send("MULTI")	
	for u, _ := range(ids) {
		c := user_manager.GetUser(u.appid, u.uid)
		on := 0
		if c > 0 {
			on = 1
		}
		content := fmt.Sprintf("%d_%d_%d", u.appid, u.uid, on)
		log.Infof("rpush user state:%d %d %d", u.appid, u.uid, on)
		conn.Send("RPUSH", "user_event_queue", content)
	}
	_, err := conn.Do("EXEC")
	
	end := time.Now()
	duration := end.Sub(begin)
	if err != nil {
		log.Info("multi rpush error:", err)
	} else {
		log.Infof("mmulti rpush:%d time:%s success", len(ids), duration)
	}
}

func (usd *UserStateDeliver) run() {
	for {
		ids := make(map[UserID]struct{})
		
		id := <- usd.wt
		if _, ok := ids[*id]; !ok {
			ids[*id] = struct{}{}
		}

	Loop:
		for {
			select {
			case id = <- usd.wt:
				if _, ok := ids[*id]; !ok {
					ids[*id] = struct{}{}
				}
			default:
				break Loop
			}
		}
		usd.deliver(ids)
	}
}

func (usd *UserStateDeliver) Start() {
	go usd.run()
}
