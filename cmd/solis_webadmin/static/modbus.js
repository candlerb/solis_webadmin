function read_registers(register, count=1, functioncode=3) {
    const buffer = new ArrayBuffer(5);
    const view = new DataView(buffer);
    view.setUint8(0, functioncode);
    view.setUint16(1, register);
    view.setUint16(3, count);
    return modbus_request(buffer)
    .then(function(res) {
        view1 = new DataView(res);
        if (view1.getUint8(1) != count*2) {
            throw new ModbusError('Response length does not match count');
        }
        return Array(count).fill().map((e,i) => view1.getUint16(2+i*2));
    });
}

function write_register(register, value, functioncode=6) {
    const buffer = new ArrayBuffer(5);
    const view = new DataView(buffer);
    view.setUint8(0, functioncode);
    view.setUint16(1, register);
    view.setUint16(3, value);
    return modbus_request(buffer)
    .then(function(res) {
        view1 = new DataView(res);
        if (view1.getUint16(1) != register) {
            throw new ModbusError('Response: unexpected register');
        }
        if (view1.getUint16(3) != value) {
            throw new ModbusError('Response: unexpected value');
        }
        return value;
    });
}

function write_registers(register, values, functioncode=16) {
    const buffer = new ArrayBuffer(6 + values.length*2);
    const view = new DataView(buffer);
    view.setUint8(0, functioncode);
    view.setUint16(1, register);
    view.setUint16(3, values.length);
    view.setUint8(5, values.length*2);
    values.forEach((e,i) => view.setUint16(6+i*2, e));
    return modbus_request(buffer)
    .then(function(res) {
        view1 = new DataView(res);
        if (view1.getUint16(1) != register) {
            throw new ModbusError('Response: unexpected register');
        }
        if (view1.getUint16(3) != values.length) {
            throw new ModbusError('Response: unexpected length');
        }
        return values;
    });
}

function buf2hex(buffer) { // buffer is an ArrayBuffer
  return [...new Uint8Array(buffer)]
      .map(x => x.toString(16).padStart(2, '0'))
      .join(' ');
}

function ModbusError(message) {
  this.message = message;
  this.name = 'ModbusError';
}

function modbus_request(request, path='modbus') {
    const view0 = new DataView(request);
    console.log("=> " + buf2hex(request));
    return fetch(path, {
        'method': 'POST',
        headers: {
          'Content-Type': 'application/octet-stream'
        },
        body: request,
    })
    .then(function(response) {
        if (!response.ok) {
            throw new ModbusError('HTTP Error: '+response.status+' '+response.statusText);
        }
        return response.arrayBuffer();
    })
    .then(function (response) {
        if (!response) {
            throw new ModbusError('Empty response');
        }
        console.log("<= " + buf2hex(response));
        if (response.length < 2) {
            throw new ModbusError('Response too short');
        }
        const view1 = new DataView(response);
        if (view0.getUint8(0) == view1.getUint8(0)) {
            return response;
        }
        if (view0.getUint8(0) | 0x80 == view1.getUint8(0)) {
            throw new ModbusError('Error response: code ' + view1.getUint8(1));
        }
        throw new ModbusError('Response with wrong function code');
    })
}
