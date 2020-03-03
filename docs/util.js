

const copyToClipboard = str => {
    const el = document.createElement('textarea');
    el.value = str;
    document.body.appendChild(el);
    el.select();
    document.execCommand('copy');
    document.body.removeChild(el);
  };

button = document.getElementById('copyto');
console.log(button)

function whenClicked(e) {  
    elem = document.getElementById('tokenDiv');
    copyToClipboard(elem.value)
}

button.onclick = whenClicked