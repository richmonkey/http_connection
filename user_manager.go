package main
import "sync"

type UserID struct {
	appid int64
	uid   int64
}

type UserManager struct {
	users map[UserID]int
	mutex  sync.Mutex
}

func NewUserManager() *UserManager {
	um := &UserManager{}
	um.users = make(map[UserID]int)
	return um
}



//用户上线
//返回add之前的连接计数
func (um *UserManager) AddUser(appid int64, uid int64) int {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	id := UserID{appid:appid, uid:uid}
	
	if c, ok := um.users[id]; ok {
		um.users[id] = c + 1
		return c
	} else {
		um.users[id] = 1
		return 0
	}
}

//用户下线,
//返回remove之前的连接计数
func (um *UserManager) RemoveUser(appid int64, uid int64) int {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	id := UserID{appid:appid, uid:uid}

	if c, ok := um.users[id]; ok {
		c -= 1
		if c <= 0 {
			delete(um.users, id)
		} else {
			um.users[id] = c
		}
		return c+1
	} else {
		return 0
	}
}

//获取用户连接数
func (um *UserManager) GetUser(appid int64, uid int64) int {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	id := UserID{appid:appid, uid:uid}
	return um.users[id]
}
