import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:cached_network_image/cached_network_image.dart';
import 'package:connectivity_plus/connectivity_plus.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
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
  final ConnectivityResult connectionStatus = ConnectivityResult.none;
  final Connectivity connectivity = Connectivity();
  late StreamSubscription<ConnectivityResult> connectivitySubscription;
  late CachedNetworkImage image;
  ConnectivityResult connection = ConnectivityResult.none;
  int amount = 1;
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
    initialTime = DateTime.now();
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

  void _refreshPicture() {
    setState(() {
        if (connection != ConnectivityResult.none) {
          amount++;
          status = "OK";
          statusColor = Colors.green[200];
          initialTime = DateTime.now();
          elapsed = Duration.zero;
          elapsedString = "0";
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
                            child : CachedNetworkImage(imageUrl: 'http://82.146.33.179:8000/api/image/$amount', width: width, height: height),
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
                              child: Center(widthFactor: 2.7, heightFactor: 1.4, child: Text(amount.toString(), style: const TextStyle(fontSize: 40))),
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
