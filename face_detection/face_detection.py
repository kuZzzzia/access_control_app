import cv2
from datetime import datetime
import threading as th
from argparse import ArgumentParser
import websocket
import requests
import os

req_lock = th.Lock()
got_req = False
keep_status = True
server = ""
port = ""

def on_message(wsapp, message):
    with req_lock:
        got_req = True

wsapp = websocket.WebSocketApp("ws://"+server+":"+port+"/msg", on_message=on_message)
wsapp.run_forever()

def send(date, count, img):
    url = "http://"+server+":"+port+"/img"
    r = requests.post(url, data={'created_at':date, 'people_number':count}, files={'img': img})


def wait_input():
    global keep_status
    input()
    keep_status = False

def detect_faces():
    face_cascade = cv2.CascadeClassifier('haarcascade_frontalface_default.xml')

    cap = cv2.VideoCapture(0)
    prev_amount = 0

    frame_count = 0
    fps = int(cap.get(cv2.CAP_PROP_FPS))

    #waiting for any key pressed in console
    th.Thread(target=wait_input, args=(), name='wait_input', daemon=True).start() 
    while keep_status:
        _, img = cap.read()
        dt = datetime.now()

        if frame_count % fps == 0:
            frame_count = 0
            gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)

            faces = face_cascade.detectMultiScale(gray, 1.1, 4)
            people_amount = len(faces)

            #color in BGR format
            with req_lock:
                if got_req or prev_amount != people_amount:
                    for (x, y, w, h) in faces:
                        cv2.rectangle(img, (x, y), (x+w, y+h), (0, 255, 255), 1)
                    prev_amount = people_amount
                    dt_str = dt.strftime("%d-%m-%Y_%H:%M:%S")
                    name = "./"+dt_str+".jpg"
                    _ = cv2.imwrite(name, img)
                    send(dt_str, people_amount, open(name, 'rb'))
                    os.remove(name)
                    got_req = False
        frame_count += 1

    cap.release()

if __name__ == '__main__':
    parser = ArgumentParser()
    parser.add_argument("-s", "--server", required=True, help="server address", metavar="s")
    parser.add_argument("-p", "--port", required=True, help="port", metavar="p")

    args = parser.parse_args()
    server = args.server
    port = args.port
    detect_faces()

