function onStoreXhrLoadEnd(keyPair, nonce, resultBox, evt) {
  if (evt.target.status == 204) {
    var decodeLink = location.protocol + "//" + location.host + "/r#" + nacl.util.encodeBase64(keyPair.publicKey) + "_" + nacl.util.encodeBase64(nonce) + "_" + nacl.util.encodeBase64(keyPair.secretKey);
    resultBox.innerHTML = "Send this link to the intended recipient: " + decodeLink;
  } else {
    // handle error
  }
}

function onEncryptClicked(secretInput, resultBox) {
  var secret = nacl.util.decodeUTF8(secretInput.value);
  var keyPair = nacl.box.keyPair();
  var nonce = nacl.randomBytes(nacl.box.nonceLength);
  var encrypted = nacl.box(secret, nonce, keyPair.publicKey, keyPair.secretKey);

  var req = new XMLHttpRequest();
  req.open("POST", "/api/store");
  req.setRequestHeader("Content-Type", "application/json");
  req.addEventListener("loadend", onStoreXhrLoadEnd.bind(document, keyPair, nonce, resultBox))
  req.send(JSON.stringify({
    pubkey: nacl.util.encodeBase64(keyPair.publicKey),
    nonce: nacl.util.encodeBase64(nonce),
    secret: nacl.util.encodeBase64(encrypted)
  }));
}

function onReceiveXhrLoadEnd(pubKey, nonce, secretKey, resultBox, event) {
  if (event.target.status == 404) {
    alert("No such key. It has either expired or someone else already retrieved it.");
    return;
  }
  var data = JSON.parse(event.target.responseText);
  var decodedSecret = nacl.util.decodeBase64(data.secret);
  var decodedPubKey = nacl.util.decodeBase64(pubKey);
  var decodedNonce = nacl.util.decodeBase64(nonce);
  var decodedSecKey = nacl.util.decodeBase64(secretKey);
  var plainText = nacl.box.open(decodedSecret, decodedNonce, decodedPubKey, decodedSecKey);
  if (plainText == false) {
    alert("Could not decrypt the data");
  } else {
    var stringValue = nacl.util.encodeUTF8(plainText);
    resultBox.innerHTML = "Here is your password: " + stringValue;
  }
}

function onDecryptClicked(resultBox) {
  if (location.hash.indexOf("_") == -1) {
    alert("Oops, that didn't work");
    return;
  }
  var components = location.hash.substr(1).split("_");
  var pubKey = components[0];
  var nonce = components[1]
  var secretKey = components[2];

  var req = new XMLHttpRequest();
  req.open("GET", "/api/retrieve/" + encodeURIComponent(pubKey) + "_" + encodeURIComponent(nonce));
  req.setRequestHeader("Accept", "application/json");
  req.addEventListener("loadend", onReceiveXhrLoadEnd.bind(document, pubKey, nonce, secretKey, resultBox))
  req.send();
}

function initReadPage() {
  var decodeButton = document.getElementById("decrypt");
  var resultBox = document.getElementById("resultBox");
  decodeButton.onclick = onDecryptClicked.bind(document, resultBox);
}

function initStorePage() {
  var secretInput = document.getElementById("secret");
  var resultBox = document.getElementById("resultBox");
  var encryptBtn = document.getElementById("encrypt");
  encryptBtn.onclick = onEncryptClicked.bind(document, secretInput, resultBox);
}


window.onload = function() {
  if (document.getElementById("encrypt") != null) {
    initStorePage();
  } else if (document.getElementById("decrypt") != null) {
    initReadPage();
  }
}
