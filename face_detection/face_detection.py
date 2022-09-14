import cv2
from datetime import datetime
import threading as th

keep_status = True

def wait_input():
    global keep_status
    input()
    keep_status = False

def detect_faces():
    face_cascade = cv2.CascadeClassifier('haarcascade_frontalface_default.xml')

    cap = cv2.VideoCapture(0)
    prev_count = 0

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
            count = len(faces)

            #color in BGR format
            if prev_count != count:
                for (x, y, w, h) in faces:
                    cv2.rectangle(img, (x, y), (x+w, y+h), (0, 255, 255), 1)
                prev_count = count
                dt_string = dt.strftime("%d-%m-%Y_%H:%M:%S")
                name = "./"+dt_string+".jpg"
                status = cv2.imwrite(name, img)
        frame_count += 1

    cap.release()

detect_faces()
