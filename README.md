#tcp connection
监控tcp连接状态，并将连接状态发送到redis队列中
队列名:user_state_queue
队列项:$appid_$uid_$on
字段$on取1表示上线，取0表示下线
