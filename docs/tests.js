


elem = document.getElementById('atwResult44');
console.log(elem)
elem.innerHTML = "ok then";

button = document.getElementById('sha256button');
console.log(button)

function whenClicked(e) {
    console.log("clicked")
    
    elem = document.getElementById('sha256input');
    console.log(elem.value)

    var sha256 = new jsSHA('SHA-256', 'TEXT');
    sha256.update(elem.value);
    var hash = sha256.getHash("B64");

    elem = document.getElementById('atwResult44');
    elem.innerHTML = "Result:  " + hash;
}

button.onclick = whenClicked
