"use strict";

function transferEncode(value) {
  return nacl.util.encodeBase64(value).replace(/\//g, "!")
}

function transferDecode(value) {
  return nacl.util.decodeBase64(value.replace(/!/g, "/"))
}

function onStoreXhrLoadEnd(secretKey, resultBox, evt) {
  if (evt.target.status == 200) {
    var data = JSON.parse(evt.target.responseText);
    var decodeLink = location.protocol + "//" + location.host + "/r#" + data.mnemo + "_" + transferEncode(secretKey);
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
  req.open("POST", "/api/secret");
  req.setRequestHeader("Content-Type", "application/json");
  req.addEventListener("loadend", onStoreXhrLoadEnd.bind(document, keyPair.secretKey, resultBox))
  req.send(JSON.stringify({
    pubkey: transferEncode(keyPair.publicKey),
    nonce: transferEncode(nonce),
    secret: transferEncode(encrypted)
  }));
}

function onReceiveXhrLoadEnd(secretKey, resultBox, evt) {
  if (evt.target.status == 404) {
    alert("No such key. It has either expired or someone else already retrieved it.");
    return;
  }
  var data = JSON.parse(evt.target.responseText);
  var decodedSecret = transferDecode(data.secret);
  var decodedPubKey = transferDecode(data.pubkey);
  var decodedNonce = transferDecode(data.nonce);
  var decodedSecKey = transferDecode(secretKey);
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
  var mnemo = components[0];
  var secretKey = components[1];

  var req = new XMLHttpRequest();
  req.open("GET", "/api/secret/" + mnemo);
  req.setRequestHeader("Accept", "application/json");
  req.addEventListener("loadend", onReceiveXhrLoadEnd.bind(document, secretKey, resultBox))
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
