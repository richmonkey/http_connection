/**
 * Copyright (c) 2014-2015, GoBelieve     
 * All rights reserved.
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 2 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  USA
 */

package main

import "fmt"
import log "github.com/golang/glog"
import "github.com/gomodule/redigo/redis"
import "errors"

func LoadUserAccessToken(token string) (int64, int64, int, bool, error) {
	conn := redis_pool.Get()
	defer conn.Close()

	key := fmt.Sprintf("access_token_%s", token)
	var uid int64
	var appid int64
	var notification_on int8
	var forbidden int
	
	exists, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return 0, 0, 0, false, err
	}
	if !exists {
		return 0, 0, 0, false,  errors.New("token non exists")
	}

	reply, err := redis.Values(conn.Do("HMGET", key, "user_id",
		"app_id", "notification_on", "forbidden"))
	if err != nil {
		log.Info("hmget error:", err)
		return 0, 0, 0, false, err
	}

	_, err = redis.Scan(reply, &uid, &appid, &notification_on, &forbidden)
	if err != nil {
		log.Warning("scan error:", err)
		return 0, 0, 0, false, err
	}
	
	return appid, uid, forbidden, notification_on != 0, nil	
}
