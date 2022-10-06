import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:http/http.dart' as http;
import 'package:firebase_core/firebase_core.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';

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
    await Firebase.initializeApp();
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
  late DateTime initialTime;
  late final Timer timer;
  Duration elapsed = Duration.zero;
  String elapsedString = "0";
  String status = "ER";
  Color? statusColor = Colors.red[200];
  Image image = Image.asset('assets/mate_logo.png', width: 360, height: 240);
  final ConnectivityResult connectionStatus = ConnectivityResult.none;
  final Connectivity connectivity = Connectivity();
  late StreamSubscription<ConnectivityResult> connectivitySubscription;
  ConnectivityResult connection = ConnectivityResult.none;
  String peopleOnPhoto = "0";
  double width = 370;
  double height = 210;

  @override
  void initState() {
    super.initState();
    connectivitySubscription =
        connectivity.onConnectivityChanged.listen((ConnectivityResult result) {
          connection = result;
        });
    _refreshPicture();
    timer = Timer.periodic(const Duration(minutes: 1), (_)
    {
      final now = DateTime.now();
      setState(() {
        elapsed = now.difference(initialTime);
        if (elapsed.inMinutes > 60) {
          elapsedString = ">60";
        } else {
          elapsedString = elapsed.inMinutes.toString();
        }
      });
    });
  }

  Future<String> _getPicture() async {
    final response = await http.get(Uri.parse('http://82.146.33.179:8000/api/v1/image'));
    if (response.statusCode == 200) {
      final octStream = response.body.substring(response.body.indexOf('application/octet-stream') + 28);
      final imageStream = octStream.substring(0, octStream.indexOf(RegExp(r'--[0-9a-z]*--')));
      image = Image.memory(Uint8List.fromList(imageStream.codeUnits), width: 360, height: 240);
      final peopleNumber = response.body.substring(response.body.indexOf('"people_number":') + 16);
      peopleOnPhoto = peopleNumber.substring(0, peopleNumber.indexOf('}'));
      final createdAt = response.body.substring(response.body.indexOf('"created_at":') + 14);
      initialTime = DateTime.parse(createdAt.substring(0, createdAt.indexOf(',') - 1));
      return "OK";
    } else {
      throw Exception('Failed to load');
    }
  }

  void _refreshPicture() {
    setState(() {
      _getPicture();
        if (connection != ConnectivityResult.none) {
          status = "OK";
          statusColor = Colors.green[200];
          elapsed = DateTime.now().difference(initialTime);
          if (elapsed.inMinutes > 60) {
            elapsedString = ">60";
          } else {
            elapsedString = elapsed.inMinutes.toString();
          }
        } else {
          status = "ER";
          statusColor = Colors.red[200];
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
                          ClipRRect(
                            borderRadius: BorderRadius.circular(20),
                            child : Center(child: image)
                          )])),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                  children: <Widget>[
                          Column(children:<Widget>[Stack(
                            children: <Widget>[Container(
                                padding: const EdgeInsets.all(3),
                                decoration: const BoxDecoration(color: Colors.black, shape: BoxShape.circle),
                                child: Container(
                                  width: 80.0,
                                  height: 80.0,
                                  decoration: BoxDecoration(
                                    color: Colors.brown[200],
                                    shape: BoxShape.circle,
                                  ),
                                  child: Center(widthFactor: 2.7, heightFactor: 1.4, child: Text(elapsedString, style: const TextStyle(fontSize: 40))),
                                )
                            ),
                            ]
                            ),
                          const Text("minutes ago\n   updated", style: TextStyle(fontSize: 15),)]),
                          Column(children:<Widget>[Stack(
                            children: <Widget>[Container(
                            padding: const EdgeInsets.all(3),
                            decoration: const BoxDecoration(color: Colors.black, shape: BoxShape.circle),
                            child: Container(
                              width: 80.0,
                              height: 80.0,
                              decoration: BoxDecoration(
                                color: Colors.brown[200],
                                shape: BoxShape.circle,
                              ),
                              child: Center(widthFactor: 2.7, heightFactor: 1.4, child: Text(peopleOnPhoto, style: const TextStyle(fontSize: 40))),
                            )
                        ),
                            ]),
                            const Text("people on\n    photo", style: TextStyle(fontSize: 15),)]),
                          Column(children:<Widget>[Stack(
                            children: <Widget>[Container(
                            padding: const EdgeInsets.all(3),
                            decoration: const BoxDecoration(color: Colors.black, shape: BoxShape.circle),
                            child: Container(
                              width: 80.0,
                              height: 80.0,
                              decoration: BoxDecoration(
                                color: statusColor,
                                shape: BoxShape.circle,
                              ),
                              child: Center(widthFactor: 1.5, heightFactor: 1.8, child: Text(status, style: const TextStyle(fontSize: 40)))),
                            )]
                        ),

                            const Text("connection\n    status", style: TextStyle(fontSize: 15),)])
                          ]
                ),
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
                )
            ],
      ), // This trailing comma makes auto-formatting nicer for build methods.
    );
  }
}
