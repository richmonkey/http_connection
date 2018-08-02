all:tcp_connection http_connection

tcp_connection:tcp_connection.go connection.go client.go protocol.go message.go  config.go monitoring.go user.go user_manager.go user_state_deliver.go sio.go
	go build -o tcp_connection tcp_connection.go connection.go client.go protocol.go message.go  config.go monitoring.go user.go user_manager.go user_state_deliver.go sio.go


http_connection:http_connection.go config.go monitoring.go user.go user_manager.go user_state_deliver.go 
	go build -o http_connection http_connection.go  config.go monitoring.go user.go user_manager.go user_state_deliver.go


clean:
	rm -f tcp_connection http_connection
