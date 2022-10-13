import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:http/http.dart' as http;
import 'package:firebase_core/firebase_core.dart';
import 'firebase_options.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';

class ImageWithInfo {
  late Image image;
  late DateTime time;
  late String peopleAmount;

  ImageWithInfo(Image i, DateTime t, String pa) {
    image = i;
    time = t;
    peopleAmount = pa;
  }
}

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  Future<void> _channelSetUp (AndroidNotificationChannel channel, FlutterLocalNotificationsPlugin flutterLocalNotificationsPlugin) async {
  await flutterLocalNotificationsPlugin
      .resolvePlatformSpecificImplementation<AndroidFlutterLocalNotificationsPlugin>()
      ?.createNotificationChannel(channel);
  }

  Future<void> _initFirebase() async {
    await Firebase.initializeApp(
      options: DefaultFirebaseOptions.currentPlatform,
    );
  }

  @override
  Widget build(BuildContext context) {
    _initFirebase().then((value) => (){
      const AndroidNotificationChannel channel = AndroidNotificationChannel(
        'high_importance_channel', // id
        'High Importance Notifications', // title
        importance: Importance.max,
      );

          final FlutterLocalNotificationsPlugin flutterLocalNotificationsPlugin =
          FlutterLocalNotificationsPlugin();

      _channelSetUp(channel, flutterLocalNotificationsPlugin);

      FirebaseMessaging.instance
          .getInitialMessage()
          .then((RemoteMessage? message) {
        if (message != null) {
          Navigator.pushNamed(context, message.data['view']);
        }
      });
    });

    return MaterialApp(
      title: 'MATE Security System',
      theme: ThemeData(
        fontFamily: 'Inter',
        primarySwatch: Colors.brown,
      ),
      home: const MyHomePage(title: 'MATE Security System'),
    );
  }
}

class MyHomePage extends StatefulWidget {
  const MyHomePage({super.key, required this.title});

  final String title;

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  Duration elapsed = Duration.zero;
  String elapsedString = "0";
  String status = "ER";
  Color? statusColor = Colors.red[200];
  final ConnectivityResult connectionStatus = ConnectivityResult.none;
  final Connectivity connectivity = Connectivity();
  late StreamSubscription<ConnectivityResult> connectivitySubscription;
  ConnectivityResult connection = ConnectivityResult.none;
  double width = 370;
  double height = 240;
  double next = 0;
  double previous = 0;
  double buttonOn = 1;
  double buttonOff = 0;
  var images = [ImageWithInfo(Image.asset('assets/mate_logo.png', width: 370, height: 240), DateTime.now(), "0")];
  int current = 0;
  int max = 0;
  String text = "minutes ago\n   updated";

  @override
  void initState() {
    _refreshPicture();
    super.initState();
    connectivitySubscription =
        connectivity.onConnectivityChanged.listen((ConnectivityResult result) {
          connection = result;
        });
    Timer.periodic(const Duration(minutes: 1), (_)
    {
      setState(() {
        updateTime();
      });
    });
  }

  Future<void> _getPicture() async {
    final response = await http.get(Uri.parse('http://82.146.33.179:8000/api/v1/image'));
    if (response.statusCode == 200) {
      final octStream = response.body.substring(response.body.indexOf('application/octet-stream') + 28);
      final imageStream = octStream.substring(0, octStream.indexOf(RegExp(r'--[0-9a-z]*--')));
      final peopleNumber = response.body.substring(response.body.indexOf('"people_number":') + 16);
      final createdAt = response.body.substring(response.body.indexOf('"created_at":') + 14);
      final initialTime = DateTime.parse(createdAt.substring(0, createdAt.indexOf(',') - 1));
      if (images[current].time != initialTime) {
        ImageWithInfo newPhoto = ImageWithInfo(
            Image.memory(
                Uint8List.fromList(imageStream.codeUnits),
                width: width,
                height: height
            ),
            initialTime,
            peopleNumber.substring(0, peopleNumber.indexOf('}')));
        images.add(newPhoto);
        max++;
        current = max;
        if (current > 1) {
          previous = buttonOn;
        }
        next = buttonOff;
        updateTime();
      }
      status = "OK";
      statusColor = Colors.green[200];
    } else {
      status = "ER";
      statusColor = Colors.red[200];
    }
  }

  void _refreshPicture() {
    setState(() {
      _getPicture();
    });
  }

  void updateTime() {
    elapsed = DateTime.now().difference(images[current].time);
    if (elapsed.inHours > 24) {
      text = " days ago\n updated";
      elapsedString = elapsed.inDays.toString();
    } else if (elapsed.inMinutes > 60) {
      text = "hours ago\n  updated";
      elapsedString = elapsed.inHours.toString();
    } else {
      text = "minutes ago\n   updated";
      elapsedString = elapsed.inMinutes.toString();
    }
  }

  void increaseCurrent() {
    setState(() {
      if (current < max) {
        current++;
        updateTime();
        previous = buttonOn;
      }
      if (current == max) {
        next = buttonOff;
      }
    });
  }

  void decreaseCurrent() {
    setState(() {
      if (current > 1) {
        current--;
        updateTime();
        next = buttonOn;
      }
      if (current == 1) {
        previous = buttonOff;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    SystemChrome.setPreferredOrientations([DeviceOrientation.portraitUp]);
    return Scaffold(
      backgroundColor: Colors.brown[400],
      appBar: AppBar(
        title: Text(widget.title),
      ),
      body: Column(
          mainAxisAlignment: MainAxisAlignment.spaceEvenly,
          children:
            <Widget>[
                Container(
                    padding: const EdgeInsets.all(2),
                    decoration: BoxDecoration(color: Colors.black, borderRadius: BorderRadius.circular(20)),
                    child: Stack(
                        children: <Widget>[
                          SizedBox(width: width, height: height, child: const Center(child:CircularProgressIndicator(strokeWidth: 4))),
                          SizedBox(
                              width: width,
                              height: height,
                              child: images[current].image)])),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                  children: <Widget>[
                    Column(
                        children:<Widget>[
                          Stack(
                            children: <Widget>[
                              Container(
                                padding: const EdgeInsets.all(3),
                                decoration: const BoxDecoration(color: Colors.black, shape: BoxShape.circle),
                                child: Container(
                                  width: 80.0,
                                  height: 80.0,
                                  decoration: BoxDecoration(
                                    color: Colors.brown[200],
                                    shape: BoxShape.circle,
                                  ),
                                  child: Center(widthFactor: 2.7, heightFactor: 1.4, child: Text(elapsedString, style: const TextStyle(fontSize: 40)))
                                )
                              )
                            ]
                          ),
                          Text(text, style: const TextStyle(fontSize: 15))
                        ]
                    ),
                    Column(
                        children:<Widget>[
                          Stack(
                              children: <Widget>[
                                Container(
                                    padding: const EdgeInsets.all(3),
                                    decoration: const BoxDecoration(color: Colors.black, shape: BoxShape.circle),
                                    child: Container(
                                        width: 80.0,
                                        height: 80.0,
                                        decoration: BoxDecoration(
                                          color: Colors.brown[200],
                                          shape: BoxShape.circle,
                                        ),
                                        child: Center(widthFactor: 2.7, heightFactor: 1.4, child: Text(images[current].peopleAmount, style: const TextStyle(fontSize: 40)))
                                    )
                                )
                              ]
                          ),
                          const Text("people on\n    photo", style: TextStyle(fontSize: 15))
                        ]
                    ),
                    Column(
                        children:<Widget>[
                          Stack(
                              children: <Widget>[
                                Container(
                                    padding: const EdgeInsets.all(3),
                                    decoration: const BoxDecoration(color: Colors.black, shape: BoxShape.circle),
                                    child: Container(
                                        width: 80.0,
                                        height: 80.0,
                                        decoration: BoxDecoration(
                                          color: statusColor,
                                          shape: BoxShape.circle,
                                        ),
                                        child: Center(widthFactor: 1.5, heightFactor: 1.8, child: Text(status, style: const TextStyle(fontSize: 40))))
                                )
                              ]
                          ),
                          const Text("connection\n    status", style: TextStyle(fontSize: 15))
                        ]
                    )
                  ]
                ),
                Column(
                  children: [
                    Container(
                        padding: const EdgeInsets.all(2),
                        decoration: BoxDecoration(color: Colors.black, borderRadius: BorderRadius.circular(20)),
                        child: ClipRRect(
                            borderRadius: BorderRadius.circular(20),
                            child: TextButton(
                                style: TextButton.styleFrom(
                                  fixedSize: const Size(320, 60),
                                  foregroundColor: Colors.black,
                                  backgroundColor: Colors.brown[200],
                                ),
                                onPressed: _refreshPicture,
                                child: const Text(
                                  "Update Now",
                                  style: TextStyle(
                                      fontSize: 35
                                  ),
                                )
                            )
                        )
                    ),
                    Center(
                      child:
                        Row(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            Opacity(
                                opacity: previous,
                                child:
                                  Container(
                                      padding: const EdgeInsets.all(2),
                                      decoration: BoxDecoration(color: Colors.black, borderRadius: BorderRadius.circular(20)),
                                      child: ClipRRect(
                                          borderRadius: BorderRadius.circular(20),
                                          child: TextButton(
                                              style: TextButton.styleFrom(
                                                fixedSize: const Size(160, 60),
                                                foregroundColor: Colors.black,
                                                backgroundColor: Colors.brown[200],
                                              ),
                                              onPressed: decreaseCurrent,
                                              child: const Text(
                                                "Previous",
                                                style: TextStyle(
                                                    fontSize: 35
                                                ),
                                              )
                                          )
                                      )
                                  )
                            ),
                          Opacity(
                            opacity: next,
                            child:
                              Container(
                                  padding: const EdgeInsets.all(2),
                                  decoration: BoxDecoration(color: Colors.black, borderRadius: BorderRadius.circular(20)),
                                  child: ClipRRect(
                                      borderRadius: BorderRadius.circular(20),
                                      child: TextButton(
                                          style: TextButton.styleFrom(
                                            fixedSize: const Size(160, 60),
                                            foregroundColor: Colors.black,
                                            backgroundColor: Colors.brown[200],
                                          ),
                                          onPressed: increaseCurrent,
                                          child: const Text(
                                            "Next",
                                            style: TextStyle(
                                                fontSize: 35
                                            ),
                                          )
                                      )
                                  )
                              )
                            )
                          ],
                        )
                    )
                  ],
                )
            ],
      ), // This trailing comma makes auto-formatting nicer for build methods.
    );
  }
}
