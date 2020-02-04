from flask import Flask, request, jsonify

app = Flask(__name__)
app.config["DEBUG"] = True


data1 = [{"first_time":"2020-01-21T14:17:19","frequency":2412000,"kismet.device.base.last_time":"2020-01-21T14:47:38",
"kismet.device.base.macaddr":"B4:79:C8:45:D5:E8","manuf":"Ruckus Wireless",
"min_signal":-77,"max_signal":-19,"minute_vec_signal_avg":-47.8620689655,"minute_vec_signal_med":-49.0,"senal.suavizada":-47.9385856224,
"distancia_senal_promedio_f1":33.0837062187,"distancia_senal_mediana_f1":1.6319201783,
"distancia_senal_promedio_f2":7.818140093,"distancia_senal_mediana_f2":0.5}]

data2 = [{"first_time":"2020-01-21T14:17:19","frequency":2412000,"kismet.device.base.last_time":"2020-01-21T14:47:40","kismet.device.base.macaddr":"B4:79:C8:45:D5:E8","manuf":"Ruckus Wireless",
"min_signal":-77,"max_signal":-19,"minute_vec_signal_avg":-48.0508474576,"minute_vec_signal_med":-49.0,"senal.suavizada":-47.9282057276,
"distancia_senal_promedio_f1":34.0997056409,"distancia_senal_mediana_f1":2.6319201783,
"distancia_senal_promedio_f2":7.9899189324,"distancia_senal_mediana_f2":0.5}]

data3 = [{"first_time":"2020-01-21T14:17:19","frequency":2412000,"kismet.device.base.last_time":"2020-01-21T14:47:40","kismet.device.base.macaddr":"B4:79:C8:45:D5:E8","manuf":"Ruckus Wireless",
"min_signal":-77,"max_signal":-19,"minute_vec_signal_avg":-48.0508474576,"minute_vec_signal_med":-49.0,"senal.suavizada":-47.9282057276,
"distancia_senal_promedio_f1":34.0997056409,"distancia_senal_mediana_f1":3.6319201783,
"distancia_senal_promedio_f2":7.9899189324,"distancia_senal_mediana_f2":0.5}]

@app.route('/', methods=['GET'])
def home():
    return '''<h1>RBPi Mock</h1>
<p>Mocking RBPI 2</p>'''

@app.route('/getData1', methods=['GET'])
def getData1():
    return jsonify(data1)

@app.route('/getData2', methods=['GET'])
def getData2():
    return jsonify(data2)

@app.route('/getData3', methods=['GET'])
def getData3():
    return jsonify(data3)

@app.route('/outputServerStub', methods=['POST'])
def outputServerStub():
    """mock server recieving trilateration output"""
    data = request.form
    print(data)
    return  jsonify(isError= False,
                    message= "Success",
                    statusCode= 200,
                    data= data), 200
app.run()