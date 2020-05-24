---
---
"use strict";

function defaultAdapter(baseUrl) {
  function request(url, data, callback) {
    var req = new XMLHttpRequest();
    req.open(data ? "POST" : "GET", url);
    req.onreadystatechange = function() {
      if (4 === req.readyState) {
        if (req.status < 200 || req.status >= 400) {
          callback(new Error("Could not send encrypted data to server"), "");
        } else {
          callback(undefined, JSON.parse(req.responseText));
        }
      }
    };
    if (data) {
      req.setRequestHeader("Content-Type", "application/json");
      req.send(data);
    } else {
      req.send();
    }
  }

  return {
    create: function(data, callback) {
      request(baseUrl, JSON.stringify(data), callback);
    },
    get: function(mnemo, callback) {
      request(baseUrl + mnemo, undefined, callback);
    }
  };
}

window.api = defaultAdapter("{{site.env.KEEDROP_API_PREFIX}}https://keedrop.com/api/secret/");

function transferEncode(value) {
  return window.nacl.util.encodeBase64(value).replace(/\//g, "!");
}

function transferDecode(value) {
  return window.nacl.util.decodeBase64(value.replace(/!/g, "/"));
}

var ERRORS = {
  "keyDecodeFail": "Could not interpret your key. Check that the link you clicked is complete.",
  "noSecretId": "It looks like someone tampered with your link.",
  "secretNotFound": "Could not retrieve secret. It has either expired (24h) or someone else already retrieved it.",
  "decryptionFailed": "Oops. Could not decrypt the data.",
  "sendFail": "Could not connect to server to store encrypted data."
};

function copyToClipboard(source, callback) {
  function fallback() {
    try {
      source.select();
      source.setSelectionRange(0, 9999);
      callback(document.execCommand("copy"));
    } catch (e) {
      callback(false);
    }
  }
  // Only works on HTTPS served sites
  if (!navigator.clipboard) {
    return fallback();
  }
  navigator.clipboard.writeText(source.value)
    .then(callback.bind(this, true), callback.bind(this, false));
}

function onCopyClick(event) {
  event.preventDefault && event.preventDefault();
  copyToClipboard(document.getElementById("resultBox"), function(success) {
    if (success) {
      event.srcElement.innerText = "Copied";
    }
  });
  return false;
}

function hideError() {
  var errorDiv = document.querySelector(".error");
  errorDiv.classList.remove("reveal");
}

function showError(messageId, hideContent) {
  var errorDiv = document.querySelector(".error");
  errorDiv.innerHTML = ERRORS[messageId] || messageId;
  errorDiv.classList.add("reveal");
  if (hideContent) {
    var content = document.querySelector(".content");
    content.classList.add("hide");
  }
}

function showResult(result) {
  var resultBox = document.getElementById("resultBox");
  resultBox.value = result;
  resultBox.parentNode.parentNode.parentNode.classList.add("reveal");
  resultBox.focus();
  resultBox.select();
  // For mobile devices
  resultBox.setSelectionRange(0, 9999);
}

function onEncryptSubmit(event) {
  // Cancel default form submit handling
  event.preventDefault && event.preventDefault();

  var form = event.currentTarget;
  var secret = window.nacl.util.decodeUTF8(form.elements.secret.value);
  var keyPair = window.nacl.box.keyPair();
  var nonce = window.nacl.randomBytes(window.nacl.box.nonceLength);
  var encrypted = window.nacl.box(secret, nonce, keyPair.publicKey, keyPair.secretKey);

  hideError();

  var resultBox = document.getElementById("resultBox");
  resultBox.value = "";
  var button = form.querySelector("button");
  button.disabled = true;

  window.api.create({
    pubkey: transferEncode(keyPair.publicKey),
    nonce: transferEncode(nonce),
    secret: transferEncode(encrypted)
  }, function(error, result) {
    button.disabled = false;
    if (error) {
      return showError("sendFail");
    }
    var decodeLink = location.protocol + "//" + location.host + "/r#" + result.mnemo + "_" + transferEncode(keyPair.secretKey);
    showResult(decodeLink);
  });
  return false;
}

function onDecryptClicked() {
  if (location.hash.indexOf("_") == -1) {
    return showError("noSecretId");
  }

  var components = location.hash.substr(1).split("_");
  var mnemo = components[0];
  var secretKey = components[1];

  try {
    var decodedSecretKey = transferDecode(secretKey);
    if (decodedSecretKey.length != 32) {
      throw new Error("Key length mismatch");
    }
    window.api.get(mnemo, function(error, result) {
      if (error) {
        return showError("keyDecodeFail", true);
      }
      var plainText = window.nacl.box.open(
        transferDecode(result.secret),
        transferDecode(result.nonce),
        transferDecode(result.pubkey),
        decodedSecretKey);
      if (plainText) {
        showResult(window.nacl.util.encodeUTF8(plainText));
      } else {
        showError("decryptionFailed");
      }
    });
  } catch(e) {
    showError("keyDecodeFail", true);
  }
}

function initReadPage() {
  var button = document.querySelector("button:disabled");
  button.disabled = false;
  button.onclick = onDecryptClicked;
}

function initStorePage() {
  var button = document.querySelector("button:disabled");
  button.disabled = false;
  var form = document.getElementById("storeForm");
  form.onsubmit = onEncryptSubmit;
}

window.addEventListener("click", function(event) {
  var source = event.srcElement;
  if (source.id === "copy") {
    return onCopyClick(event);
  }
});

window.addEventListener("DOMContentLoaded", function() {
  if (document.getElementById("encrypt") != null) {
    initStorePage();
  } else if (document.getElementById("decrypt") != null) {
    initReadPage();
  }
});
