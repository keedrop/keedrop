"use strict";

function transferEncode(value) {
  return window.nacl.util.encodeBase64(value).replace(/\//g, "!");
}

function transferDecode(value) {
  return window.nacl.util.decodeBase64(value.replace(/!/g, "/"));
}

function onStoreXhrLoadEnd(secretKey, resultBox, evt) {
  if (evt.target.status == 200) {
    var data = JSON.parse(evt.target.responseText);
    var decodeLink = location.protocol + "//" + location.host + "/r#" + data.mnemo + "_" + transferEncode(secretKey);
    resultBox.value = decodeLink;
    resultBox.parentNode.parentNode.classList.add("reveal");
  } else {
    // handle error
  }
}

var ERRORS = {
  "keyDecodeFail": "Could not interpret your key. Check that the link you clicked is complete.",
  "noSecretId": "It looks like someone tampered with your link.",
  "secretNotFound": "Could not retrieve secret. It has either expired (24h) or someone else already retrieved it.",
  "decryptionFailed": "Oops. Could not decrypt the data."
};

function showError(messageId, hideContent) {
  var errorDiv = document.querySelector(".error");
  errorDiv.innerHTML = ERRORS[messageId];
  errorDiv.classList.add("reveal");
  if (hideContent) {
    var content = document.querySelector(".content");
    content.classList.add("hide");
  }
}

function onEncryptClicked(secretInput, resultBox) {
  var secret = window.nacl.util.decodeUTF8(secretInput.value);
  var keyPair = window.nacl.box.keyPair();
  var nonce = window.nacl.randomBytes(window.nacl.box.nonceLength);
  var encrypted = window.nacl.box(secret, nonce, keyPair.publicKey, keyPair.secretKey);

  var req = new XMLHttpRequest();
  req.open("POST", "/api/secret");
  req.setRequestHeader("Content-Type", "application/json");
  req.addEventListener("loadend", onStoreXhrLoadEnd.bind(document, keyPair.secretKey, resultBox));
  req.send(JSON.stringify({
    pubkey: transferEncode(keyPair.publicKey),
    nonce: transferEncode(nonce),
    secret: transferEncode(encrypted)
  }));
}

function onReceiveXhrLoadEnd(decodedSecKey, resultBox, evt) {
  if (evt.target.status == 404) {
    showError("secretNotFound");
    return;
  }
  var data = JSON.parse(evt.target.responseText);
  var decodedSecret = transferDecode(data.secret);
  var decodedPubKey = transferDecode(data.pubkey);
  var decodedNonce = transferDecode(data.nonce);
  var plainText = window.nacl.box.open(decodedSecret, decodedNonce, decodedPubKey, decodedSecKey);
  if (plainText) {
    var stringValue = window.nacl.util.encodeUTF8(plainText);
    resultBox.value = stringValue;
    resultBox.parentNode.parentNode.classList.add("reveal");
  } else {
    showError("decryptionFailed");
  } 
}

function onDecryptClicked(resultBox) {
  if (location.hash.indexOf("_") == -1) {
    showError("noSecretdId");
    return;
  }

  var components = location.hash.substr(1).split("_");
  var mnemo = components[0];
  var secretKey = components[1];

  try {
    var decodedSecretKey = transferDecode(secretKey);
    if (decodedSecretKey.length != 32) {
      throw new Error("Key length mismatch");
    }
    var req = new XMLHttpRequest();
    req.open("GET", "/api/secret/" + mnemo);
    req.setRequestHeader("Accept", "application/json");
    req.addEventListener("loadend", onReceiveXhrLoadEnd.bind(document, decodedSecretKey, resultBox));
    req.send();
  } catch(e) {
    showError("keyDecodeFail", true);
  }
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
};
