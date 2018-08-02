# -*- coding: utf-8 -*-

import struct
import socket
import threading
import time
import requests
import json
import uuid
import base64
import select
import sys
import md5

APP_ID = 7
APP_KEY = "sVDIlIiDUm7tWPYWhi6kfNbrqui3ez44"
APP_SECRET = '0WiCxAU1jh76SbgaaFC7qIaBPm2zkyM1'
URL = "http://dev.api.gobelieve.io"
DEVICE_ID = "f9d2a7c2-701a-11e5-9c3e-34363bd464b2"

HOST = "127.0.0.1"
PORT = 24000

#command
MSG_AUTH_STATUS = 3
MSG_PING = 13
MSG_PONG = 14
MSG_AUTH_TOKEN = 15



#platform
PLATFORM_IOS = 1
PLATFORM_ANDROID = 2
PLATFORM_WEB = 3
PLATFORM_SERVER = 4

PROTOCOL_VERSION = 1

class AuthenticationToken:
    def __init__(self):
        self.token = ""
        self.platform_id = PLATFORM_SERVER
        self.device_id = ""

    
def send_message(cmd, seq, msg, sock):
    if cmd == MSG_AUTH_TOKEN:
        b = struct.pack("!BB", msg.platform_id, len(msg.token)) + msg.token + struct.pack("!B", len(msg.device_id)) + msg.device_id
        length = len(b)
        h = struct.pack("!iibbbb", length, seq, cmd, PROTOCOL_VERSION, 0, 0)
        sock.sendall(h+b)
    elif cmd == MSG_PING:
        h = struct.pack("!iibbbb", 0, seq, cmd, PROTOCOL_VERSION, 0, 0)
        sock.sendall(h)
    else:
        print "eeeeee"

def recv_message(sock):
    buf = sock.recv(12)
    if len(buf) != 12:
        return 0, 0, None
    length, seq, cmd = struct.unpack("!iib", buf[:9])

    if length == 0:
        return cmd, seq, None

    content = sock.recv(length)
    if len(content) != length:
        return 0, 0, None

    if cmd == MSG_AUTH_STATUS:
        status, = struct.unpack("!i", content)
        return cmd, seq, status
    elif cmd == MSG_PONG:
        return cmd, seq, None
    else:
        return cmd, seq, content

class Client(object):
    def __init__(self):
        self.seq = 0
        self.sock = None

    def connect_server(self, device_id, token, host=None):
        if host is not None:
            address = (host, PORT)
        else:
            address = (HOST, PORT)
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)  
        sock.connect(address)
        auth = AuthenticationToken()
        auth.token = token
        auth.device_id = device_id
        self.seq = self.seq + 1
        send_message(MSG_AUTH_TOKEN, self.seq, auth, sock)
        cmd, _, msg = recv_message(sock)
        if cmd != MSG_AUTH_STATUS or msg != 0:
            return False

        self.sock = sock
        return True

    def close(self):
        self.sock.close()

    def recv_message(self):
        while True:
            rlist, _, xlist = select.select([self.sock], [], [self.sock], 30)
            if not rlist and not xlist:
                #timeout
                self.seq += 1
                print "ping..."
                send_message(MSG_PING, self.seq, None, self.sock)
                continue
            if xlist:
                return 0, 0, None
            if rlist:
                cmd, s, m = recv_message(self.sock)
                return cmd, s, m



def _login(appid, app_secret, uid):
    url = URL + "/auth/grant"
    obj = {"uid":uid, "user_name":str(uid)}
    secret = md5.new(app_secret).digest().encode("hex")
    basic = base64.b64encode(str(appid) + ":" + secret)
    headers = {'Content-Type': 'application/json; charset=UTF-8',
               'Authorization': 'Basic ' + basic}
     
    res = requests.post(url, data=json.dumps(obj), headers=headers)
    if res.status_code != 200:
        print res.status_code, res.content
        return None
    obj = json.loads(res.text)
    return obj["data"]["token"]

def login(uid):
    return _login(APP_ID, APP_SECRET, uid)


if __name__ == "__main__":

    token = login(1)
    print "token:", token

    while True:
        try:
            client = Client()
            r = client.connect_server(DEVICE_ID, token)
            if not r:
                continue
            while True:
                print "recv message..."
                cmd, s, m = client.recv_message()
                #socket disconnect
                if cmd == 0 and s == 0 and m is None:
                    print "socket disconnect"
                    break

                print "cmd:", cmd
             
                if cmd == MSG_PONG:
                    print "pong..."
                    continue
                else:
                    print "unknow message:", cmd
                    continue

        except Exception, e:
            print "exception:", e
            time.sleep(1)
            continue
