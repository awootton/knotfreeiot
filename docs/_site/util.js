

// import { nacl } from './nacl.js'; :-(

function KnotFreeTokenPayload() {
  this.exp = 1609372800;
  this.iss = "none";
  this.jti = "none";
  this.in = 32.0; // bytes per sec
  this.out = 32.0; // bytes per sec
  this.su = 4.0; // topics/addresses
  this.co = 2.0; // connections
  this.url = "knotfree.net";
}

function TokenRequest() {

  this.pkey = toHexString(new Uint8Array(pair.publicKey))
  this.payload = new KnotFreeTokenPayload()
  var d = new Date()
  this.comment = "My token"// + d.getFullYear()+"-"+ (1+d.getMonth()) + "-" + d.getDate()
}


const copyToClipboard = str => {
  const el = document.createElement('textarea');
  el.value = str;
  document.body.appendChild(el);
  el.select();
  document.execCommand('copy');
  document.body.removeChild(el);
};

button = document.getElementById('copyto');
button.disabled = true;

function whenClicked(e) {
  elem = document.getElementById('tokenDiv');
  copyToClipboard(elem.value)
}

button.onclick = whenClicked

var timerCountDown = 16
var myTimerVar
function myTimerFunc() {
  elem = document.getElementById('tokenDiv');
  elem.value = "just a moment ... " + timerCountDown
  timerCountDown--
  console.log("timer")
}

//var pair = nacl.sign.keyPair()
var pair = nacl.box.keyPair()

button3 = document.getElementById('gettoken');
console.log(button3)
function when3Clicked(e) {

  timerCountDown = 16
  myTimerVar = setInterval(myTimerFunc, 1000);

  //elem = document.getElementById('tokenDiv');
  //elem.value = "just a moment ..."
  myTimerFunc()

  // make the message 
  tr = new TokenRequest()
  var myJSON = JSON.stringify(tr);
  console.log(myJSON)

  // send to server
  var http = new XMLHttpRequest();
  http.onreadystatechange = function () {
    if (this.readyState == 4 && this.status == 200) {
      clearInterval(myTimerVar);
      //console.log( this.responseText );
      var obj = JSON.parse(this.responseText)
      nonce = unicodeStringToTypedArray(obj.nonce)
      tmp = toByteArray(obj.payload)
      box = new Uint8Array(tmp)
      tmp = toByteArray(obj.pkey)
      theirPublicKey = new Uint8Array(tmp)
      mySecretKey = pair.secretKey

      msg = nacl.box.open(box, nonce, theirPublicKey, mySecretKey)
      //strmsg = typedArrayToUnicodeString(msg)
      var strmsg = String.fromCharCode.apply(null, msg);
      console.log(strmsg)
      elem = document.getElementById('tokenDiv');
      elem.value = strmsg

      button = document.getElementById('copyto');
      button.disabled = false;

    }
  };
  var url = 'api1/getToken';
  http.open('POST', url, true);
  http.timeout = 25 * 1000;
  http.send(myJSON);
}
button3.onclick = when3Clicked


function toHexString(byteArray) {
  return Array.prototype.map.call(byteArray, function (byte) {
    return ('0' + (byte & 0xFF).toString(16)).slice(-2);
  }).join('');
}

function toByteArray(hexString) {
  var result = [];
  for (var i = 0; i < hexString.length; i += 2) {
    result.push(parseInt(hexString.substr(i, 2), 16));
  }
  return result;
}

function unicodeStringToTypedArray(s) {
  var escstr = encodeURIComponent(s);
  var binstr = escstr.replace(/%([0-9A-F]{2})/g, function (match, p1) {
    return String.fromCharCode('0x' + p1);
  });
  var ua = new Uint8Array(binstr.length);
  Array.prototype.forEach.call(binstr, function (ch, i) {
    ua[i] = ch.charCodeAt(0);
  });
  return ua;
}

function typedArrayToUnicodeString(ua) {
  var binstr = Array.prototype.map.call(ua, function (ch) {
    return String.fromCharCode(ch);
  }).join('');
  var escstr = binstr.replace(/(.)/g, function (m, p) {
    var code = p.charCodeAt(p).toString(16).toUpperCase();
    if (code.length < 2) {
      code = '0' + code;
    }
    return '%' + code;
  });
  return decodeURIComponent(escstr);
}



