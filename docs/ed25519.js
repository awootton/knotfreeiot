

function toHexString(byteArray) {
    return Array.prototype.map.call(byteArray, function(byte) {
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

const copyToClipboard2 = str => {
    const el = document.createElement('textarea');
    el.value = str;
    document.body.appendChild(el);
    el.select();
    document.execCommand('copy');
    document.body.removeChild(el);
  };


cpmments = document.getElementById('commentsDiv');
var d = new Date()
cpmments.value = "My new key pair " + d.getFullYear()+"-"+ (1+d.getMonth()) + "-" + d.getDate()

function whenCreateClicked(e) {
    console.log("clicked")  

    pair = nacl.sign.keyPair()

    //console.log(toHexString(new Uint8Array(pair.publicKey)))

    //console.log(toHexString(new Uint8Array(pair.secretKey)))

    user = document.getElementById('usernameDiv');
    user.value = toHexString(new Uint8Array(pair.publicKey))

    pass = document.getElementById('passwordDiv');
    pass.value = toHexString(new Uint8Array(pair.secretKey))
}
createButton = document.getElementById('createKeyPair');
console.log(createButton)
createButton.onclick = whenCreateClicked

copyButton = document.getElementById('copyto');
console.log(copyButton)
function whenCopyClicked(e) {  
    console.log("copy clicked")
    elem1 = document.getElementById('usernameDiv');
    elem2 = document.getElementById('passwordDiv');
    elem3 = document.getElementById('commentsDiv');
    val = "%%user:" + elem1.value + "%\n%pass:" + elem2.value + "%\n%" + elem3.value + "%%"
    copyToClipboard2(val)
}
copyButton.onclick = whenCopyClicked